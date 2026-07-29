package controller

import (
	"context"
	"errors"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	vedro "github.com/svetoch-dev/vedro/api/v1alpha1"
	"github.com/svetoch-dev/vedro/internal/cloud"
	"github.com/svetoch-dev/vedro/internal/conditions"
)

var _ = Describe("BucketAccessReconciler", func() {
	var (
		reconciler   *BucketAccessReconciler
		provider     *fakeProvider
		bucketAccess *fakeBucketAccess
	)

	BeforeEach(func() {
		bucketAccess = &fakeBucketAccess{
			ensureResult: &cloud.BucketAccessAttrs{
				BucketName:    "external-bucket",
				BucketId:      "external-bucket-id",
				PrincipalId:   "principal-id",
				GrantedAccess: vedro.ObjectReader,
			},
		}
		provider = &fakeProvider{
			capabilities: cloud.Capabilities{
				BucketAccess: cloud.BucketAccessCapabilities{
					ObjectReader: true,
					ObjectWriter: true,
					ObjectAdmin:  true,
					BucketAdmin:  true,
				},
			},
			bucketAccess: bucketAccess,
		}
		reconciler = &BucketAccessReconciler{
			Client: k8sClient,
			Scheme: k8sClient.Scheme(),
			ProviderFactory: func(
				context.Context,
				vedro.ProviderConfig,
				client.Client,
			) (cloud.Provider, error) {
				return provider, nil
			},
		}
	})

	It("ignores missing BucketAccess resources", func() {
		result, err := reconciler.Reconcile(ctx, reconcile.Request{
			NamespacedName: types.NamespacedName{
				Name:      "missing-access",
				Namespace: "default",
			},
		})

		Expect(err).NotTo(HaveOccurred())
		Expect(result).To(Equal(reconcile.Result{}))
		Expect(bucketAccess.ensureCalls).To(BeZero())
	})

	It("adds the finalizer and reports a missing Bucket dependency", func() {
		access := createBucketAccess(ctx, "missing-bucket", "missing", "principal")

		result, err := reconcileBucketAccess(ctx, reconciler, access)

		Expect(err).NotTo(HaveOccurred())
		Expect(result).To(Equal(reconcile.Result{}))
		fetched := getBucketAccess(ctx, client.ObjectKeyFromObject(access))
		Expect(fetched.Finalizers).To(ContainElement(bucketAccessFinalizer))
		Expect(fetched.Status.ObservedGeneration).To(Equal(fetched.Generation))
		expectAccessCondition(
			fetched,
			conditions.TypeReady,
			metav1.ConditionFalse,
			conditions.ReasonBucketNotFound,
		)
		expectAccessCondition(
			fetched,
			conditions.TypeBucketReady,
			metav1.ConditionFalse,
			conditions.ReasonBucketNotFound,
		)
		Expect(bucketAccess.ensureCalls).To(BeZero())
	})

	It("waits for dependencies to become ready", func() {
		createBucket(ctx, "bucket")
		principal := createCloudPrincipal(ctx, "principal")
		markCloudPrincipalReady(ctx, principal)
		access := createBucketAccess(ctx, "dependencies-not-ready", "bucket", "principal")

		result, err := reconcileBucketAccess(ctx, reconciler, access)

		Expect(err).NotTo(HaveOccurred())
		Expect(result).To(Equal(reconcile.Result{}))
		fetched := getBucketAccess(ctx, client.ObjectKeyFromObject(access))
		expectAccessCondition(
			fetched,
			conditions.TypeReady,
			metav1.ConditionFalse,
			conditions.ReasonBucketAccessDependencyNotReady,
		)
		expectAccessCondition(
			fetched,
			conditions.TypeBucketReady,
			metav1.ConditionFalse,
			conditions.ReasonNoConditions,
		)
		Expect(bucketAccess.ensureCalls).To(BeZero())
	})

	It("rejects dependencies that use different ProviderConfigs", func() {
		bucket := createBucket(ctx, "bucket")
		principal := createCloudPrincipal(ctx, "principal", func(p *vedro.CloudPrincipal) {
			p.Spec.ProviderRef.Name = "other-provider"
		})
		markBucketReady(ctx, bucket)
		markCloudPrincipalReady(ctx, principal)
		access := createBucketAccess(ctx, "provider-mismatch", "bucket", "principal")

		result, err := reconcileBucketAccess(ctx, reconciler, access)

		Expect(err).NotTo(HaveOccurred())
		Expect(result).To(Equal(reconcile.Result{}))
		fetched := getBucketAccess(ctx, client.ObjectKeyFromObject(access))
		expectAccessCondition(
			fetched,
			conditions.TypeReady,
			metav1.ConditionFalse,
			conditions.ReasonBucketAccessProviderConfigMissMatch,
		)
		Expect(bucketAccess.ensureCalls).To(BeZero())
	})

	It("records unsupported access levels without calling the provider", func() {
		createProviderConfig(ctx)
		bucket := createBucket(ctx, "bucket")
		principal := createCloudPrincipal(ctx, "principal")
		markBucketReady(ctx, bucket)
		markCloudPrincipalReady(ctx, principal)
		access := createBucketAccess(ctx, "unsupported-access", "bucket", "principal")
		provider.capabilities.BucketAccess.ObjectReader = false

		result, err := reconcileBucketAccess(ctx, reconciler, access)

		Expect(err).NotTo(HaveOccurred())
		Expect(result).To(Equal(reconcile.Result{}))
		fetched := getBucketAccess(ctx, client.ObjectKeyFromObject(access))
		Expect(fetched.Status.UnsupportedFeatures).To(HaveLen(1))
		Expect(fetched.Status.UnsupportedFeatures[0].Reason).To(
			Equal(vedro.BucketAccessUnsupportedObjectReader),
		)
		expectAccessCondition(
			fetched,
			conditions.TypeReady,
			metav1.ConditionFalse,
			conditions.ReasonBucketAccessUnsupportedFeatures,
		)
		Expect(bucketAccess.ensureCalls).To(BeZero())
		Expect(provider.cleanupCalled).To(BeTrue())
	})

	It("sets successful status after ensuring access", func() {
		createProviderConfig(ctx)
		bucket := createBucket(ctx, "bucket")
		principal := createCloudPrincipal(ctx, "principal")
		markBucketReady(ctx, bucket)
		markCloudPrincipalReady(ctx, principal)
		access := createBucketAccess(ctx, "successful-reconcile", "bucket", "principal")

		result, err := reconcileBucketAccess(ctx, reconciler, access)

		Expect(err).NotTo(HaveOccurred())
		Expect(result).To(Equal(reconcile.Result{}))
		Expect(bucketAccess.ensureCalls).To(Equal(1))
		Expect(bucketAccess.lastBucket.Name).To(Equal("bucket"))
		Expect(bucketAccess.lastPrincipal.Name).To(Equal("principal"))
		Expect(bucketAccess.lastAccess.Name).To(Equal("successful-reconcile"))
		Expect(provider.cleanupCalled).To(BeTrue())

		fetched := getBucketAccess(ctx, client.ObjectKeyFromObject(access))
		Expect(fetched.Status.ObservedGeneration).To(Equal(fetched.Generation))
		Expect(fetched.Status.ObservedProvider).To(Equal("test-provider"))
		Expect(fetched.Status.Applied).To(Equal(
			(*vedro.BucketAccessProperties)(bucketAccess.ensureResult),
		))
		expectAccessCondition(
			fetched,
			conditions.TypeReady,
			metav1.ConditionTrue,
			conditions.ReasonBucketAccessReconciled,
		)
		expectAccessCondition(
			fetched,
			conditions.TypeBucketReady,
			metav1.ConditionTrue,
			conditions.ReasonBucketReady,
		)
		expectAccessCondition(
			fetched,
			conditions.TypeCloudPrincipalReady,
			metav1.ConditionTrue,
			conditions.ReasonCloudPrincipalReady,
		)
	})

	It("records errors returned while ensuring access", func() {
		createProviderConfig(ctx)
		bucket := createBucket(ctx, "bucket")
		principal := createCloudPrincipal(ctx, "principal")
		markBucketReady(ctx, bucket)
		markCloudPrincipalReady(ctx, principal)
		access := createBucketAccess(ctx, "ensure-error", "bucket", "principal")
		bucketAccess.ensureErr = errors.New("ensure access failed")

		result, err := reconcileBucketAccess(ctx, reconciler, access)

		Expect(err).To(MatchError("ensure access failed"))
		Expect(result).To(Equal(reconcile.Result{}))
		Expect(bucketAccess.ensureCalls).To(Equal(1))
		Expect(provider.cleanupCalled).To(BeTrue())
		fetched := getBucketAccess(ctx, client.ObjectKeyFromObject(access))
		expectAccessCondition(
			fetched,
			conditions.TypeReady,
			metav1.ConditionFalse,
			conditions.ReasonBucketAccessEnsureError,
		)
	})

	It("deletes external access and removes the finalizer", func() {
		createProviderConfig(ctx)
		access := createAppliedBucketAccess(ctx, "delete-access")
		addBucketAccessFinalizer(ctx, access)
		Expect(k8sClient.Delete(ctx, access)).To(Succeed())

		result, err := reconcileBucketAccess(ctx, reconciler, access)

		Expect(err).NotTo(HaveOccurred())
		Expect(result).To(Equal(reconcile.Result{}))
		Expect(bucketAccess.deleteCalls).To(Equal(1))
		Expect(bucketAccess.lastAccess.Status.Applied).NotTo(BeNil())
		Expect(provider.cleanupCalled).To(BeTrue())
		expectBucketAccessNotFound(ctx, client.ObjectKeyFromObject(access))
	})

	It("records deletion errors and keeps the finalizer", func() {
		createProviderConfig(ctx)
		access := createAppliedBucketAccess(ctx, "delete-error")
		bucketAccess.deleteErr = errors.New("delete access failed")
		addBucketAccessFinalizer(ctx, access)
		Expect(k8sClient.Delete(ctx, access)).To(Succeed())

		_, err := reconcileBucketAccess(ctx, reconciler, access)

		Expect(err).To(MatchError("delete access failed"))
		Expect(bucketAccess.deleteCalls).To(Equal(1))
		Expect(provider.cleanupCalled).To(BeTrue())
		fetched := getBucketAccess(ctx, client.ObjectKeyFromObject(access))
		Expect(fetched.Finalizers).To(ContainElement(bucketAccessFinalizer))
		expectAccessCondition(
			fetched,
			conditions.TypeReady,
			metav1.ConditionFalse,
			conditions.ReasonBucketAccessDeleteError,
		)
	})
})

