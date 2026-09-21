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
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
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

	It("finishes deletion after the referencing resource is removed", func() {
		providerConfig := createUnreadyProviderConfigNamed(ctx, "provider-config-reference-removed")
		bucket := createBucket(ctx, "temporary-provider-reference", func(spec *vedro.BucketSpec) {
			spec.ProviderRef.Name = providerConfig.Name
		})
		addProviderConfigFinalizer(ctx, providerConfig)
		Expect(k8sClient.Delete(ctx, providerConfig)).To(Succeed())

		result, err := reconcileProviderConfig(ctx, reconciler, providerConfig)

		Expect(err).NotTo(HaveOccurred())
		Expect(result.RequeueAfter).To(Equal(10 * time.Second))
		Expect(k8sClient.Delete(ctx, bucket)).To(Succeed())

		result, err = reconcileProviderConfig(ctx, reconciler, providerConfig)

		Expect(err).NotTo(HaveOccurred())
		Expect(result).To(Equal(reconcile.Result{}))
		err = k8sClient.Get(ctx, client.ObjectKeyFromObject(providerConfig), &vedro.ProviderConfig{})
		Expect(apierrors.IsNotFound(err)).To(BeTrue())
	})

	It("recovers from an invalid spec after the next generation validates", func() {
		providerConfig := createUnreadyProviderConfigNamed(ctx, "provider-config-recovers")
		provider.validation = validation.Invalid("invalid region")

		_, err := reconcileProviderConfig(ctx, reconciler, providerConfig)
		Expect(err).NotTo(HaveOccurred())
		invalid := getProviderConfig(ctx, client.ObjectKeyFromObject(providerConfig))
		invalidGeneration := invalid.Generation
		condition := meta.FindStatusCondition(invalid.Status.Conditions, conditions.TypeProviderConfigReady)
		Expect(condition.Status).To(Equal(metav1.ConditionFalse))
		Expect(condition.Reason).To(Equal(conditions.ReasonProviderConfigInvalidSpec))

		invalid.Spec.Region = "europe-west2"
		Expect(k8sClient.Update(ctx, invalid)).To(Succeed())
		provider.validation = validation.Valid()

		_, err = reconcileProviderConfig(ctx, reconciler, providerConfig)

		Expect(err).NotTo(HaveOccurred())
		fetched := getProviderConfig(ctx, client.ObjectKeyFromObject(providerConfig))
		Expect(fetched.Generation).To(BeNumerically(">", invalidGeneration))
		Expect(fetched.Status.ObservedGeneration).To(Equal(fetched.Generation))
		Expect(fetched.Status.Conditions).To(HaveLen(1))
		condition = meta.FindStatusCondition(fetched.Status.Conditions, conditions.TypeProviderConfigReady)
		Expect(condition.Status).To(Equal(metav1.ConditionTrue))
		Expect(condition.Reason).To(Equal(conditions.ReasonProviderConfigReconciled))
		Expect(condition.Message).To(Equal("ProviderConfig Reconciled"))
	})

	It("reconciles idempotently", func() {
		providerConfig := createUnreadyProviderConfigNamed(ctx, "provider-config-idempotent")

		_, err := reconcileProviderConfig(ctx, reconciler, providerConfig)
		Expect(err).NotTo(HaveOccurred())
		_, err = reconcileProviderConfig(ctx, reconciler, providerConfig)

		Expect(err).NotTo(HaveOccurred())
		fetched := getProviderConfig(ctx, client.ObjectKeyFromObject(providerConfig))
		Expect(fetched.Finalizers).To(ConsistOf(providerConfigFinalizer))
		Expect(fetched.Status.Conditions).To(HaveLen(1))
		Expect(provider.cleanupCalls).To(Equal(2))
	})

	Describe("CRD validation", func() {
		It("rejects provider type changes", func() {
			providerConfig := createUnreadyProviderConfigNamed(ctx, "immutable-provider-type")
			providerConfig.Spec.Type = vedro.ProviderTypeYandexCloud

			err := k8sClient.Update(ctx, providerConfig)

			Expect(apierrors.IsInvalid(err)).To(BeTrue())
			Expect(err).To(MatchError(ContainSubstring("type is immutable")))
		})

		It("rejects project ID changes", func() {
			providerConfig := createUnreadyProviderConfigNamed(ctx, "immutable-project-id")
			providerConfig.Spec.ProjectId = "another-project"

			err := k8sClient.Update(ctx, providerConfig)

			Expect(apierrors.IsInvalid(err)).To(BeTrue())
			Expect(err).To(MatchError(ContainSubstring("projectId is immutable")))
		})

		It("requires a credentials Secret for StaticCredentials", func() {
			providerConfig := newProviderConfig("static-without-secret")
			providerConfig.Spec.Method = vedro.AuthMethodStaticCredentials

			err := k8sClient.Create(ctx, providerConfig)

			Expect(apierrors.IsInvalid(err)).To(BeTrue())
			Expect(err).To(MatchError(ContainSubstring(
				"credentialsSecretRef is required when method is StaticCredentials",
			)))
		})

		It("rejects a credentials Secret for WorkloadIdentity", func() {
			providerConfig := newProviderConfig("workload-with-secret")
			providerConfig.Spec.CredentialsSecretRef = &corev1.SecretReference{
				Name: "provider-secret", Namespace: "default",
			}

			err := k8sClient.Create(ctx, providerConfig)

			Expect(apierrors.IsInvalid(err)).To(BeTrue())
			Expect(err).To(MatchError(ContainSubstring(
				"credentialsSecretRef must not be set when method is WorkloadIdentity",
			)))
		})
	})

	Describe("Secret watch mapping", func() {
		It("enqueues every ProviderConfig referencing the Secret", func() {
			first := newProviderConfig("first-secret-provider")
			first.Spec.Method = vedro.AuthMethodStaticCredentials
			first.Spec.CredentialsSecretRef = &corev1.SecretReference{Name: "credentials", Namespace: "team"}
			second := first.DeepCopy()
			second.Name = "second-secret-provider"
			unrelated := newProviderConfig("unrelated-secret-provider")
			unrelated.Spec.Method = vedro.AuthMethodStaticCredentials
			unrelated.Spec.CredentialsSecretRef = &corev1.SecretReference{Name: "other", Namespace: "team"}
			otherNamespace := first.DeepCopy()
			otherNamespace.Name = "other-namespace-provider"
			otherNamespace.Spec.CredentialsSecretRef.Namespace = "other"
			watchClient := newProviderConfigWatchClient(first, second, unrelated, otherNamespace)
			watchReconciler := &ProviderConfigReconciler{Client: watchClient}

			requests := watchReconciler.findProviderConfigsOfSecret(ctx, &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{Name: "credentials", Namespace: "team"},
			})

			Expect(requests).To(ConsistOf(
				reconcile.Request{NamespacedName: types.NamespacedName{Name: first.Name}},
				reconcile.Request{NamespacedName: types.NamespacedName{Name: second.Name}},
			))
		})

		It("returns no requests for an unrelated Secret or object type", func() {
			providerConfig := newProviderConfig("watched-provider")
			providerConfig.Spec.Method = vedro.AuthMethodStaticCredentials
			providerConfig.Spec.CredentialsSecretRef = &corev1.SecretReference{Name: "credentials", Namespace: "team"}
			watchReconciler := &ProviderConfigReconciler{
				Client: newProviderConfigWatchClient(providerConfig),
			}

			requests := watchReconciler.findProviderConfigsOfSecret(ctx, &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{Name: "other", Namespace: "team"},
			})
			Expect(requests).To(BeEmpty())
			Expect(watchReconciler.findProviderConfigsOfSecret(ctx, &vedro.Bucket{})).To(BeNil())
		})

		It("returns no requests when ProviderConfigs cannot be listed", func() {
			watchReconciler := &ProviderConfigReconciler{Client: &providerConfigErrorClient{
				Client:  newProviderConfigWatchClient(),
				listErr: errors.New("list failed"),
			}}

			requests := watchReconciler.findProviderConfigsOfSecret(ctx, &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{Name: "credentials", Namespace: "team"},
			})

			Expect(requests).To(BeNil())
		})
	})

	Describe("client failures", func() {
		It("returns an error when adding the finalizer fails", func() {
			providerConfig := createUnreadyProviderConfigNamed(ctx, "add-finalizer-error")
			reconciler.Client = &providerConfigErrorClient{
				Client: k8sClient, updateErr: errors.New("update failed"),
			}

			_, err := reconcileProviderConfig(ctx, reconciler, providerConfig)

			Expect(err).To(MatchError("update failed"))
			Expect(provider.cleanupCalled).To(BeFalse())
		})

		It("returns an error when references cannot be listed", func() {
			providerConfig := createUnreadyProviderConfigNamed(ctx, "reference-list-error")
			addProviderConfigFinalizer(ctx, providerConfig)
			Expect(k8sClient.Delete(ctx, providerConfig)).To(Succeed())
			reconciler.Client = &providerConfigErrorClient{
				Client: k8sClient, listErr: errors.New("list failed"),
			}

			_, err := reconcileProviderConfig(ctx, reconciler, providerConfig)

			Expect(err).To(MatchError("list failed"))
		})

		It("returns an error when removing the finalizer fails", func() {
			providerConfig := createUnreadyProviderConfigNamed(ctx, "remove-finalizer-error")
			addProviderConfigFinalizer(ctx, providerConfig)
			Expect(k8sClient.Delete(ctx, providerConfig)).To(Succeed())
			reconciler.Client = &providerConfigErrorClient{
				Client: k8sClient, updateErr: errors.New("update failed"),
			}

			_, err := reconcileProviderConfig(ctx, reconciler, providerConfig)

			Expect(err).To(MatchError("update failed"))
		})

		It("returns an error when status cannot be patched", func() {
			providerConfig := createUnreadyProviderConfigNamed(ctx, "status-patch-error")
			wrapped := &providerConfigErrorClient{
				Client: k8sClient, statusPatchErr: errors.New("status patch failed"),
			}
			reconciler.Client = wrapped

			_, err := reconcileProviderConfig(ctx, reconciler, providerConfig)

			Expect(err).To(MatchError("status patch failed"))
			Expect(wrapped.statusPatchCalls).To(Equal(1))
			Expect(provider.cleanupCalled).To(BeTrue())
		})

		It("retries a conflicting status patch", func() {
			providerConfig := createUnreadyProviderConfigNamed(ctx, "status-patch-conflict")
			wrapped := &providerConfigErrorClient{
				Client: k8sClient, statusPatchConflicts: 1,
			}
			reconciler.Client = wrapped

			_, err := reconcileProviderConfig(ctx, reconciler, providerConfig)

			Expect(err).NotTo(HaveOccurred())
			Expect(wrapped.statusPatchCalls).To(Equal(2))
			fetched := getProviderConfig(ctx, client.ObjectKeyFromObject(providerConfig))
			condition := meta.FindStatusCondition(fetched.Status.Conditions, conditions.TypeProviderConfigReady)
			Expect(condition.Status).To(Equal(metav1.ConditionTrue))
		})
	})
})

