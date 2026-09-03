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

var _ = Describe("CloudPrincipalReconciler", func() {
	var (
		reconciler *CloudPrincipalReconciler
		provider   *fakeProvider
	)

	BeforeEach(func() {
		provider = &fakeProvider{
			capabilities: cloud.Capabilities{
				Principal: cloud.PrincipalCapabilities{
					ManagedKinds: map[vedro.PrincipalKind]bool{
						vedro.PrincipalKindServiceAccount: true,
					},
					ReferencedKinds: map[vedro.PrincipalKind]bool{
						vedro.PrincipalKindServiceAccount: true,
					},
				},
			},
			principal: &fakePrincipalProvider{
				validateResult: validation.Valid(),
				ensureResult: &cloud.PrincipalAttrs{
					Name: "external-principal",
					Id:   "principal-id",
				},
			},
		}

		reconciler = &CloudPrincipalReconciler{
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

	It("ignores missing CloudPrincipals", func() {
		result, err := reconciler.Reconcile(ctx, reconcile.Request{
			NamespacedName: types.NamespacedName{
				Name:      "missing-principal",
				Namespace: "default",
			},
		})

		Expect(err).NotTo(HaveOccurred())
		Expect(result).To(Equal(reconcile.Result{}))
	})

	It("adds the finalizer and marks ProviderConfig missing", func() {
		principal := createCloudPrincipal(ctx, "missing-provider", func(p *vedro.CloudPrincipal) {
			p.Spec.ProviderRef.Name = "missing-provider"
		})

		result, err := reconcileCloudPrincipal(ctx, reconciler, principal)

		Expect(err).NotTo(HaveOccurred())
		Expect(result).To(Equal(reconcile.Result{}))

		fetched := getCloudPrincipal(ctx, client.ObjectKeyFromObject(principal))
		Expect(fetched.Finalizers).To(ContainElement(principalFinalizer))
		Expect(fetched.Status.ObservedGeneration).To(Equal(fetched.Generation))

		providerCondition := meta.FindStatusCondition(fetched.Status.Conditions, conditions.TypeProviderConfigReady)
		Expect(providerCondition).NotTo(BeNil())
		Expect(providerCondition.Status).To(Equal(metav1.ConditionFalse))
		Expect(providerCondition.Reason).To(Equal(conditions.ReasonProviderConfigNotFound))

		readyCondition := meta.FindStatusCondition(fetched.Status.Conditions, conditions.TypeReady)
		Expect(readyCondition).NotTo(BeNil())
		Expect(readyCondition.Status).To(Equal(metav1.ConditionFalse))
		Expect(readyCondition.Reason).To(Equal(conditions.ReasonProviderConfigNotFound))
	})

	It("records provider factory errors", func() {
		principal := createCloudPrincipal(ctx, "provider-factory-error")
		createProviderConfig(ctx)
		reconciler.ProviderFactory = func(
			ctx context.Context,
			cfg vedro.ProviderConfig,
			kubeClient client.Client,
		) (cloud.Provider, error) {
			return nil, errors.New("provider setup failed")
		}

		_, err := reconcileCloudPrincipal(ctx, reconciler, principal)

		Expect(err).To(HaveOccurred())

		fetched := getCloudPrincipal(ctx, client.ObjectKeyFromObject(principal))
		providerCondition := meta.FindStatusCondition(fetched.Status.Conditions, conditions.TypeProviderConfigReady)
		Expect(providerCondition).NotTo(BeNil())
		Expect(providerCondition.Status).To(Equal(metav1.ConditionFalse))
		Expect(providerCondition.Reason).To(Equal(conditions.ReasonProviderConfigError))
		Expect(providerCondition.Message).To(Equal("provider setup failed"))

		readyCondition := meta.FindStatusCondition(fetched.Status.Conditions, conditions.TypeReady)
		Expect(readyCondition).NotTo(BeNil())
		Expect(readyCondition.Status).To(Equal(metav1.ConditionFalse))
		Expect(readyCondition.Reason).To(Equal(conditions.ReasonProviderConfigError))
		Expect(provider.principal.ensureCalls).To(Equal(0))
		Expect(provider.cleanupCalled).To(BeFalse())
	})

	It("records invalid CloudPrincipal specs without ensuring the external principal", func() {
		principal := createCloudPrincipal(ctx, "invalid-spec")
		createProviderConfig(ctx)
		provider.principal.validateResult = validation.Invalid("invalid principal name")

		result, err := reconcileCloudPrincipal(ctx, reconciler, principal)

		Expect(err).NotTo(HaveOccurred())
		Expect(result).To(Equal(reconcile.Result{}))

		fetched := getCloudPrincipal(ctx, client.ObjectKeyFromObject(principal))
		condition := meta.FindStatusCondition(fetched.Status.Conditions, conditions.TypeReady)
		Expect(condition).NotTo(BeNil())
		Expect(condition.Status).To(Equal(metav1.ConditionFalse))
		Expect(condition.Reason).To(Equal(conditions.ReasonCloudPrincipalInvalidSpec))
		Expect(condition.Message).To(Equal("invalid principal name"))
		Expect(provider.principal.ensureCalls).To(Equal(0))
		Expect(provider.cleanupCalled).To(BeTrue())
	})

	It("sets successful CloudPrincipal status after ensuring the external principal", func() {
		principal := createCloudPrincipal(ctx, "successful-reconcile")
		createProviderConfig(ctx)

		result, err := reconcileCloudPrincipal(ctx, reconciler, principal)

		Expect(err).NotTo(HaveOccurred())
		Expect(result).To(Equal(reconcile.Result{}))

		fetched := getCloudPrincipal(ctx, client.ObjectKeyFromObject(principal))
		Expect(fetched.Status.ExternalName).To(Equal("external-principal"))
		Expect(fetched.Status.ExternalId).To(Equal("principal-id"))
		Expect(fetched.Status.ObservedProvider).To(Equal("test-provider"))
		Expect(fetched.Status.ObservedGeneration).To(Equal(fetched.Generation))

		providerCondition := meta.FindStatusCondition(fetched.Status.Conditions, conditions.TypeProviderConfigReady)
		Expect(providerCondition).NotTo(BeNil())
		Expect(providerCondition.Status).To(Equal(metav1.ConditionTrue))
		Expect(providerCondition.Reason).To(Equal(conditions.ReasonProviderConfigReconciled))

		readyCondition := meta.FindStatusCondition(fetched.Status.Conditions, conditions.TypeReady)
		Expect(readyCondition).NotTo(BeNil())
		Expect(readyCondition.Status).To(Equal(metav1.ConditionTrue))
		Expect(readyCondition.Reason).To(Equal(conditions.ReasonCloudPrincipalReconciled))
		Expect(provider.principal.ensureCalls).To(Equal(1))
		Expect(provider.cleanupCalled).To(BeTrue())
	})

	It("records ensure errors", func() {
		principal := createCloudPrincipal(ctx, "ensure-error")
		createProviderConfig(ctx)
		provider.principal.ensureErr = errors.New("ensure failed")

		result, err := reconcileCloudPrincipal(ctx, reconciler, principal)

		Expect(err).To(MatchError("ensure failed"))
		Expect(result).To(Equal(reconcile.Result{}))

		fetched := getCloudPrincipal(ctx, client.ObjectKeyFromObject(principal))
		condition := meta.FindStatusCondition(fetched.Status.Conditions, conditions.TypeReady)
		Expect(condition).NotTo(BeNil())
		Expect(condition.Status).To(Equal(metav1.ConditionFalse))
		Expect(condition.Reason).To(Equal(conditions.ReasonCloudPrincipalEnsureError))
		Expect(condition.Message).To(Equal("ensure failed"))
		Expect(provider.principal.ensureCalls).To(Equal(1))
		Expect(provider.cleanupCalled).To(BeTrue())
	})

	It("does not fail reconcile when provider cleanup fails", func() {
		principal := createCloudPrincipal(ctx, "cleanup-error")
		createProviderConfig(ctx)
		provider.cleanupErr = errors.New("cleanup failed")

		result, err := reconcileCloudPrincipal(ctx, reconciler, principal)

		Expect(err).NotTo(HaveOccurred())
		Expect(result).To(Equal(reconcile.Result{}))
		Expect(provider.cleanupCalled).To(BeTrue())

		fetched := getCloudPrincipal(ctx, client.ObjectKeyFromObject(principal))
		condition := meta.FindStatusCondition(fetched.Status.Conditions, conditions.TypeReady)
		Expect(condition).NotTo(BeNil())
		Expect(condition.Status).To(Equal(metav1.ConditionTrue))
		Expect(condition.Reason).To(Equal(conditions.ReasonCloudPrincipalReconciled))
	})

	It("retains the external principal and removes the finalizer for Retain policy", func() {
		principal := createCloudPrincipal(ctx, "retain-policy")
		addPrincipalFinalizer(ctx, principal)
		Expect(k8sClient.Delete(ctx, principal)).To(Succeed())

		result, err := reconcileCloudPrincipal(ctx, reconciler, principal)

		Expect(err).NotTo(HaveOccurred())
		Expect(result).To(Equal(reconcile.Result{}))
		Expect(provider.principal.deleteCalls).To(Equal(0))
		expectCloudPrincipalNotFound(ctx, client.ObjectKeyFromObject(principal))
	})

	It("deletes the external principal and removes the finalizer for Delete policy", func() {
		principal := createCloudPrincipal(ctx, "delete-policy", func(p *vedro.CloudPrincipal) {
			p.Spec.Managed.DeletionPolicy = vedro.DeletionPolicyDelete
		})
		createProviderConfig(ctx)
		addPrincipalFinalizer(ctx, principal)
		Expect(k8sClient.Delete(ctx, principal)).To(Succeed())

		result, err := reconcileCloudPrincipal(ctx, reconciler, principal)

		Expect(err).NotTo(HaveOccurred())
		Expect(result).To(Equal(reconcile.Result{}))
		Expect(provider.principal.deleteCalls).To(Equal(1))
		Expect(provider.cleanupCalled).To(BeTrue())
		expectCloudPrincipalNotFound(ctx, client.ObjectKeyFromObject(principal))
	})

	It("uses the observed ProviderConfig when deleting a CloudPrincipal", func() {
		principal := createCloudPrincipal(ctx, "observed-provider", func(p *vedro.CloudPrincipal) {
			p.Spec.Managed.DeletionPolicy = vedro.DeletionPolicyDelete
		})
		createProviderConfigNamed(ctx, "observed-provider")
		fetched := getCloudPrincipal(ctx, client.ObjectKeyFromObject(principal))
		fetched.Status.ObservedProvider = "observed-provider"
		Expect(k8sClient.Status().Update(ctx, fetched)).To(Succeed())
		addPrincipalFinalizer(ctx, principal)

		var configuredProvider string
		reconciler.ProviderFactory = func(
			_ context.Context,
			cfg vedro.ProviderConfig,
			_ client.Client,
		) (cloud.Provider, error) {
			configuredProvider = cfg.Name
			return provider, nil
		}
		Expect(k8sClient.Delete(ctx, principal)).To(Succeed())

		_, err := reconcileCloudPrincipal(ctx, reconciler, principal)

		Expect(err).NotTo(HaveOccurred())
		Expect(configuredProvider).To(Equal("observed-provider"))
		Expect(provider.principal.deleteCalls).To(Equal(1))
		expectCloudPrincipalNotFound(ctx, client.ObjectKeyFromObject(principal))
	})

	It("records delete errors and requeues the CloudPrincipal", func() {
		principal := createCloudPrincipal(ctx, "delete-error", func(p *vedro.CloudPrincipal) {
			p.Spec.Managed.DeletionPolicy = vedro.DeletionPolicyDelete
		})
		createProviderConfig(ctx)
		provider.principal.deleteErr = errors.New("delete failed")
		addPrincipalFinalizer(ctx, principal)
		Expect(k8sClient.Delete(ctx, principal)).To(Succeed())

		_, err := reconcileCloudPrincipal(ctx, reconciler, principal)

		Expect(err).To(MatchError("delete failed"))
		Expect(provider.principal.deleteCalls).To(Equal(1))
		Expect(provider.cleanupCalled).To(BeTrue())

		fetched := getCloudPrincipal(ctx, client.ObjectKeyFromObject(principal))
		Expect(fetched.Finalizers).To(ContainElement(principalFinalizer))
		condition := meta.FindStatusCondition(fetched.Status.Conditions, conditions.TypeReady)
		Expect(condition).NotTo(BeNil())
		Expect(condition.Status).To(Equal(metav1.ConditionFalse))
		Expect(condition.Reason).To(Equal(conditions.ReasonCloudPrincipalDeleteError))
		Expect(condition.Message).To(Equal("delete failed"))
	})
})

type fakePrincipalProvider struct {
	validateResult validation.ValidationResult
	ensureResult   *cloud.PrincipalAttrs
	ensureErr      error
	deleteErr      error
	ensureCalls    int
	deleteCalls    int
}

func (p *fakePrincipalProvider) ValidatePrincipalSpec(
	principal vedro.CloudPrincipal,
) validation.ValidationResult {
	return p.validateResult
}

func (p *fakePrincipalProvider) EnsurePrincipal(
	ctx context.Context,
	principal vedro.CloudPrincipal,
) (*cloud.PrincipalAttrs, error) {
	p.ensureCalls++
	return p.ensureResult, p.ensureErr
}

func (p *fakePrincipalProvider) DeletePrincipal(
	ctx context.Context,
	principal vedro.CloudPrincipal,
) error {
	p.deleteCalls++
	return p.deleteErr
}

func createCloudPrincipal(
	ctx context.Context,
	name string,
	mutators ...func(*vedro.CloudPrincipal),
) *vedro.CloudPrincipal {
	principal := &vedro.CloudPrincipal{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "vedro.svetoch.dev/v1alpha1",
			Kind:       "CloudPrincipal",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: "default",
		},
		Spec: vedro.CloudPrincipalSpec{
			ProviderRef:      vedro.ProviderConfigReference{Name: "test-provider"},
			Kind:             vedro.PrincipalKindServiceAccount,
			ManagementPolicy: vedro.PrincipalManagementPolicyManaged,
			Managed: &vedro.ManagedPrincipalSpec{
				Name:           name,
				DeletionPolicy: vedro.DeletionPolicyRetain,
			},
		},
	}

	for _, mutate := range mutators {
		mutate(principal)
	}

	Expect(k8sClient.Create(ctx, principal)).To(Succeed())
	DeferCleanup(func() {
		cleanupCloudPrincipal(ctx, client.ObjectKeyFromObject(principal))
	})

	return principal
}

func reconcileCloudPrincipal(
	ctx context.Context,
	reconciler *CloudPrincipalReconciler,
	principal *vedro.CloudPrincipal,
) (reconcile.Result, error) {
	return reconciler.Reconcile(ctx, reconcile.Request{
		NamespacedName: client.ObjectKeyFromObject(principal),
	})
}

func getCloudPrincipal(ctx context.Context, key client.ObjectKey) *vedro.CloudPrincipal {
	principal := &vedro.CloudPrincipal{}
	Expect(k8sClient.Get(ctx, key, principal)).To(Succeed())
	return principal
}

func addPrincipalFinalizer(ctx context.Context, principal *vedro.CloudPrincipal) {
	fetched := getCloudPrincipal(ctx, client.ObjectKeyFromObject(principal))
	fetched.Finalizers = append(fetched.Finalizers, principalFinalizer)
	Expect(k8sClient.Update(ctx, fetched)).To(Succeed())
}

func expectCloudPrincipalNotFound(ctx context.Context, key client.ObjectKey) {
	err := k8sClient.Get(ctx, key, &vedro.CloudPrincipal{})
	Expect(apierrors.IsNotFound(err)).To(BeTrue())
}

func cleanupCloudPrincipal(ctx context.Context, key client.ObjectKey) {
	principal := &vedro.CloudPrincipal{}
	err := k8sClient.Get(ctx, key, principal)
	if apierrors.IsNotFound(err) {
		return
	}
	Expect(err).NotTo(HaveOccurred())

	principal.Finalizers = nil
	Expect(k8sClient.Update(ctx, principal)).To(Succeed())
	err = k8sClient.Delete(ctx, principal)
	if err != nil {
		Expect(apierrors.IsNotFound(err)).To(BeTrue())
	}
}