type fakeBucketAccess struct {
	ensureResult *cloud.BucketAccessAttrs
	ensureErr    error
	deleteErr    error
	ensureCalls  int
	deleteCalls  int

	lastBucket    vedro.Bucket
	lastPrincipal vedro.CloudPrincipal
	lastAccess    vedro.BucketAccess
}

func (b *fakeBucketAccess) EnsureBucketAccess(
	ctx context.Context,
	bucket vedro.Bucket,
	principal vedro.CloudPrincipal,
	access vedro.BucketAccess,
) (*cloud.BucketAccessAttrs, error) {
	b.ensureCalls++
	b.lastBucket = bucket
	b.lastPrincipal = principal
	b.lastAccess = access
	return b.ensureResult, b.ensureErr
}

func (b *fakeBucketAccess) DeleteBucketAccess(
	ctx context.Context,
	access vedro.BucketAccess,
) error {
	b.deleteCalls++
	b.lastAccess = access
	return b.deleteErr
}

func createBucketAccess(
	ctx context.Context,
	name string,
	bucketName string,
	principalName string,
) *vedro.BucketAccess {
	access := &vedro.BucketAccess{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "vedro.svetoch.dev/v1alpha1",
			Kind:       "BucketAccess",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: "default",
		},
		Spec: vedro.BucketAccessSpec{
			BucketRef: vedro.BucketReference{
				Name:      bucketName,
				Namespace: "default",
			},
			PrincipalRef: vedro.PrincipalReference{
				Name:      principalName,
				Namespace: "default",
			},
			Access: vedro.Access{Level: vedro.ObjectReader},
		},
	}

	Expect(k8sClient.Create(ctx, access)).To(Succeed())
	DeferCleanup(func() {
		cleanupBucketAccess(ctx, client.ObjectKeyFromObject(access))
	})
	return access
}

