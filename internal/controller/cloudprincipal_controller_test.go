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
					Name:   "external-principal",
					Kind:   vedro.PrincipalKindServiceAccount,
					Policy: vedro.PrincipalManagementPolicyManaged,
					Id:     "principal-id",
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

	It("rejects management policy changes", func() {
		principal := createCloudPrincipal(ctx, "immutable-management-policy")
		principal.Spec.ManagementPolicy = vedro.PrincipalManagementPolicyReference
		principal.Spec.Managed = nil
		principal.Spec.Reference = &vedro.ReferencedPrincipalSpec{Name: "referenced-principal"}

		err := k8sClient.Update(ctx, principal)

		Expect(apierrors.IsInvalid(err)).To(BeTrue())
		Expect(err).To(MatchError(ContainSubstring("managementPolicy is immutable")))
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

	It("reconciles an AllUsers Reference principal without a reference set", func() {
		provider.capabilities.Principal.ReferencedKinds[vedro.PrincipalKindAllUsers] = true
		provider.principal.ensureResult = &cloud.PrincipalAttrs{
			Id:     "allUsers",
			Kind:   vedro.PrincipalKindAllUsers,
			Policy: vedro.PrincipalManagementPolicyReference,
		}
		principal := createCloudPrincipal(ctx, "all-users-reconcile", func(p *vedro.CloudPrincipal) {
			p.Spec.Kind = vedro.PrincipalKindAllUsers
			p.Spec.ManagementPolicy = vedro.PrincipalManagementPolicyReference
			p.Spec.Managed = nil
		})

		createProviderConfig(ctx)

		result, err := reconcileCloudPrincipal(ctx, reconciler, principal)

		Expect(err).NotTo(HaveOccurred())
		Expect(result).To(Equal(reconcile.Result{}))
		Expect(provider.principal.ensureCalls).To(Equal(1))

		fetched := getCloudPrincipal(ctx, client.ObjectKeyFromObject(principal))
		Expect(fetched.Status.ExternalId).To(Equal("allUsers"))
		Expect(fetched.Status.Kind).To(Equal(vedro.PrincipalKindAllUsers))
		Expect(fetched.Status.ManagementPolicy).To(Equal(
			vedro.PrincipalManagementPolicyReference,
		))

		readyCondition := meta.FindStatusCondition(fetched.Status.Conditions, conditions.TypeReady)
		Expect(readyCondition).NotTo(BeNil())
		Expect(readyCondition.Status).To(Equal(metav1.ConditionTrue))
		Expect(readyCondition.Reason).To(Equal(conditions.ReasonCloudPrincipalReconciled))
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

	DescribeTable("rejects managed principals restricted by usage policy",
		func(restrict func(*vedro.UsagePolicySpec)) {
			principal := createCloudPrincipal(ctx, "restricted-principal")
			createProviderConfig(ctx)
			updateUsagePolicy(ctx, restrict)
			_, err := reconcileCloudPrincipal(ctx, reconciler, principal)
			Expect(err).NotTo(HaveOccurred())
			Expect(provider.principal.ensureCalls).To(BeZero())
			fetched := getCloudPrincipal(ctx, client.ObjectKeyFromObject(principal))
			condition := meta.FindStatusCondition(fetched.Status.Conditions, conditions.TypeReady)
			Expect(condition).NotTo(BeNil())
			Expect(condition.Status).To(Equal(metav1.ConditionFalse))
			Expect(condition.Reason).To(Equal(conditions.ReasonCloudPrincipalSpecRestricted))
			Expect(condition.Message).NotTo(BeEmpty())
			Expect(condition.ObservedGeneration).To(Equal(fetched.Generation))
		},
		Entry("namespace", func(p *vedro.UsagePolicySpec) {
			p.AllowedNamespaces = vedro.AllowedNamespacesSpec{Names: []string{"other"}}
		}),
		Entry("name", func(p *vedro.UsagePolicySpec) { p.PrincipalPolicy.AllowedNamePatterns = []string{"^other$"} }),
		Entry("kind", func(p *vedro.UsagePolicySpec) {
			p.PrincipalPolicy.AllowedKinds = []vedro.PrincipalKind{vedro.PrincipalKindUser}
		}),
		Entry("management disabled", func(p *vedro.UsagePolicySpec) { p.PrincipalPolicy.AllowManaged = false }),
	)

	DescribeTable("preserves provisioned principals when recorded deletion targets are restricted",
		func(restrictKind bool) {
			principal := createCloudPrincipal(ctx, "restricted-delete", func(p *vedro.CloudPrincipal) { p.Spec.Managed.DeletionPolicy = vedro.DeletionPolicyDelete })
			createProviderConfig(ctx)
			_, err := reconcileCloudPrincipal(ctx, reconciler, principal)
			Expect(err).NotTo(HaveOccurred())
			fetched := getCloudPrincipal(ctx, client.ObjectKeyFromObject(principal))
			if restrictKind {
				fetched.Spec.Kind = vedro.PrincipalKindUser
			} else {
				fetched.Spec.Managed.Name = "allowed-name"
			}
			Expect(k8sClient.Update(ctx, fetched)).To(Succeed())
			updateUsagePolicy(ctx, func(p *vedro.UsagePolicySpec) {
				if restrictKind {
					p.PrincipalPolicy.AllowedKinds = []vedro.PrincipalKind{vedro.PrincipalKindUser}
				} else {
					p.PrincipalPolicy.AllowedNamePatterns = []string{"^allowed-name$"}
				}
			})
			Expect(k8sClient.Delete(ctx, fetched)).To(Succeed())
			_, err = reconcileCloudPrincipal(ctx, reconciler, principal)
			Expect(err).NotTo(HaveOccurred())
			Expect(provider.principal.deleteCalls).To(BeZero())
			fetched = getCloudPrincipal(ctx, client.ObjectKeyFromObject(principal))
			Expect(fetched.Finalizers).To(ContainElement(principalFinalizer))
			Expect(meta.FindStatusCondition(fetched.Status.Conditions, conditions.TypeReady).Reason).To(Equal(conditions.ReasonCloudPrincipalRemoveFinalizerError))
			updateUsagePolicy(ctx, func(p *vedro.UsagePolicySpec) {
				p.PrincipalPolicy.AllowedKinds = []vedro.PrincipalKind{vedro.PrincipalKindServiceAccount}
				p.PrincipalPolicy.AllowedNamePatterns = []string{"^external-principal$"}
			})
			_, err = reconcileCloudPrincipal(ctx, reconciler, principal)
			Expect(err).NotTo(HaveOccurred())
			Expect(provider.principal.deleteCalls).To(Equal(1))
			expectCloudPrincipalNotFound(ctx, client.ObjectKeyFromObject(principal))
		},
		Entry("recorded kind", true), Entry("recorded name", false),
	)

	It("removes a rejected unprovisioned principal without calling cloud deletion", func() {
		principal := createCloudPrincipal(ctx, "rejected-delete", func(p *vedro.CloudPrincipal) { p.Spec.Managed.DeletionPolicy = vedro.DeletionPolicyDelete })
		createProviderConfig(ctx)
		updateUsagePolicy(ctx, func(p *vedro.UsagePolicySpec) { p.PrincipalPolicy.AllowManaged = false })
		_, err := reconcileCloudPrincipal(ctx, reconciler, principal)
		Expect(err).NotTo(HaveOccurred())
		Expect(provider.principal.ensureCalls).To(BeZero())
		Expect(k8sClient.Delete(ctx, principal)).To(Succeed())
		_, err = reconcileCloudPrincipal(ctx, reconciler, principal)
		Expect(err).NotTo(HaveOccurred())
		Expect(provider.principal.deleteCalls).To(BeZero())
		expectCloudPrincipalNotFound(ctx, client.ObjectKeyFromObject(principal))
	})

	It("deletes the external principal and removes the finalizer for Delete policy", func() {
		principal := createCloudPrincipal(ctx, "delete-policy", func(p *vedro.CloudPrincipal) {
			p.Spec.Managed.DeletionPolicy = vedro.DeletionPolicyDelete
		})
		createProviderConfig(ctx)
		result, err := reconcileCloudPrincipal(ctx, reconciler, principal)

		Expect(err).NotTo(HaveOccurred())
		Expect(result).To(Equal(reconcile.Result{}))

		Expect(k8sClient.Delete(ctx, principal)).To(Succeed())

		result, err = reconcileCloudPrincipal(ctx, reconciler, principal)

		Expect(err).NotTo(HaveOccurred())
		Expect(result).To(Equal(reconcile.Result{}))
		Expect(provider.principal.deleteCalls).To(Equal(1))
		Expect(provider.cleanupCalled).To(BeTrue())
		expectCloudPrincipalNotFound(ctx, client.ObjectKeyFromObject(principal))
	})

	It("uses the observed ProviderConfig when deleting a CloudPrincipal", func() {
		principal := createCloudPrincipal(ctx, "observed-provider", func(p *vedro.CloudPrincipal) {
			p.Spec.ProviderRef.Name = "observed-provider"
			p.Spec.Managed.DeletionPolicy = vedro.DeletionPolicyDelete
		})
		createProviderConfigNamed(ctx, "observed-provider")
		_, err := reconcileCloudPrincipal(ctx, reconciler, principal)

		Expect(err).NotTo(HaveOccurred())

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

		_, err = reconcileCloudPrincipal(ctx, reconciler, principal)

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
		_, err := reconcileCloudPrincipal(ctx, reconciler, principal)
		Expect(err).NotTo(HaveOccurred())

		Expect(k8sClient.Delete(ctx, principal)).To(Succeed())

		provider.principal.deleteErr = errors.New("delete failed")

		_, err = reconcileCloudPrincipal(ctx, reconciler, principal)

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

	It("rejects a CloudPrincipal with both managed and reference set", func() {
		principal := &vedro.CloudPrincipal{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "vedro.svetoch.dev/v1alpha1",
				Kind:       "CloudPrincipal",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      "managed-and-reference",
				Namespace: "default",
			},
			Spec: vedro.CloudPrincipalSpec{
				ProviderRef:      vedro.ProviderConfigReference{Name: "test-provider"},
				Kind:             vedro.PrincipalKindServiceAccount,
				ManagementPolicy: vedro.PrincipalManagementPolicyManaged,
				Managed:          &vedro.ManagedPrincipalSpec{Name: "managed-principal"},
				Reference:        &vedro.ReferencedPrincipalSpec{Name: "referenced-principal"},
			},
		}

		err := k8sClient.Create(ctx, principal)

		Expect(apierrors.IsInvalid(err)).To(BeTrue())
		Expect(err).To(MatchError(ContainSubstring(
			"managed must be set and reference must not be set",
		)))
	})

	It("rejects a Reference principal without a reference set", func() {
		principal := &vedro.CloudPrincipal{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "vedro.svetoch.dev/v1alpha1",
				Kind:       "CloudPrincipal",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      "reference-without-reference",
				Namespace: "default",
			},
			Spec: vedro.CloudPrincipalSpec{
				ProviderRef:      vedro.ProviderConfigReference{Name: "test-provider"},
				Kind:             vedro.PrincipalKindServiceAccount,
				ManagementPolicy: vedro.PrincipalManagementPolicyReference,
			},
		}

		err := k8sClient.Create(ctx, principal)

		Expect(apierrors.IsInvalid(err)).To(BeTrue())
		Expect(err).To(MatchError(ContainSubstring(
			"reference must be set unless kind is AllUsers, and managed must not be set",
		)))
	})

	It("allows an AllUsers Reference principal without a reference set", func() {
		principal := &vedro.CloudPrincipal{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "vedro.svetoch.dev/v1alpha1",
				Kind:       "CloudPrincipal",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      "all-users-without-reference",
				Namespace: "default",
			},
			Spec: vedro.CloudPrincipalSpec{
				ProviderRef:      vedro.ProviderConfigReference{Name: "test-provider"},
				Kind:             vedro.PrincipalKindAllUsers,
				ManagementPolicy: vedro.PrincipalManagementPolicyReference,
			},
		}

		Expect(k8sClient.Create(ctx, principal)).To(Succeed())
		DeferCleanup(func() {
			cleanupCloudPrincipal(ctx, client.ObjectKeyFromObject(principal))
		})
	})

	It("rejects a Managed principal without a managed set", func() {
		principal := &vedro.CloudPrincipal{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "vedro.svetoch.dev/v1alpha1",
				Kind:       "CloudPrincipal",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      "managed-without-managed",
				Namespace: "default",
			},
			Spec: vedro.CloudPrincipalSpec{
				ProviderRef:      vedro.ProviderConfigReference{Name: "test-provider"},
				Kind:             vedro.PrincipalKindServiceAccount,
				ManagementPolicy: vedro.PrincipalManagementPolicyManaged,
			},
		}

		err := k8sClient.Create(ctx, principal)

		Expect(apierrors.IsInvalid(err)).To(BeTrue())
		Expect(err).To(MatchError(ContainSubstring(
			"managed must be set and reference must not be set",
		)))
	})

	It("records unsupported features without ensuring the external principal", func() {
		principal := createCloudPrincipal(ctx, "unsupported-features", func(p *vedro.CloudPrincipal) {
			p.Spec.Kind = vedro.PrincipalKindUser
		})
		createProviderConfig(ctx)

		result, err := reconcileCloudPrincipal(ctx, reconciler, principal)

		Expect(err).NotTo(HaveOccurred())
		Expect(result).To(Equal(reconcile.Result{}))

		fetched := getCloudPrincipal(ctx, client.ObjectKeyFromObject(principal))
		Expect(fetched.Status.UnsupportedFeatures).To(HaveLen(1))
		Expect(fetched.Status.UnsupportedFeatures[0].Reason).To(Equal(
			vedro.PrincipalUnsupportedManagedUser,
		))

		condition := meta.FindStatusCondition(fetched.Status.Conditions, conditions.TypeReady)
		Expect(condition).NotTo(BeNil())
		Expect(condition.Status).To(Equal(metav1.ConditionFalse))
		Expect(condition.Reason).To(Equal(conditions.ReasonCloudPrincipalUnsupportedFeatures))
		Expect(provider.principal.ensureCalls).To(Equal(0))
		Expect(provider.cleanupCalled).To(BeTrue())
	})

	It("clears unsupported features when the provider gains support", func() {
		principal := createCloudPrincipal(ctx, "unsupported-then-supported", func(p *vedro.CloudPrincipal) {
			p.Spec.Kind = vedro.PrincipalKindUser
		})
		createProviderConfig(ctx)

		_, err := reconcileCloudPrincipal(ctx, reconciler, principal)
		Expect(err).NotTo(HaveOccurred())

		fetched := getCloudPrincipal(ctx, client.ObjectKeyFromObject(principal))
		Expect(fetched.Status.UnsupportedFeatures).To(HaveLen(1))

		provider.capabilities.Principal.ManagedKinds[vedro.PrincipalKindUser] = true
		_, err = reconcileCloudPrincipal(ctx, reconciler, principal)
		Expect(err).NotTo(HaveOccurred())

		fetched = getCloudPrincipal(ctx, client.ObjectKeyFromObject(principal))
		Expect(fetched.Status.UnsupportedFeatures).To(BeEmpty())
		condition := meta.FindStatusCondition(fetched.Status.Conditions, conditions.TypeReady)
		Expect(condition).NotTo(BeNil())
		Expect(condition.Status).To(Equal(metav1.ConditionTrue))
		Expect(provider.principal.ensureCalls).To(Equal(1))
	})

	DescribeTable("enforces reference policy before resolving the external principal",
		func(disableReferences bool) {
			principal := createCloudPrincipal(ctx, "restricted-reference", func(p *vedro.CloudPrincipal) {
				p.Spec.ManagementPolicy = vedro.PrincipalManagementPolicyReference
				p.Spec.Managed = nil
				p.Spec.Reference = &vedro.ReferencedPrincipalSpec{Name: "account@example.com"}
			})
			createProviderConfig(ctx)
			updateUsagePolicy(ctx, func(p *vedro.UsagePolicySpec) {
				if disableReferences {
					p.PrincipalPolicy.AllowReferences = false
				} else {
					p.PrincipalPolicy.AllowedReferencePatterns = []string{`^[^@]+@other\.com$`}
				}
			})
			_, err := reconcileCloudPrincipal(ctx, reconciler, principal)
			Expect(err).NotTo(HaveOccurred())
			Expect(provider.principal.ensureCalls).To(BeZero())
			fetched := getCloudPrincipal(ctx, client.ObjectKeyFromObject(principal))
			condition := meta.FindStatusCondition(fetched.Status.Conditions, conditions.TypeReady)
			Expect(condition).NotTo(BeNil())
			Expect(condition.Status).To(Equal(metav1.ConditionFalse))
			Expect(condition.Reason).To(Equal(conditions.ReasonCloudPrincipalSpecRestricted))
		},
		Entry("references disabled", true), Entry("reference name denied", false),
	)

	It("reconciles referenced principals", func() {
		principal := createCloudPrincipal(ctx, "referenced-principal", func(p *vedro.CloudPrincipal) {
			p.Spec.ManagementPolicy = vedro.PrincipalManagementPolicyReference
			p.Spec.Managed = nil
			p.Spec.Reference = &vedro.ReferencedPrincipalSpec{Name: "referenced-principal"}
			provider.principal.ensureResult = &cloud.PrincipalAttrs{
				Name:   "referenced-principal",
				Id:     "serviceAccount:principal-id",
				Kind:   vedro.PrincipalKindServiceAccount,
				Policy: vedro.PrincipalManagementPolicyReference,
			}
		})
		createProviderConfig(ctx)

		result, err := reconcileCloudPrincipal(ctx, reconciler, principal)

		Expect(err).NotTo(HaveOccurred())
		Expect(result).To(Equal(reconcile.Result{}))

		fetched := getCloudPrincipal(ctx, client.ObjectKeyFromObject(principal))
		Expect(fetched.Status.ExternalName).To(Equal("referenced-principal"))
		Expect(fetched.Status.ExternalId).To(Equal("serviceAccount:principal-id"))
		Expect(fetched.Status.Kind).To(Equal(vedro.PrincipalKindServiceAccount))
		Expect(fetched.Status.ManagementPolicy).To(Equal(vedro.PrincipalManagementPolicyReference))

		readyCondition := meta.FindStatusCondition(fetched.Status.Conditions, conditions.TypeReady)
		Expect(readyCondition).NotTo(BeNil())
		Expect(readyCondition.Status).To(Equal(metav1.ConditionTrue))
		Expect(provider.principal.ensureCalls).To(Equal(1))
	})

	It("removes the finalizer without deleting the external principal for Reference policy", func() {
		principal := createCloudPrincipal(ctx, "reference-deletion", func(p *vedro.CloudPrincipal) {
			p.Spec.ManagementPolicy = vedro.PrincipalManagementPolicyReference
			p.Spec.Managed = nil
			p.Spec.Reference = &vedro.ReferencedPrincipalSpec{Name: "referenced-principal"}
		})
		addPrincipalFinalizer(ctx, principal)
		Expect(k8sClient.Delete(ctx, principal)).To(Succeed())

		result, err := reconcileCloudPrincipal(ctx, reconciler, principal)

		Expect(err).NotTo(HaveOccurred())
		Expect(result).To(Equal(reconcile.Result{}))
		Expect(provider.principal.deleteCalls).To(Equal(0))
		expectCloudPrincipalNotFound(ctx, client.ObjectKeyFromObject(principal))
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
