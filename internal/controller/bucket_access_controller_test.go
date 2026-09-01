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
		access := createBucketAccess(ctx, "missing-bucket", "missing")

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
		access := createBucketAccess(ctx, "dependencies-not-ready", "bucket")

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
		access := createBucketAccess(ctx, "provider-mismatch", "bucket")

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
		access := createBucketAccess(ctx, "unsupported-access", "bucket")
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
		access := createBucketAccess(ctx, "successful-reconcile", "bucket")

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
		access := createBucketAccess(ctx, "ensure-error", "bucket")
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

	It("cascades BucketAccess deletion when its Bucket is deleted", func() {
		createProviderConfig(ctx)
		bucket := createBucket(ctx, "bucket")
		createCloudPrincipal(ctx, "principal")
		access := createAppliedBucketAccess(ctx, "cascade-from-bucket")
		addBucketAccessFinalizer(ctx, access)
		controllerutilAddFinalizer(ctx, bucket)
		Expect(k8sClient.Delete(ctx, bucket)).To(Succeed())

		bucketReconciler := &BucketReconciler{
			Client:          k8sClient,
			Scheme:          k8sClient.Scheme(),
			ProviderFactory: reconciler.ProviderFactory,
		}
		result, err := bucketReconciler.Reconcile(ctx, reconcile.Request{
			NamespacedName: client.ObjectKeyFromObject(bucket),
		})
		Expect(err).NotTo(HaveOccurred())
		Expect(result.RequeueAfter).NotTo(BeZero())

		result, err = reconcileBucketAccess(ctx, reconciler, access)
		Expect(err).NotTo(HaveOccurred())
		Expect(result).To(Equal(reconcile.Result{}))
		terminatingAccess := getBucketAccess(ctx, client.ObjectKeyFromObject(access))
		Expect(terminatingAccess.DeletionTimestamp.IsZero()).To(BeFalse())
		Expect(bucketAccess.deleteCalls).To(BeZero())

		result, err = reconcileBucketAccess(ctx, reconciler, access)
		Expect(err).NotTo(HaveOccurred())
		Expect(result).To(Equal(reconcile.Result{}))
		Expect(bucketAccess.deleteCalls).To(Equal(1))
		Expect(provider.cleanupCalled).To(BeTrue())
		expectBucketAccessNotFound(ctx, client.ObjectKeyFromObject(access))

		result, err = bucketReconciler.Reconcile(ctx, reconcile.Request{
			NamespacedName: client.ObjectKeyFromObject(bucket),
		})
		Expect(err).NotTo(HaveOccurred())
		Expect(result).To(Equal(reconcile.Result{}))
		err = k8sClient.Get(ctx, client.ObjectKeyFromObject(bucket), &vedro.Bucket{})
		Expect(apierrors.IsNotFound(err)).To(BeTrue())
	})

	It("cascades BucketAccess deletion when its CloudPrincipal is deleted", func() {
		createProviderConfig(ctx)
		createBucket(ctx, "bucket")
		principal := createCloudPrincipal(ctx, "principal")
		access := createAppliedBucketAccess(ctx, "cascade-from-principal")
		addBucketAccessFinalizer(ctx, access)
		addPrincipalFinalizer(ctx, principal)
		Expect(k8sClient.Delete(ctx, principal)).To(Succeed())

		principalReconciler := &CloudPrincipalReconciler{
			Client:          k8sClient,
			Scheme:          k8sClient.Scheme(),
			ProviderFactory: reconciler.ProviderFactory,
		}
		result, err := reconcileCloudPrincipal(ctx, principalReconciler, principal)
		Expect(err).NotTo(HaveOccurred())
		Expect(result.RequeueAfter).NotTo(BeZero())

		result, err = reconcileBucketAccess(ctx, reconciler, access)
		Expect(err).NotTo(HaveOccurred())
		Expect(result).To(Equal(reconcile.Result{}))
		terminatingAccess := getBucketAccess(ctx, client.ObjectKeyFromObject(access))
		Expect(terminatingAccess.DeletionTimestamp.IsZero()).To(BeFalse())
		Expect(bucketAccess.deleteCalls).To(BeZero())

		result, err = reconcileBucketAccess(ctx, reconciler, access)
		Expect(err).NotTo(HaveOccurred())
		Expect(result).To(Equal(reconcile.Result{}))
		Expect(bucketAccess.deleteCalls).To(Equal(1))
		Expect(provider.cleanupCalled).To(BeTrue())
		expectBucketAccessNotFound(ctx, client.ObjectKeyFromObject(access))

		result, err = reconcileCloudPrincipal(ctx, principalReconciler, principal)
		Expect(err).NotTo(HaveOccurred())
		Expect(result).To(Equal(reconcile.Result{}))
		expectCloudPrincipalNotFound(ctx, client.ObjectKeyFromObject(principal))
	})

	It("keeps the parent and BucketAccess finalizers when cascaded access deletion fails", func() {
		createProviderConfig(ctx)
		bucket := createBucket(ctx, "bucket")
		createCloudPrincipal(ctx, "principal")
		access := createAppliedBucketAccess(ctx, "cascade-delete-error")
		addBucketAccessFinalizer(ctx, access)
		controllerutilAddFinalizer(ctx, bucket)
		Expect(k8sClient.Delete(ctx, bucket)).To(Succeed())

		_, err := reconcileBucketAccess(ctx, reconciler, access)
		Expect(err).NotTo(HaveOccurred())

		bucketAccess.deleteErr = errors.New("delete access failed")
		_, err = reconcileBucketAccess(ctx, reconciler, access)
		Expect(err).To(MatchError("delete access failed"))
		fetchedAccess := getBucketAccess(ctx, client.ObjectKeyFromObject(access))
		Expect(fetchedAccess.Finalizers).To(ContainElement(bucketAccessFinalizer))

		bucketReconciler := &BucketReconciler{
			Client:          k8sClient,
			Scheme:          k8sClient.Scheme(),
			ProviderFactory: reconciler.ProviderFactory,
		}
		result, err := bucketReconciler.Reconcile(ctx, reconcile.Request{
			NamespacedName: client.ObjectKeyFromObject(bucket),
		})
		Expect(err).NotTo(HaveOccurred())
		Expect(result.RequeueAfter).NotTo(BeZero())
		fetchedBucket := getBucket(ctx, client.ObjectKeyFromObject(bucket))
		Expect(fetchedBucket.Finalizers).To(ContainElement(bucketFinalizer))
	})

	It("waits for every referenced BucketAccess before deleting a Bucket", func() {
		createProviderConfig(ctx)
		bucket := createBucket(ctx, "bucket")
		createCloudPrincipal(ctx, "principal")
		firstAccess := createAppliedBucketAccess(ctx, "cascade-multiple-first")
		secondAccess := createAppliedBucketAccess(ctx, "cascade-multiple-second")
		addBucketAccessFinalizer(ctx, firstAccess)
		addBucketAccessFinalizer(ctx, secondAccess)
		controllerutilAddFinalizer(ctx, bucket)
		Expect(k8sClient.Delete(ctx, bucket)).To(Succeed())

		_, err := reconcileBucketAccess(ctx, reconciler, firstAccess)
		Expect(err).NotTo(HaveOccurred())
		_, err = reconcileBucketAccess(ctx, reconciler, firstAccess)
		Expect(err).NotTo(HaveOccurred())
		expectBucketAccessNotFound(ctx, client.ObjectKeyFromObject(firstAccess))

		bucketReconciler := &BucketReconciler{
			Client:          k8sClient,
			Scheme:          k8sClient.Scheme(),
			ProviderFactory: reconciler.ProviderFactory,
		}
		result, err := bucketReconciler.Reconcile(ctx, reconcile.Request{
			NamespacedName: client.ObjectKeyFromObject(bucket),
		})
		Expect(err).NotTo(HaveOccurred())
		Expect(result.RequeueAfter).NotTo(BeZero())
		Expect(getBucketAccess(ctx, client.ObjectKeyFromObject(secondAccess))).NotTo(BeNil())

		_, err = reconcileBucketAccess(ctx, reconciler, secondAccess)
		Expect(err).NotTo(HaveOccurred())
		_, err = reconcileBucketAccess(ctx, reconciler, secondAccess)
		Expect(err).NotTo(HaveOccurred())
		expectBucketAccessNotFound(ctx, client.ObjectKeyFromObject(secondAccess))
		Expect(bucketAccess.deleteCalls).To(Equal(2))

		result, err = bucketReconciler.Reconcile(ctx, reconcile.Request{
			NamespacedName: client.ObjectKeyFromObject(bucket),
		})
		Expect(err).NotTo(HaveOccurred())
		Expect(result).To(Equal(reconcile.Result{}))
		err = k8sClient.Get(ctx, client.ObjectKeyFromObject(bucket), &vedro.Bucket{})
		Expect(apierrors.IsNotFound(err)).To(BeTrue())
	})

	It("retains external access while cascading a Retain BucketAccess", func() {
		bucket := createBucket(ctx, "bucket")
		createCloudPrincipal(ctx, "principal")
		access := createAppliedBucketAccess(ctx, "cascade-retain")
		fetchedAccess := getBucketAccess(ctx, client.ObjectKeyFromObject(access))
		fetchedAccess.Spec.DeletionPolicy = vedro.DeletionPolicyRetain
		Expect(k8sClient.Update(ctx, fetchedAccess)).To(Succeed())
		addBucketAccessFinalizer(ctx, access)
		controllerutilAddFinalizer(ctx, bucket)
		Expect(k8sClient.Delete(ctx, bucket)).To(Succeed())

		_, err := reconcileBucketAccess(ctx, reconciler, access)
		Expect(err).NotTo(HaveOccurred())
		_, err = reconcileBucketAccess(ctx, reconciler, access)
		Expect(err).NotTo(HaveOccurred())
		expectBucketAccessNotFound(ctx, client.ObjectKeyFromObject(access))
		Expect(bucketAccess.deleteCalls).To(BeZero())
		Expect(provider.cleanupCalled).To(BeFalse())

		bucketReconciler := &BucketReconciler{
			Client:          k8sClient,
			Scheme:          k8sClient.Scheme(),
			ProviderFactory: reconciler.ProviderFactory,
		}
		result, err := bucketReconciler.Reconcile(ctx, reconcile.Request{
			NamespacedName: client.ObjectKeyFromObject(bucket),
		})
		Expect(err).NotTo(HaveOccurred())
		Expect(result).To(Equal(reconcile.Result{}))
		err = k8sClient.Get(ctx, client.ObjectKeyFromObject(bucket), &vedro.Bucket{})
		Expect(apierrors.IsNotFound(err)).To(BeTrue())
	})

	It("does not let an unrelated BucketAccess block Bucket deletion", func() {
		bucket := createBucket(ctx, "bucket")
		access := createBucketAccess(ctx, "unrelated-access", "other-bucket")
		controllerutilAddFinalizer(ctx, bucket)
		Expect(k8sClient.Delete(ctx, bucket)).To(Succeed())

		bucketReconciler := &BucketReconciler{
			Client:          k8sClient,
			Scheme:          k8sClient.Scheme(),
			ProviderFactory: reconciler.ProviderFactory,
		}
		result, err := bucketReconciler.Reconcile(ctx, reconcile.Request{
			NamespacedName: client.ObjectKeyFromObject(bucket),
		})

		Expect(err).NotTo(HaveOccurred())
		Expect(result).To(Equal(reconcile.Result{}))
		err = k8sClient.Get(ctx, client.ObjectKeyFromObject(bucket), &vedro.Bucket{})
		Expect(apierrors.IsNotFound(err)).To(BeTrue())
		Expect(getBucketAccess(ctx, client.ObjectKeyFromObject(access)).DeletionTimestamp.IsZero()).To(BeTrue())
	})

	It("cascades an unapplied BucketAccess without initializing a provider", func() {
		bucket := createBucket(ctx, "bucket")
		createCloudPrincipal(ctx, "principal")
		access := createBucketAccess(ctx, "cascade-unapplied", "bucket")
		addBucketAccessFinalizer(ctx, access)
		controllerutilAddFinalizer(ctx, bucket)
		Expect(k8sClient.Delete(ctx, bucket)).To(Succeed())

		_, err := reconcileBucketAccess(ctx, reconciler, access)
		Expect(err).NotTo(HaveOccurred())
		_, err = reconcileBucketAccess(ctx, reconciler, access)
		Expect(err).NotTo(HaveOccurred())
		expectBucketAccessNotFound(ctx, client.ObjectKeyFromObject(access))
		Expect(bucketAccess.deleteCalls).To(BeZero())
		Expect(provider.cleanupCalled).To(BeFalse())

		bucketReconciler := &BucketReconciler{
			Client:          k8sClient,
			Scheme:          k8sClient.Scheme(),
			ProviderFactory: reconciler.ProviderFactory,
		}
		result, err := bucketReconciler.Reconcile(ctx, reconcile.Request{
			NamespacedName: client.ObjectKeyFromObject(bucket),
		})
		Expect(err).NotTo(HaveOccurred())
		Expect(result).To(Equal(reconcile.Result{}))
	})

	It("maps dependency watches only to their referenced BucketAccess objects", func() {
		bucket := createBucket(ctx, "watched-bucket")
		createBucket(ctx, "other-bucket")
		principal := createCloudPrincipal(ctx, "principal")
		createCloudPrincipal(ctx, "other-principal")
		both := createBucketAccess(ctx, "watch-both", "watched-bucket")
		bucketOnly := createBucketAccess(ctx, "watch-bucket-only", "watched-bucket")
		principalOnly := createBucketAccess(ctx, "watch-principal-only", "other-bucket")
		fetched := getBucketAccess(ctx, client.ObjectKeyFromObject(bucketOnly))
		fetched.Spec.PrincipalRef.Name = "other-principal"
		Expect(k8sClient.Update(ctx, fetched)).To(Succeed())

		bucketRequests := reconciler.findBucketAccessOfBucket(ctx, bucket)
		Expect(bucketRequests).To(ConsistOf(
			reconcile.Request{NamespacedName: client.ObjectKeyFromObject(both)},
			reconcile.Request{NamespacedName: client.ObjectKeyFromObject(bucketOnly)},
		))

		principalRequests := reconciler.findBucketAccessOfPrincipal(ctx, principal)
		Expect(principalRequests).To(ConsistOf(
			reconcile.Request{NamespacedName: client.ObjectKeyFromObject(both)},
			reconcile.Request{NamespacedName: client.ObjectKeyFromObject(principalOnly)},
		))
	})

	It("deletes applied access while its ProviderConfig is terminating and not ready", func() {
		createProviderConfig(ctx)
		providerConfig := getProviderConfig(ctx, types.NamespacedName{Name: "test-provider"})
		markProviderConfigNotReady(ctx, providerConfig)
		addProviderConfigFinalizer(ctx, providerConfig)
		Expect(k8sClient.Delete(ctx, providerConfig)).To(Succeed())
		access := createAppliedBucketAccess(ctx, "delete-with-terminating-provider")
		addBucketAccessFinalizer(ctx, access)
		Expect(k8sClient.Delete(ctx, access)).To(Succeed())

		result, err := reconcileBucketAccess(ctx, reconciler, access)

		Expect(err).NotTo(HaveOccurred())
		Expect(result).To(Equal(reconcile.Result{}))
		Expect(bucketAccess.deleteCalls).To(Equal(1))
		Expect(provider.cleanupCalled).To(BeTrue())
		expectBucketAccessNotFound(ctx, client.ObjectKeyFromObject(access))
	})

	It("removes the finalizer without calling the provider when access was not applied", func() {
		access := createBucketAccess(ctx, "delete-unapplied", "bucket")
		addBucketAccessFinalizer(ctx, access)
		Expect(k8sClient.Delete(ctx, access)).To(Succeed())

		result, err := reconcileBucketAccess(ctx, reconciler, access)

		Expect(err).NotTo(HaveOccurred())
		Expect(result).To(Equal(reconcile.Result{}))
		Expect(bucketAccess.deleteCalls).To(BeZero())
		Expect(provider.cleanupCalled).To(BeFalse())
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

	It("It returns an error without initializing a provider when its ProviderConfig is not ready during deletion", func() {
		createProviderConfig(ctx)
		access := createAppliedBucketAccess(ctx, "delete-invalid-provider")
		addBucketAccessFinalizer(ctx, access)
		providerConfig := getProviderConfig(ctx, types.NamespacedName{
			Name: "test-provider",
		})
		markProviderConfigNotReady(ctx, providerConfig)

		Expect(k8sClient.Delete(ctx, access)).To(Succeed())

		_, err := reconcileBucketAccess(ctx, reconciler, access)

		Expect(err).To(MatchError("ProviderConfig is not Ready"))
		Expect(bucketAccess.deleteCalls).To(BeZero())
		Expect(provider.cleanupCalled).To(BeFalse())
		fetched := getBucketAccess(ctx, client.ObjectKeyFromObject(access))
		Expect(fetched.Finalizers).To(ContainElement(bucketAccessFinalizer))
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
				Name:      "principal",
				Namespace: "default",
			},
			DeletionPolicy: vedro.DeletionPolicyDelete,
			Access:         vedro.Access{Level: vedro.ObjectReader},
		},
	}

	Expect(k8sClient.Create(ctx, access)).To(Succeed())
	DeferCleanup(func() {
		cleanupBucketAccess(ctx, client.ObjectKeyFromObject(access))
	})
	return access
}

func createAppliedBucketAccess(ctx context.Context, name string) *vedro.BucketAccess {
	access := createBucketAccess(ctx, name, "bucket")
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