type providerConfigTestProvider struct {
	*fakeProvider
	validation   validation.ValidationResult
	cleanupCalls int
}

func (p *providerConfigTestProvider) ValidateProviderConfigSpec(
	vedro.ProviderConfig,
) validation.ValidationResult {
	return p.validation
}

func (p *providerConfigTestProvider) Cleanup(ctx context.Context) error {
	p.cleanupCalls++
	return p.fakeProvider.Cleanup(ctx)
}

func createProviderConfig(ctx context.Context) {
	createProviderConfigNamed(ctx, "test-provider")
}

// updateUsagePolicy simulates a policy edit followed by successful provider reconciliation.
func updateUsagePolicy(ctx context.Context, mutate func(*vedro.UsagePolicySpec)) {
	provider := &vedro.ProviderConfig{}
	Expect(k8sClient.Get(ctx, client.ObjectKey{Name: "test-provider"}, provider)).To(Succeed())
	mutate(&provider.Spec.UsagePolicy)
	Expect(k8sClient.Update(ctx, provider)).To(Succeed())
	markProviderConfigReady(ctx, provider)
}

func createProviderConfigNamed(ctx context.Context, name string) {
	providerConfig := createUnreadyProviderConfigNamed(ctx, name)
	markProviderConfigReady(ctx, providerConfig)
}