func createAppliedBucketAccess(ctx context.Context, name string) *vedro.BucketAccess {
	access := createBucketAccess(ctx, name, "bucket", "principal")
	fetched := getBucketAccess(ctx, client.ObjectKeyFromObject(access))
	fetched.Status.ObservedProvider = "test-provider"
	fetched.Status.Applied = &vedro.BucketAccessProperties{
		BucketName:    "external-bucket",
		BucketId:      "external-bucket-id",
		PrincipalId:   "principal-id",
		GrantedAccess: vedro.ObjectReader,
	}
	Expect(k8sClient.Status().Update(ctx, fetched)).To(Succeed())
	return access
}

func reconcileBucketAccess(
	ctx context.Context,
	reconciler *BucketAccessReconciler,
	access *vedro.BucketAccess,
) (reconcile.Result, error) {
	return reconciler.Reconcile(ctx, reconcile.Request{
		NamespacedName: client.ObjectKeyFromObject(access),
	})
}

func markBucketReady(ctx context.Context, bucket *vedro.Bucket) {
	fetched := getBucket(ctx, client.ObjectKeyFromObject(bucket))
	fetched.Status.ObservedGeneration = fetched.Generation
	fetched.Status.ExternalName = "external-bucket"
	fetched.Status.ExternalId = "external-bucket-id"
	meta.SetStatusCondition(&fetched.Status.Conditions, metav1.Condition{
		Type:               conditions.TypeReady,
		Status:             metav1.ConditionTrue,
		ObservedGeneration: fetched.Generation,
		Reason:             conditions.ReasonBucketReconciled,
		Message:            "Bucket Reconciled",
	})
	Expect(k8sClient.Status().Update(ctx, fetched)).To(Succeed())
}

