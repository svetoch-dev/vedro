/*
Copyright 2026.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

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
	"github.com/svetoch-dev/vedro/internal/validation"
)

var _ = Describe("BucketReconciler", func() {
	var (
		reconciler *BucketReconciler
		provider   *fakeProvider
	)

	BeforeEach(func() {
		provider = &fakeProvider{
			capabilities: cloud.Capabilities{
				Bucket: cloud.BucketCapabilities{
					Lifecycle: cloud.LifecycleCapabilities{
						RuleNames:      true,
						RuleExpiration: true,
					},
					StorageClass: cloud.StorageClassCapabilities{
						Ice:  true,
						Cold: true,
						Warm: true,
					},
					Versioning:             true,
					PublicAccessPrevention: true,
					Labels:                 true,
				},
			},
			bucket: &fakeBucketProvider{
				validateResult: validation.Valid(),
				ensureResult: &cloud.BucketAttrs{
					Name:     "external-bucket",
					Id:       "external-bucket-id",
					Location: "europe-west1",
					Properties: &vedro.BucketProperties{
						StorageClass: vedro.BucketStorageClassStandard,
						Labels: map[string]string{
							"env": "test",
						},
					},
				},
			},
		}

		reconciler = &BucketReconciler{
			Client: k8sClient,
			Scheme: k8sClient.Scheme(),
			ProviderFactory: func(
				ctx context.Context,
				cfg vedro.ProviderConfig,
				kubeClient client.Client,
			) (cloud.Provider, error) {
				return provider, nil
			},
		}
	})

	It("ignores missing Buckets", func() {
		result, err := reconciler.Reconcile(ctx, reconcile.Request{
			NamespacedName: types.NamespacedName{
				Name:      "missing-bucket",
				Namespace: "default",
			},
		})

		Expect(err).NotTo(HaveOccurred())
		Expect(result).To(Equal(reconcile.Result{}))
	})

	It("adds the finalizer and marks ProviderConfig missing", func() {
		bucket := createBucket(ctx, "missing-provider", func(spec *vedro.BucketSpec) {
			spec.ProviderRef.Name = "missing-provider"
		})

		result, err := reconciler.Reconcile(ctx, reconcile.Request{
			NamespacedName: client.ObjectKeyFromObject(bucket),
		})

		Expect(err).NotTo(HaveOccurred())
		Expect(result).To(Equal(reconcile.Result{}))

		fetched := getBucket(ctx, client.ObjectKeyFromObject(bucket))
		Expect(fetched.Finalizers).To(ContainElement(bucketFinalizer))
		Expect(fetched.Status.ObservedGeneration).To(Equal(fetched.Generation))

		condition := meta.FindStatusCondition(fetched.Status.Conditions, conditions.TypeProviderConfigReady)
		Expect(condition).NotTo(BeNil())
		Expect(condition.Status).To(Equal(metav1.ConditionFalse))
		Expect(condition.Reason).To(Equal(conditions.ReasonProviderConfigNotFound))
	})

	It("records provider factory errors", func() {
		bucket := createBucket(ctx, "provider-factory-error")
		createProviderConfig(ctx)
		reconciler.ProviderFactory = func(
			ctx context.Context,
			cfg vedro.ProviderConfig,
			kubeClient client.Client,
		) (cloud.Provider, error) {
			return nil, errors.New("provider setup failed")
		}

		_, err := reconciler.Reconcile(ctx, reconcile.Request{
			NamespacedName: client.ObjectKeyFromObject(bucket),
		})

		Expect(err).To(HaveOccurred())

		fetched := getBucket(ctx, client.ObjectKeyFromObject(bucket))
		providerCondition := meta.FindStatusCondition(fetched.Status.Conditions, conditions.TypeProviderConfigReady)
		Expect(providerCondition).NotTo(BeNil())
		Expect(providerCondition.Status).To(Equal(metav1.ConditionFalse))
		Expect(providerCondition.Reason).To(Equal(conditions.ReasonProviderConfigError))

		readyCondition := meta.FindStatusCondition(fetched.Status.Conditions, conditions.TypeReady)
		Expect(readyCondition).NotTo(BeNil())
		Expect(readyCondition.Status).To(Equal(metav1.ConditionFalse))
		Expect(readyCondition.Reason).To(Equal(conditions.ReasonProviderConfigError))
		Expect(provider.bucket.ensureCalls).To(Equal(0))
		Expect(provider.cleanupCalled).To(BeFalse())
	})

	It("records invalid Bucket specs without ensuring the external bucket", func() {
		bucket := createBucket(ctx, "invalid-spec")
		createProviderConfig(ctx)
		provider.bucket.validateResult = validation.Invalid("invalid location")

		result, err := reconciler.Reconcile(ctx, reconcile.Request{
			NamespacedName: client.ObjectKeyFromObject(bucket),
		})

		Expect(err).NotTo(HaveOccurred())
		Expect(result).To(Equal(reconcile.Result{}))

		fetched := getBucket(ctx, client.ObjectKeyFromObject(bucket))
		condition := meta.FindStatusCondition(fetched.Status.Conditions, conditions.TypeReady)
		Expect(condition).NotTo(BeNil())
		Expect(condition.Status).To(Equal(metav1.ConditionFalse))
		Expect(condition.Reason).To(Equal(conditions.ReasonBucketInvalidSpec))
		Expect(condition.Message).To(Equal("invalid location"))
		Expect(provider.bucket.ensureCalls).To(Equal(0))
	})

	It("fails fast when unsupported features are requested with Fail policy", func() {
		bucket := createBucket(ctx, "unsupported-fail", func(spec *vedro.BucketSpec) {
			spec.Versioning = &vedro.BucketVersioning{Enabled: true}
			spec.UnsupportedFeaturePolicy = vedro.UnsupportedFeaturePolicyFail
		})
		createProviderConfig(ctx)
		provider.capabilities.Bucket.Versioning = false

		result, err := reconciler.Reconcile(ctx, reconcile.Request{
			NamespacedName: client.ObjectKeyFromObject(bucket),
		})

		Expect(err).NotTo(HaveOccurred())
		Expect(result).To(Equal(reconcile.Result{}))

		fetched := getBucket(ctx, client.ObjectKeyFromObject(bucket))
		Expect(fetched.Status.UnsupportedFeatures).NotTo(BeEmpty())
		condition := meta.FindStatusCondition(fetched.Status.Conditions, conditions.TypeReady)
		Expect(condition).NotTo(BeNil())
		Expect(condition.Status).To(Equal(metav1.ConditionFalse))
		Expect(condition.Reason).To(Equal(conditions.ReasonBucketUnsupportedFeatures))
		Expect(provider.bucket.ensureCalls).To(Equal(0))
	})

	It("warns about unsupported features and continues reconciling with Warn policy", func() {
		bucket := createBucket(ctx, "unsupported-warn", func(spec *vedro.BucketSpec) {
			spec.Versioning = &vedro.BucketVersioning{Enabled: true}
			spec.UnsupportedFeaturePolicy = vedro.UnsupportedFeaturePolicyWarn
		})
		createProviderConfig(ctx)
		provider.capabilities.Bucket.Versioning = false

		result, err := reconciler.Reconcile(ctx, reconcile.Request{
			NamespacedName: client.ObjectKeyFromObject(bucket),
		})

		Expect(err).NotTo(HaveOccurred())
		Expect(result).To(Equal(reconcile.Result{}))

		fetched := getBucket(ctx, client.ObjectKeyFromObject(bucket))
		Expect(fetched.Status.UnsupportedFeatures).NotTo(BeEmpty())
		Expect(fetched.Status.ExternalName).To(Equal("external-bucket"))
		Expect(provider.bucket.ensureCalls).To(Equal(1))
	})

	It("sets successful Bucket status after ensuring the external bucket", func() {
		bucket := createBucket(ctx, "successful-reconcile")
		createProviderConfig(ctx)

		result, err := reconciler.Reconcile(ctx, reconcile.Request{
			NamespacedName: client.ObjectKeyFromObject(bucket),
		})

		Expect(err).NotTo(HaveOccurred())
		Expect(result).To(Equal(reconcile.Result{}))

		fetched := getBucket(ctx, client.ObjectKeyFromObject(bucket))
		Expect(fetched.Status.ExternalName).To(Equal("external-bucket"))
		Expect(fetched.Status.ExternalId).To(Equal("external-bucket-id"))
		Expect(fetched.Status.Location).To(Equal("europe-west1"))
		Expect(fetched.Status.ObservedProvider).To(Equal("test-provider"))
		Expect(fetched.Status.Applied).NotTo(BeNil())
		Expect(fetched.Status.Applied.Labels).To(HaveKeyWithValue("env", "test"))

		providerCondition := meta.FindStatusCondition(fetched.Status.Conditions, conditions.TypeProviderConfigReady)
		Expect(providerCondition).NotTo(BeNil())
		Expect(providerCondition.Status).To(Equal(metav1.ConditionTrue))
		Expect(providerCondition.Reason).To(Equal(conditions.ReasonProviderConfigReconciled))

		readyCondition := meta.FindStatusCondition(fetched.Status.Conditions, conditions.TypeReady)
		Expect(readyCondition).NotTo(BeNil())
		Expect(readyCondition.Status).To(Equal(metav1.ConditionTrue))
		Expect(readyCondition.Reason).To(Equal(conditions.ReasonBucketReconciled))
		Expect(provider.bucket.ensureCalls).To(Equal(1))
		Expect(provider.cleanupCalled).To(BeTrue())
	})

	It("does not fail reconcile when provider cleanup fails", func() {
		bucket := createBucket(ctx, "cleanup-error")
		createProviderConfig(ctx)
		provider.cleanupErr = errors.New("cleanup failed")

		result, err := reconciler.Reconcile(ctx, reconcile.Request{
			NamespacedName: client.ObjectKeyFromObject(bucket),
		})

		Expect(err).NotTo(HaveOccurred())
		Expect(result).To(Equal(reconcile.Result{}))
		Expect(provider.cleanupCalled).To(BeTrue())

		fetched := getBucket(ctx, client.ObjectKeyFromObject(bucket))
		readyCondition := meta.FindStatusCondition(fetched.Status.Conditions, conditions.TypeReady)
		Expect(readyCondition).NotTo(BeNil())
		Expect(readyCondition.Status).To(Equal(metav1.ConditionTrue))
		Expect(readyCondition.Reason).To(Equal(conditions.ReasonBucketReconciled))
	})

	It("records ensure errors", func() {
		bucket := createBucket(ctx, "ensure-error")
		createProviderConfig(ctx)
		provider.bucket.ensureErr = errors.New("ensure failed")

		result, err := reconciler.Reconcile(ctx, reconcile.Request{
			NamespacedName: client.ObjectKeyFromObject(bucket),
		})

		Expect(err).To(MatchError("ensure failed"))
		Expect(result).To(Equal(reconcile.Result{}))

		fetched := getBucket(ctx, client.ObjectKeyFromObject(bucket))
		condition := meta.FindStatusCondition(fetched.Status.Conditions, conditions.TypeReady)
		Expect(condition).NotTo(BeNil())
		Expect(condition.Status).To(Equal(metav1.ConditionFalse))
		Expect(condition.Reason).To(Equal(conditions.ReasonBucketEnsureError))
		Expect(condition.Message).To(Equal("ensure failed"))
		Expect(provider.bucket.ensureCalls).To(Equal(1))
	})

	It("retains the external bucket and removes the finalizer for Retain policy", func() {
		bucket := createBucket(ctx, "retain-policy")
		controllerutilAddFinalizer(ctx, bucket)
		Expect(k8sClient.Delete(ctx, bucket)).To(Succeed())

		result, err := reconciler.Reconcile(ctx, reconcile.Request{
			NamespacedName: client.ObjectKeyFromObject(bucket),
		})

		Expect(err).NotTo(HaveOccurred())
		Expect(result).To(Equal(reconcile.Result{}))
		Expect(provider.bucket.deleteCalls).To(BeZero())
		Expect(provider.cleanupCalled).To(BeFalse())

		fetched := &vedro.Bucket{}
		err = k8sClient.Get(ctx, client.ObjectKeyFromObject(bucket), fetched)
		Expect(apierrors.IsNotFound(err)).To(BeTrue())
	})

	It("deletes the external bucket and removes the finalizer for Delete policy", func() {
		bucket := createBucket(ctx, "delete-policy", func(spec *vedro.BucketSpec) {
			spec.DeletionPolicy = vedro.DeletionPolicyDelete
		})
		createProviderConfig(ctx)
		controllerutilAddFinalizer(ctx, bucket)
		Expect(k8sClient.Delete(ctx, bucket)).To(Succeed())

		result, err := reconciler.Reconcile(ctx, reconcile.Request{
			NamespacedName: client.ObjectKeyFromObject(bucket),
		})

		Expect(err).NotTo(HaveOccurred())
		Expect(result).To(Equal(reconcile.Result{}))
		Expect(provider.bucket.deleteCalls).To(Equal(1))

		fetched := &vedro.Bucket{}
		err = k8sClient.Get(ctx, client.ObjectKeyFromObject(bucket), fetched)
		Expect(apierrors.IsNotFound(err)).To(BeTrue())
	})

	It("uses the observed ProviderConfig when deleting a bucket", func() {
		bucket := createBucket(ctx, "observed-provider", func(spec *vedro.BucketSpec) {
			spec.DeletionPolicy = vedro.DeletionPolicyDelete
		})
		createProviderConfigNamed(ctx, "observed-provider")
		fetched := getBucket(ctx, client.ObjectKeyFromObject(bucket))
		fetched.Status.ObservedProvider = "observed-provider"
		Expect(k8sClient.Status().Update(ctx, fetched)).To(Succeed())
		controllerutilAddFinalizer(ctx, bucket)

		var configuredProvider string
		reconciler.ProviderFactory = func(
			_ context.Context,
			cfg vedro.ProviderConfig,
			_ client.Client,
		) (cloud.Provider, error) {
			configuredProvider = cfg.Name
			return provider, nil
		}
		Expect(k8sClient.Delete(ctx, bucket)).To(Succeed())

		_, err := reconciler.Reconcile(ctx, reconcile.Request{
			NamespacedName: client.ObjectKeyFromObject(bucket),
		})

		Expect(err).NotTo(HaveOccurred())
		Expect(configuredProvider).To(Equal("observed-provider"))
		Expect(provider.bucket.deleteCalls).To(Equal(1))
	})

	It("records delete errors and keeps the bucket finalizer", func() {
		bucket := createBucket(ctx, "delete-error", func(spec *vedro.BucketSpec) {
			spec.DeletionPolicy = vedro.DeletionPolicyDelete
		})
		createProviderConfig(ctx)
		provider.bucket.deleteErr = errors.New("delete failed")
		controllerutilAddFinalizer(ctx, bucket)
		Expect(k8sClient.Delete(ctx, bucket)).To(Succeed())

		_, err := reconciler.Reconcile(ctx, reconcile.Request{
			NamespacedName: client.ObjectKeyFromObject(bucket),
		})

		Expect(err).To(MatchError("delete failed"))
		Expect(provider.bucket.deleteCalls).To(Equal(1))
		Expect(provider.cleanupCalled).To(BeTrue())

		fetched := getBucket(ctx, client.ObjectKeyFromObject(bucket))
		Expect(fetched.Finalizers).To(ContainElement(bucketFinalizer))
		condition := meta.FindStatusCondition(fetched.Status.Conditions, conditions.TypeReady)
		Expect(condition).NotTo(BeNil())
		Expect(condition.Status).To(Equal(metav1.ConditionFalse))
		Expect(condition.Reason).To(Equal(conditions.ReasonBucketDeleteError))
		Expect(condition.Message).To(Equal("delete failed"))
	})
})

type fakeProvider struct {
	capabilities cloud.Capabilities
	bucket       *fakeBucketProvider
	principal    *fakePrincipalProvider
	bucketAccess *fakeBucketAccess
	cleanupErr   error

	cleanupCalled bool
}

func (p *fakeProvider) Capabilities() cloud.Capabilities {
	return p.capabilities
}

func (p *fakeProvider) Bucket() cloud.BucketProvider {
	return p.bucket
}

func (p *fakeProvider) Principal() cloud.PrincipalProvider {
	return p.principal
}

func (p *fakeProvider) Access() cloud.BucketAccessProvider {
	return p.bucketAccess
}

func (p *fakeProvider) ValidateProviderConfigSpec(cfg vedro.ProviderConfig) validation.ValidationResult {
	return validation.Valid()
}

func (p *fakeProvider) Cleanup(ctx context.Context) error {
	p.cleanupCalled = true
	return p.cleanupErr
}

type fakeBucketProvider struct {
	validateResult validation.ValidationResult
	ensureResult   *cloud.BucketAttrs
	ensureErr      error
	deleteErr      error
	ensureCalls    int
	deleteCalls    int
}

func (p *fakeBucketProvider) ValidateBucketSpec(spec vedro.Bucket, pType vedro.ProviderType) validation.ValidationResult {
	return p.validateResult
}

func (p *fakeBucketProvider) EnsureBucket(
	ctx context.Context,
	spec vedro.Bucket,
) (*cloud.BucketAttrs, error) {
	p.ensureCalls++
	return p.ensureResult, p.ensureErr
}

func (p *fakeBucketProvider) DeleteBucket(ctx context.Context, bckt vedro.Bucket) error {
	p.deleteCalls++
	return p.deleteErr
}

func createBucket(
	ctx context.Context,
	name string,
	mutators ...func(*vedro.BucketSpec),
) *vedro.Bucket {
	bucket := &vedro.Bucket{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "vedro.svetoch.dev/v1alpha1",
			Kind:       "Bucket",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: "default",
		},
		Spec: vedro.BucketSpec{
			ProviderRef: vedro.ProviderConfigReference{
				Name: "test-provider",
			},
			Location:                 "europe-west1",
			StorageClass:             vedro.BucketStorageClassStandard,
			DeletionPolicy:           vedro.DeletionPolicyRetain,
			UnsupportedFeaturePolicy: vedro.UnsupportedFeaturePolicyFail,
		},
	}

	for _, mutate := range mutators {
		mutate(&bucket.Spec)
	}

	Expect(k8sClient.Create(ctx, bucket)).To(Succeed())
	DeferCleanup(func() {
		cleanupBucket(ctx, client.ObjectKeyFromObject(bucket))
	})

	return bucket
}

func getBucket(ctx context.Context, key client.ObjectKey) *vedro.Bucket {
	bucket := &vedro.Bucket{}
	Expect(k8sClient.Get(ctx, key, bucket)).To(Succeed())
	return bucket
}

func cleanupBucket(ctx context.Context, key client.ObjectKey) {
	bucket := &vedro.Bucket{}
	err := k8sClient.Get(ctx, key, bucket)
	if apierrors.IsNotFound(err) {
		return
	}
	Expect(err).NotTo(HaveOccurred())

	bucket.Finalizers = nil
	Expect(k8sClient.Update(ctx, bucket)).To(Succeed())
	err = k8sClient.Delete(ctx, bucket)
	if err != nil {
		Expect(apierrors.IsNotFound(err)).To(BeTrue())
	}
}

func controllerutilAddFinalizer(ctx context.Context, bucket *vedro.Bucket) {
	fetched := getBucket(ctx, client.ObjectKeyFromObject(bucket))
	fetched.Finalizers = append(fetched.Finalizers, bucketFinalizer)
	Expect(k8sClient.Update(ctx, fetched)).To(Succeed())
}