func newProviderConfig(name string) *vedro.ProviderConfig {
	return &vedro.ProviderConfig{
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
			UsagePolicy: vedro.UsagePolicySpec{
				AllowedNamespaces: vedro.AllowedNamespacesSpec{
					All: true,
				},
				BucketPolicy: vedro.BucketPolicySpec{
					AllowedNamePatterns: []string{
						".*",
					},
				},
				PrincipalPolicy: vedro.PrincipalPolicySpec{
					AllowManaged:    true,
					AllowReferences: true,
					AllowedNamePatterns: []string{
						".*",
					},
					AllowedReferencePatterns: []string{
						".*",
					},
				},
			},
		},
	}
}

func createUnreadyProviderConfigNamed(ctx context.Context, name string) *vedro.ProviderConfig {
	providerConfig := newProviderConfig(name)

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

func newProviderConfigWatchClient(
	providerConfigs ...*vedro.ProviderConfig,
) client.Client {
	objects := make([]client.Object, 0, len(providerConfigs))
	for _, providerConfig := range providerConfigs {
		objects = append(objects, providerConfig)
	}

	return fake.NewClientBuilder().
		WithScheme(k8sClient.Scheme()).
		WithObjects(objects...).
		WithIndex(
			&vedro.ProviderConfig{},
			providerConfigSecretRefIndex,
			func(obj client.Object) []string {
				providerConfig := obj.(*vedro.ProviderConfig)
				if providerConfig.Spec.CredentialsSecretRef == nil ||
					providerConfig.Spec.CredentialsSecretRef.Name == "" {
					return nil
				}
				return []string{types.NamespacedName{
					Name:      providerConfig.Spec.CredentialsSecretRef.Name,
					Namespace: providerConfig.Spec.CredentialsSecretRef.Namespace,
				}.String()}
			},
		).
		Build()
}

type providerConfigErrorClient struct {
	client.Client
	updateErr            error
	listErr              error
	statusPatchErr       error
	statusPatchConflicts int
	statusPatchCalls     int
}

func (c *providerConfigErrorClient) Update(
	ctx context.Context,
	obj client.Object,
	opts ...client.UpdateOption,
) error {
	if c.updateErr != nil {
		return c.updateErr
	}
	return c.Client.Update(ctx, obj, opts...)
}

func (c *providerConfigErrorClient) List(
	ctx context.Context,
	list client.ObjectList,
	opts ...client.ListOption,
) error {
	if c.listErr != nil {
		return c.listErr
	}
	return c.Client.List(ctx, list, opts...)
}

func (c *providerConfigErrorClient) Status() client.SubResourceWriter {
	return &providerConfigStatusWriter{
		SubResourceWriter: c.Client.Status(),
		client:            c,
	}
}

type providerConfigStatusWriter struct {
	client.SubResourceWriter
	client *providerConfigErrorClient
}

func (w *providerConfigStatusWriter) Patch(
	ctx context.Context,
	obj client.Object,
	patch client.Patch,
	opts ...client.SubResourcePatchOption,
) error {
	w.client.statusPatchCalls++
	if w.client.statusPatchConflicts > 0 {
		w.client.statusPatchConflicts--
		return apierrors.NewConflict(
			schema.GroupResource{Group: vedro.GroupVersion.Group, Resource: "providerconfigs"},
			obj.GetName(),
			errors.New("conflict"),
		)
	}
	if w.client.statusPatchErr != nil {
		return w.client.statusPatchErr
	}
	return w.SubResourceWriter.Patch(ctx, obj, patch, opts...)
}
