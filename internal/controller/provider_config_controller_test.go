package controller

import (
	"context"
	"errors"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	vedro "github.com/svetoch-dev/vedro/api/v1alpha1"
	"github.com/svetoch-dev/vedro/internal/cloud"
	"github.com/svetoch-dev/vedro/internal/conditions"
	"github.com/svetoch-dev/vedro/internal/validation"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

var _ = Describe("ProviderConfigReconciler", func() {
	var (
		reconciler *ProviderConfigReconciler
		provider   *providerConfigTestProvider
	)

	BeforeEach(func() {
		provider = &providerConfigTestProvider{
			fakeProvider: &fakeProvider{},
			validation:   validation.Valid(),
		}
		reconciler = &ProviderConfigReconciler{
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

	It("ignores missing ProviderConfigs", func() {
		result, err := reconciler.Reconcile(ctx, reconcile.Request{
			NamespacedName: types.NamespacedName{Name: "missing-provider-config"},
		})

		Expect(err).NotTo(HaveOccurred())
		Expect(result).To(Equal(reconcile.Result{}))
	})

	It("adds a finalizer and marks a valid ProviderConfig ready", func() {
		providerConfig := createUnreadyProviderConfigNamed(ctx, "provider-config-success")

		result, err := reconcileProviderConfig(ctx, reconciler, providerConfig)

		Expect(err).NotTo(HaveOccurred())
		Expect(result).To(Equal(reconcile.Result{}))

		fetched := getProviderConfig(ctx, client.ObjectKeyFromObject(providerConfig))
		Expect(fetched.Finalizers).To(ContainElement(providerConfigFinalizer))
		Expect(fetched.Status.ObservedGeneration).To(Equal(fetched.Generation))
		condition := meta.FindStatusCondition(
			fetched.Status.Conditions,
			conditions.TypeProviderConfigReady,
		)
		Expect(condition).NotTo(BeNil())
		Expect(condition.Status).To(Equal(metav1.ConditionTrue))
		Expect(condition.Reason).To(Equal(conditions.ReasonProviderConfigReconciled))
		Expect(condition.Message).To(Equal("ProviderConfig Reconciled"))
		Expect(provider.cleanupCalled).To(BeTrue())
	})

	It("records provider factory errors", func() {
		providerConfig := createUnreadyProviderConfigNamed(ctx, "provider-config-factory-error")
		reconciler.ProviderFactory = func(
			context.Context,
			vedro.ProviderConfig,
			client.Client,
		) (cloud.Provider, error) {
			return nil, errors.New("provider setup failed")
		}

		result, err := reconcileProviderConfig(ctx, reconciler, providerConfig)

		Expect(err).To(MatchError("provider setup failed"))
		Expect(result).To(Equal(reconcile.Result{}))
		fetched := getProviderConfig(ctx, client.ObjectKeyFromObject(providerConfig))
		condition := meta.FindStatusCondition(
			fetched.Status.Conditions,
			conditions.TypeProviderConfigReady,
		)
		Expect(condition).NotTo(BeNil())
		Expect(condition.Status).To(Equal(metav1.ConditionFalse))
		Expect(condition.Reason).To(Equal(conditions.ReasonProviderConfigError))
		Expect(condition.Message).To(Equal("provider setup failed"))
		Expect(provider.cleanupCalled).To(BeFalse())
	})

	It("records invalid ProviderConfig specs", func() {
		providerConfig := createUnreadyProviderConfigNamed(ctx, "provider-config-invalid")
		provider.validation = validation.Invalid("invalid project id")

		result, err := reconcileProviderConfig(ctx, reconciler, providerConfig)

		Expect(err).NotTo(HaveOccurred())
		Expect(result).To(Equal(reconcile.Result{}))
		fetched := getProviderConfig(ctx, client.ObjectKeyFromObject(providerConfig))
		condition := meta.FindStatusCondition(
			fetched.Status.Conditions,
			conditions.TypeProviderConfigReady,
		)
		Expect(condition).NotTo(BeNil())
		Expect(condition.Status).To(Equal(metav1.ConditionFalse))
		Expect(condition.Reason).To(Equal(conditions.ReasonProviderConfigInvalidSpec))
		Expect(condition.Message).To(Equal("invalid project id"))
		Expect(provider.cleanupCalled).To(BeTrue())
	})

	It("does not fail reconcile when provider cleanup fails", func() {
		providerConfig := createUnreadyProviderConfigNamed(ctx, "provider-config-cleanup-error")
		provider.cleanupErr = errors.New("cleanup failed")

		result, err := reconcileProviderConfig(ctx, reconciler, providerConfig)

		Expect(err).NotTo(HaveOccurred())
		Expect(result).To(Equal(reconcile.Result{}))
		Expect(provider.cleanupCalled).To(BeTrue())
		fetched := getProviderConfig(ctx, client.ObjectKeyFromObject(providerConfig))
		condition := meta.FindStatusCondition(
			fetched.Status.Conditions,
			conditions.TypeProviderConfigReady,
		)
		Expect(condition).NotTo(BeNil())
		Expect(condition.Status).To(Equal(metav1.ConditionTrue))
	})

	It("removes its finalizer during deletion when it is not referenced", func() {
		providerConfig := createUnreadyProviderConfigNamed(ctx, "provider-config-delete")
		addProviderConfigFinalizer(ctx, providerConfig)
		Expect(k8sClient.Delete(ctx, providerConfig)).To(Succeed())

		result, err := reconcileProviderConfig(ctx, reconciler, providerConfig)

		Expect(err).NotTo(HaveOccurred())
		Expect(result).To(Equal(reconcile.Result{}))
		fetched := &vedro.ProviderConfig{}
		err = k8sClient.Get(ctx, client.ObjectKeyFromObject(providerConfig), fetched)
		Expect(apierrors.IsNotFound(err)).To(BeTrue())
		Expect(provider.cleanupCalled).To(BeFalse())
	})

	It("keeps its finalizer and requeues deletion while referenced", func() {
		providerConfig := createUnreadyProviderConfigNamed(ctx, "provider-config-referenced")
		createBucket(ctx, "provider-config-reference", func(spec *vedro.BucketSpec) {
			spec.ProviderRef.Name = providerConfig.Name
		})
		addProviderConfigFinalizer(ctx, providerConfig)
		Expect(k8sClient.Delete(ctx, providerConfig)).To(Succeed())

		result, err := reconcileProviderConfig(ctx, reconciler, providerConfig)

		Expect(err).NotTo(HaveOccurred())
		Expect(result.RequeueAfter).To(Equal(10 * time.Second))
		fetched := getProviderConfig(ctx, client.ObjectKeyFromObject(providerConfig))
		Expect(fetched.Finalizers).To(ContainElement(providerConfigFinalizer))
		Expect(provider.cleanupCalled).To(BeFalse())
	})

	It("keeps its finalizer while referenced by a CloudPrincipal", func() {
		providerConfig := createUnreadyProviderConfigNamed(ctx, "provider-config-principal-reference")
		createCloudPrincipal(ctx, "provider-config-principal-reference", func(principal *vedro.CloudPrincipal) {
			principal.Spec.ProviderRef.Name = providerConfig.Name
		})
		addProviderConfigFinalizer(ctx, providerConfig)
		Expect(k8sClient.Delete(ctx, providerConfig)).To(Succeed())

		result, err := reconcileProviderConfig(ctx, reconciler, providerConfig)

		Expect(err).NotTo(HaveOccurred())
		Expect(result.RequeueAfter).To(Equal(10 * time.Second))
		fetched := getProviderConfig(ctx, client.ObjectKeyFromObject(providerConfig))
		Expect(fetched.Finalizers).To(ContainElement(providerConfigFinalizer))
		Expect(provider.cleanupCalled).To(BeFalse())
	})
})

type providerConfigTestProvider struct {
	*fakeProvider
	validation validation.ValidationResult
}

func (p *providerConfigTestProvider) ValidateProviderConfigSpec(
	vedro.ProviderConfig,
) validation.ValidationResult {
	return p.validation
}

func createProviderConfig(ctx context.Context) {
	createProviderConfigNamed(ctx, "test-provider")
}

func createProviderConfigNamed(ctx context.Context, name string) {
	providerConfig := createUnreadyProviderConfigNamed(ctx, name)
	markProviderConfigReady(ctx, providerConfig)
}

func createUnreadyProviderConfigNamed(ctx context.Context, name string) *vedro.ProviderConfig {
	providerConfig := &vedro.ProviderConfig{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "vedro.svetoch.dev/v1alpha1",
			Kind:       "ProviderConfig",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Spec: vedro.ProviderConfigSpec{
			Type:      vedro.ProviderTypeGCP,
			ProjectId: "test-project",
			Region:    "europe-west1",
			Method:    vedro.AuthMethodWorkloadIdentity,
		},
	}

	Expect(k8sClient.Create(ctx, providerConfig)).To(Succeed())

	DeferCleanup(func() {
		cleanupProviderConfig(ctx, client.ObjectKeyFromObject(providerConfig))
	})

	return providerConfig
}

func reconcileProviderConfig(
	ctx context.Context,
	reconciler *ProviderConfigReconciler,
	providerConfig *vedro.ProviderConfig,
) (reconcile.Result, error) {
	return reconciler.Reconcile(ctx, reconcile.Request{
		NamespacedName: client.ObjectKeyFromObject(providerConfig),
	})
}

func getProviderConfig(ctx context.Context, key client.ObjectKey) *vedro.ProviderConfig {
	fetched := &vedro.ProviderConfig{}
	Expect(k8sClient.Get(
		ctx,
		key,
		fetched,
	)).To(Succeed())

	return fetched
}

func markProviderConfigReady(ctx context.Context, providerConfig *vedro.ProviderConfig) {
	fetched := getProviderConfig(ctx, client.ObjectKeyFromObject(providerConfig))

	fetched.Status.ObservedGeneration = fetched.Generation

	meta.SetStatusCondition(&fetched.Status.Conditions, metav1.Condition{
		Type:               "Ready",
		Status:             metav1.ConditionTrue,
		ObservedGeneration: fetched.Generation,
		Reason:             conditions.ReasonProviderConfigReconciled,
		Message:            "ProviderConfig Reconciled",
	})

	Expect(k8sClient.Status().Update(ctx, fetched)).To(Succeed())
}

func markProviderConfigNotReady(ctx context.Context, providerConfig *vedro.ProviderConfig) {
	fetched := getProviderConfig(ctx, client.ObjectKeyFromObject(providerConfig))

	fetched.Status.ObservedGeneration = fetched.Generation

	meta.SetStatusCondition(&fetched.Status.Conditions, metav1.Condition{
		Type:               "Ready",
		Status:             metav1.ConditionFalse,
		ObservedGeneration: fetched.Generation,
		Reason:             conditions.ReasonProviderConfigError,
		Message:            "ProviderConfig Error",
	})

	Expect(k8sClient.Status().Update(ctx, fetched)).To(Succeed())
}

func cleanupProviderConfig(ctx context.Context, key client.ObjectKey) {
	providerConfig := &vedro.ProviderConfig{}
	err := k8sClient.Get(ctx, key, providerConfig)
	if apierrors.IsNotFound(err) {
		return
	}
	Expect(err).NotTo(HaveOccurred())
	if len(providerConfig.Finalizers) > 0 {
		providerConfig.Finalizers = nil
		Expect(k8sClient.Update(ctx, providerConfig)).To(Succeed())
	}
	err = k8sClient.Delete(ctx, providerConfig)
	if err != nil {
		Expect(apierrors.IsNotFound(err)).To(BeTrue())
	}
}

func addProviderConfigFinalizer(ctx context.Context, providerConfig *vedro.ProviderConfig) {
	fetched := getProviderConfig(ctx, client.ObjectKeyFromObject(providerConfig))
	fetched.Finalizers = append(fetched.Finalizers, providerConfigFinalizer)
	Expect(k8sClient.Update(ctx, fetched)).To(Succeed())
}