func markCloudPrincipalReady(ctx context.Context, principal *vedro.CloudPrincipal) {
	fetched := getCloudPrincipal(ctx, client.ObjectKeyFromObject(principal))
	fetched.Status.ObservedGeneration = fetched.Generation
	fetched.Status.ExternalName = "external-principal"
	fetched.Status.ExternalId = "principal-id"
	meta.SetStatusCondition(&fetched.Status.Conditions, metav1.Condition{
		Type:               conditions.TypeReady,
		Status:             metav1.ConditionTrue,
		ObservedGeneration: fetched.Generation,
		Reason:             conditions.ReasonCloudPrincipalReconciled,
		Message:            "CloudPrincipal Reconciled",
	})
	Expect(k8sClient.Status().Update(ctx, fetched)).To(Succeed())
}

func getBucketAccess(ctx context.Context, key client.ObjectKey) *vedro.BucketAccess {
	access := &vedro.BucketAccess{}
	Expect(k8sClient.Get(ctx, key, access)).To(Succeed())
	return access
}

func expectAccessCondition(
	access *vedro.BucketAccess,
	conditionType string,
	status metav1.ConditionStatus,
	reason string,
) {
	condition := meta.FindStatusCondition(access.Status.Conditions, conditionType)
	Expect(condition).NotTo(BeNil())
	Expect(condition.Status).To(Equal(status))
	Expect(condition.Reason).To(Equal(reason))
}

func addBucketAccessFinalizer(ctx context.Context, access *vedro.BucketAccess) {
	fetched := getBucketAccess(ctx, client.ObjectKeyFromObject(access))
	fetched.Finalizers = append(fetched.Finalizers, bucketAccessFinalizer)
	Expect(k8sClient.Update(ctx, fetched)).To(Succeed())
}

func expectBucketAccessNotFound(ctx context.Context, key client.ObjectKey) {
	err := k8sClient.Get(ctx, key, &vedro.BucketAccess{})
	Expect(apierrors.IsNotFound(err)).To(BeTrue())
}

func cleanupBucketAccess(ctx context.Context, key client.ObjectKey) {
	access := &vedro.BucketAccess{}
	err := k8sClient.Get(ctx, key, access)
	if apierrors.IsNotFound(err) {
		return
	}
	Expect(err).NotTo(HaveOccurred())

	access.Finalizers = nil
	Expect(k8sClient.Update(ctx, access)).To(Succeed())
	err = k8sClient.Delete(ctx, access)
	if err != nil {
		Expect(apierrors.IsNotFound(err)).To(BeTrue())
	}
}
