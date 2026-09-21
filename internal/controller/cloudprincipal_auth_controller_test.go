package controller

import (
	"context"
	"errors"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
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

var _ = Describe("CloudPrincipalAuthReconciler", func() {
	var (
		reconciler    *CloudPrincipalAuthReconciler
		provider      *fakeProvider
		principalAuth *fakePrincipalAuthProvider
	)

	BeforeEach(func() {
		principalAuth = &fakePrincipalAuthProvider{
			ensureResult: &cloud.PrincipalAuthResult{
				CredentialsID: "credentials-id",
				SecretData: map[string][]byte{
					"credentials.json": []byte("secret-data"),
				},
			},
		}
		provider = &fakeProvider{
			capabilities: cloud.Capabilities{
				PrincipalAuth: cloud.PrincipalAuthCapabilities{
					StaticCredentials: true,
					WorkloadIdentity:  true,
					StaticCredentialsKinds: map[vedro.PrincipalKind]bool{
						vedro.PrincipalKindServiceAccount: true,
					},
					WorkloadIdentityKinds: map[vedro.PrincipalKind]bool{
						vedro.PrincipalKindServiceAccount: true,
					},
				},
			},
			principalAuth: principalAuth,
		}
		reconciler = &CloudPrincipalAuthReconciler{
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

	It("ignores missing CloudPrincipalAuth resources", func() {
		result, err := reconciler.Reconcile(ctx, reconcile.Request{
			NamespacedName: types.NamespacedName{
				Name:      "missing-auth",
				Namespace: "default",
			},
		})

		Expect(err).NotTo(HaveOccurred())
		Expect(result).To(Equal(reconcile.Result{}))
		Expect(principalAuth.ensureCalls).To(BeZero())
	})

	It("adds the finalizer and reports a missing CloudPrincipal dependency", func() {
		auth := createCloudPrincipalAuth(ctx, "missing-principal", "missing")

		result, err := reconcileCloudPrincipalAuth(ctx, reconciler, auth)

		Expect(err).NotTo(HaveOccurred())
		Expect(result).To(Equal(reconcile.Result{}))
		fetched := getCloudPrincipalAuth(ctx, client.ObjectKeyFromObject(auth))
		Expect(fetched.Finalizers).To(ContainElement(principalAuthFinalizer))
		Expect(fetched.Status.ObservedGeneration).To(Equal(fetched.Generation))
		expectPrincipalAuthCondition(fetched, conditions.TypeReady, metav1.ConditionFalse, conditions.ReasonCloudPrincipalNotFound)
		expectPrincipalAuthCondition(fetched, conditions.TypeCloudPrincipalReady, metav1.ConditionFalse, conditions.ReasonCloudPrincipalNotFound)
		Expect(principalAuth.ensureCalls).To(BeZero())
	})

	It("waits until the CloudPrincipal dependency is ready", func() {
		createCloudPrincipal(ctx, "principal")
		auth := createCloudPrincipalAuth(ctx, "principal-not-ready", "principal")

		result, err := reconcileCloudPrincipalAuth(ctx, reconciler, auth)

		Expect(err).NotTo(HaveOccurred())
		Expect(result).To(Equal(reconcile.Result{}))
		fetched := getCloudPrincipalAuth(ctx, client.ObjectKeyFromObject(auth))
		expectPrincipalAuthCondition(fetched, conditions.TypeReady, metav1.ConditionFalse, conditions.ReasonCloudPrincipalAuthDependencyNotReady)
		expectPrincipalAuthCondition(fetched, conditions.TypeCloudPrincipalReady, metav1.ConditionFalse, conditions.ReasonNoConditions)
		Expect(principalAuth.ensureCalls).To(BeZero())
	})

	It("rejects authentication for a referenced CloudPrincipal", func() {
		principal := createCloudPrincipal(ctx, "referenced-principal", func(p *vedro.CloudPrincipal) {
			p.Spec.ManagementPolicy = vedro.PrincipalManagementPolicyReference
			p.Spec.Managed = nil
			p.Spec.Reference = &vedro.ReferencedPrincipalSpec{Name: "external-principal"}
		})
		markCloudPrincipalReady(ctx, principal)
		auth := createCloudPrincipalAuth(ctx, "referenced-principal-auth", principal.Name)

		_, err := reconcileCloudPrincipalAuth(ctx, reconciler, auth)

		Expect(err).NotTo(HaveOccurred())
		fetched := getCloudPrincipalAuth(ctx, client.ObjectKeyFromObject(auth))
		expectPrincipalAuthCondition(fetched, conditions.TypeReady, metav1.ConditionFalse, conditions.ReasonCloudPrincipalIsNotManaged)
		Expect(principalAuth.ensureCalls).To(BeZero())
	})

	It("records provider factory errors", func() {
		principal := createReadyCloudPrincipal(ctx, "factory-error-principal")
		createProviderConfig(ctx)
		auth := createCloudPrincipalAuth(ctx, "factory-error", principal.Name)
		reconciler.ProviderFactory = func(context.Context, vedro.ProviderConfig, client.Client) (cloud.Provider, error) {
			return nil, errors.New("provider setup failed")
		}

		_, err := reconcileCloudPrincipalAuth(ctx, reconciler, auth)

		Expect(err).To(MatchError("provider setup failed"))
		fetched := getCloudPrincipalAuth(ctx, client.ObjectKeyFromObject(auth))
		expectPrincipalAuthCondition(fetched, conditions.TypeReady, metav1.ConditionFalse, conditions.ReasonProviderConfigError)
		Expect(principalAuth.ensureCalls).To(BeZero())
	})

	It("records unsupported authentication methods without calling the provider", func() {
		principal := createReadyCloudPrincipal(ctx, "unsupported-principal")
		createProviderConfig(ctx)
		auth := createCloudPrincipalAuth(ctx, "unsupported-auth", principal.Name)
		provider.capabilities.PrincipalAuth.StaticCredentials = false

		result, err := reconcileCloudPrincipalAuth(ctx, reconciler, auth)

		Expect(err).NotTo(HaveOccurred())
		Expect(result).To(Equal(reconcile.Result{}))
		fetched := getCloudPrincipalAuth(ctx, client.ObjectKeyFromObject(auth))
		Expect(fetched.Status.UnsupportedFeatures).To(HaveLen(1))
		Expect(fetched.Status.UnsupportedFeatures[0].Reason).To(Equal(vedro.PrincipalAuthUnsupportedStaticCredentials))
		expectPrincipalAuthCondition(fetched, conditions.TypeReady, metav1.ConditionFalse, conditions.ReasonCloudPrincipalAuthUnsupportedFeatures)
		Expect(principalAuth.ensureCalls).To(BeZero())
		Expect(provider.cleanupCalled).To(BeTrue())
	})

	It("denies authentication in a restricted namespace and recovers", func() {
		principal := createReadyCloudPrincipal(ctx, "restricted-principal")
		createProviderConfig(ctx)
		auth := createCloudPrincipalAuth(ctx, "restricted-auth", principal.Name)
		updateUsagePolicy(ctx, func(p *vedro.UsagePolicySpec) {
			p.AllowedNamespaces = vedro.AllowedNamespacesSpec{Names: []string{"other"}}
		})

		_, err := reconcileCloudPrincipalAuth(ctx, reconciler, auth)

		Expect(err).NotTo(HaveOccurred())
		Expect(principalAuth.ensureCalls).To(BeZero())
		fetched := getCloudPrincipalAuth(ctx, client.ObjectKeyFromObject(auth))
		expectPrincipalAuthCondition(fetched, conditions.TypeReady, metav1.ConditionFalse, conditions.ReasonCloudPrincipalAuthSpecRestricted)
		Expect(meta.FindStatusCondition(fetched.Status.Conditions, conditions.TypeReady).Message).To(ContainSubstring("default"))

		updateUsagePolicy(ctx, func(p *vedro.UsagePolicySpec) {
			p.AllowedNamespaces = vedro.AllowedNamespacesSpec{All: true}
		})
		_, err = reconcileCloudPrincipalAuth(ctx, reconciler, auth)
		Expect(err).NotTo(HaveOccurred())
		Expect(principalAuth.ensureCalls).To(Equal(1))
		expectPrincipalAuthCondition(getCloudPrincipalAuth(ctx, client.ObjectKeyFromObject(auth)), conditions.TypeReady, metav1.ConditionTrue, conditions.ReasonCloudPrincipalAuthReconciled)
	})

	It("records errors returned while ensuring authentication", func() {
		principal := createReadyCloudPrincipal(ctx, "ensure-error-principal")
		createProviderConfig(ctx)
		auth := createCloudPrincipalAuth(ctx, "ensure-error", principal.Name)
		principalAuth.ensureErr = errors.New("ensure failed")

		result, err := reconcileCloudPrincipalAuth(ctx, reconciler, auth)

		Expect(err).To(MatchError("ensure failed"))
		Expect(result).To(Equal(reconcile.Result{}))
		fetched := getCloudPrincipalAuth(ctx, client.ObjectKeyFromObject(auth))
		expectPrincipalAuthCondition(fetched, conditions.TypeReady, metav1.ConditionFalse, conditions.ReasonCloudPrincipalAuthEnsureError)
		Expect(principalAuth.ensureCalls).To(Equal(1))
		Expect(provider.cleanupCalled).To(BeTrue())
	})

	It("creates a Secret and records successful static credentials", func() {
		principal := createReadyCloudPrincipal(ctx, "static-principal")
		createProviderConfig(ctx)
		auth := createCloudPrincipalAuth(ctx, "static-auth", principal.Name)

		result, err := reconcileCloudPrincipalAuth(ctx, reconciler, auth)

		Expect(err).NotTo(HaveOccurred())
		Expect(result).To(Equal(reconcile.Result{}))
		Expect(principalAuth.ensureCalls).To(Equal(1))
		Expect(principalAuth.lastPrincipal.Name).To(Equal(principal.Name))

		secret := &corev1.Secret{}
		Expect(k8sClient.Get(ctx, client.ObjectKey{Name: "static-auth-secret", Namespace: "default"}, secret)).To(Succeed())
		Expect(secret.Data).To(Equal(map[string][]byte{"credentials.json": []byte("secret-data")}))
		Expect(metav1.IsControlledBy(secret, auth)).To(BeTrue())

		fetched := getCloudPrincipalAuth(ctx, client.ObjectKeyFromObject(auth))
		Expect(fetched.Status.ObservedProvider).To(Equal("test-provider"))
		Expect(fetched.Status.Applied).To(Equal(&vedro.CloudPrincipalAuthProperties{
			Method:        vedro.AuthMethodStaticCredentials,
			CredentialsId: "credentials-id",
			PrincipalId:   "principal-id",
			SecretRef:     &vedro.NamespacedName{Name: "static-auth-secret", Namespace: "default"},
		}))
		expectPrincipalAuthCondition(fetched, conditions.TypeStaticCredentialsConfigured, metav1.ConditionTrue, conditions.ReasonStaticCredentialsReconciled)
		expectPrincipalAuthCondition(fetched, conditions.TypeReady, metav1.ConditionTrue, conditions.ReasonCloudPrincipalAuthReconciled)
	})

	It("patches a ServiceAccount and records successful workload identity", func() {
		principal := createReadyCloudPrincipal(ctx, "workload-principal")
		createProviderConfig(ctx)
		serviceAccount := &corev1.ServiceAccount{ObjectMeta: metav1.ObjectMeta{Name: "workload-sa", Namespace: "default"}}
		Expect(k8sClient.Create(ctx, serviceAccount)).To(Succeed())
		DeferCleanup(func() {
			Expect(client.IgnoreNotFound(k8sClient.Delete(ctx, serviceAccount))).To(Succeed())
		})
		auth := createWorkloadIdentityAuth(ctx, "workload-auth", principal.Name, serviceAccount.Name)
		principalAuth.ensureResult = &cloud.PrincipalAuthResult{
			CredentialsID: "workload-credentials-id",
			ServiceAccountPatch: &corev1.ServiceAccount{
				TypeMeta: metav1.TypeMeta{APIVersion: "v1", Kind: "ServiceAccount"},
				ObjectMeta: metav1.ObjectMeta{
					Name:        serviceAccount.Name,
					Namespace:   serviceAccount.Namespace,
					Annotations: map[string]string{"iam.example.test/service-account": "external-principal"},
				},
			},
		}

		result, err := reconcileCloudPrincipalAuth(ctx, reconciler, auth)

		Expect(err).NotTo(HaveOccurred())
		Expect(result).To(Equal(reconcile.Result{}))
		patched := &corev1.ServiceAccount{}
		Expect(k8sClient.Get(ctx, client.ObjectKeyFromObject(serviceAccount), patched)).To(Succeed())
		Expect(patched.Annotations).To(HaveKeyWithValue("iam.example.test/service-account", "external-principal"))

		fetched := getCloudPrincipalAuth(ctx, client.ObjectKeyFromObject(auth))
		Expect(fetched.Status.Applied).To(Equal(&vedro.CloudPrincipalAuthProperties{
			Method:            vedro.AuthMethodWorkloadIdentity,
			CredentialsId:     "workload-credentials-id",
			PrincipalId:       "principal-id",
			ServiceAccountRef: &vedro.NamespacedName{Name: serviceAccount.Name, Namespace: serviceAccount.Namespace},
		}))
		expectPrincipalAuthCondition(fetched, conditions.TypeWorkloadIdentityConfigured, metav1.ConditionTrue, conditions.ReasonWorkloadIdentityReconciled)
		expectPrincipalAuthCondition(fetched, conditions.TypeReady, metav1.ConditionTrue, conditions.ReasonCloudPrincipalAuthReconciled)
	})

	It("retains static credentials and removes their owner reference on deletion", func() {
		principal := createReadyCloudPrincipal(ctx, "retain-principal")
		createProviderConfig(ctx)
		auth := createCloudPrincipalAuth(ctx, "retain-auth", principal.Name)
		_, err := reconcileCloudPrincipalAuth(ctx, reconciler, auth)
		Expect(err).NotTo(HaveOccurred())

		Expect(k8sClient.Delete(ctx, auth)).To(Succeed())
		_, err = reconcileCloudPrincipalAuth(ctx, reconciler, auth)

		Expect(err).NotTo(HaveOccurred())
		Expect(principalAuth.deleteCalls).To(BeZero())
		expectCloudPrincipalAuthNotFound(ctx, client.ObjectKeyFromObject(auth))
		secret := &corev1.Secret{}
		Expect(k8sClient.Get(ctx, client.ObjectKey{Name: "retain-auth-secret", Namespace: "default"}, secret)).To(Succeed())
		Expect(secret.OwnerReferences).To(BeEmpty())
		DeferCleanup(func() {
			Expect(client.IgnoreNotFound(k8sClient.Delete(ctx, secret))).To(Succeed())
		})
	})

	It("deletes cloud authentication material for Delete policy", func() {
		principal := createReadyCloudPrincipal(ctx, "delete-principal")
		createProviderConfig(ctx)
		auth := createCloudPrincipalAuth(ctx, "delete-auth", principal.Name, func(auth *vedro.CloudPrincipalAuth) {
			auth.Spec.StaticCredentials.DeletionPolicy = vedro.DeletionPolicyDelete
		})
		_, err := reconcileCloudPrincipalAuth(ctx, reconciler, auth)
		Expect(err).NotTo(HaveOccurred())

		Expect(k8sClient.Delete(ctx, auth)).To(Succeed())
		result, err := reconcileCloudPrincipalAuth(ctx, reconciler, auth)

		Expect(err).NotTo(HaveOccurred())
		Expect(result).To(Equal(reconcile.Result{}))
		Expect(principalAuth.deleteCalls).To(Equal(1))
		Expect(provider.cleanupCalled).To(BeTrue())
		expectCloudPrincipalAuthNotFound(ctx, client.ObjectKeyFromObject(auth))
	})
})

type fakePrincipalAuthProvider struct {
	ensureResult *cloud.PrincipalAuthResult
	ensureErr    error
	deleteErr    error
	ensureCalls  int
	deleteCalls  int

	lastPrincipalAuth vedro.CloudPrincipalAuth
	lastPrincipal     vedro.CloudPrincipal
}

func (f *fakePrincipalAuthProvider) EnsureAuthentication(
	ctx context.Context,
	principalAuth vedro.CloudPrincipalAuth,
	principal vedro.CloudPrincipal,
) (*cloud.PrincipalAuthResult, error) {
	f.ensureCalls++
	f.lastPrincipalAuth = principalAuth
	f.lastPrincipal = principal
	return f.ensureResult, f.ensureErr
}

func (f *fakePrincipalAuthProvider) DeleteAuthentication(
	ctx context.Context,
	principalAuth vedro.CloudPrincipalAuth,
) error {
	f.deleteCalls++
	f.lastPrincipalAuth = principalAuth
	return f.deleteErr
}

func createReadyCloudPrincipal(ctx context.Context, name string) *vedro.CloudPrincipal {
	principal := createCloudPrincipal(ctx, name)
	markCloudPrincipalReady(ctx, principal)
	return principal
}

func createCloudPrincipalAuth(
	ctx context.Context,
	name string,
	principalName string,
	mutators ...func(*vedro.CloudPrincipalAuth),
) *vedro.CloudPrincipalAuth {
	auth := &vedro.CloudPrincipalAuth{
		TypeMeta:   metav1.TypeMeta{APIVersion: "vedro.svetoch.dev/v1alpha1", Kind: "CloudPrincipalAuth"},
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: "default"},
		Spec: vedro.CloudPrincipalAuthSpec{
			PrincipalRef: vedro.PrincipalReference{Name: principalName, Namespace: "default"},
			Method:       vedro.AuthMethodStaticCredentials,
			StaticCredentials: &vedro.StaticCredentialsSpec{
				SecretRef:      vedro.AuthObjectReference{Name: name + "-secret"},
				DeletionPolicy: vedro.DeletionPolicyRetain,
			},
		},
	}
	for _, mutate := range mutators {
		mutate(auth)
	}

	Expect(k8sClient.Create(ctx, auth)).To(Succeed())
	DeferCleanup(func() {
		cleanupCloudPrincipalAuth(ctx, client.ObjectKeyFromObject(auth))
	})
	return auth
}

func createWorkloadIdentityAuth(
	ctx context.Context,
	name string,
	principalName string,
	serviceAccountName string,
) *vedro.CloudPrincipalAuth {
	return createCloudPrincipalAuth(ctx, name, principalName, func(auth *vedro.CloudPrincipalAuth) {
		auth.Spec.Method = vedro.AuthMethodWorkloadIdentity
		auth.Spec.StaticCredentials = nil
		auth.Spec.WorkloadIdentity = &vedro.WorkloadIdentitySpec{
			ServiceAccountRef: vedro.AuthObjectReference{Name: serviceAccountName},
			DeletionPolicy:    vedro.DeletionPolicyRetain,
		}
	})
}

func reconcileCloudPrincipalAuth(
	ctx context.Context,
	reconciler *CloudPrincipalAuthReconciler,
	auth *vedro.CloudPrincipalAuth,
) (reconcile.Result, error) {
	return reconciler.Reconcile(ctx, reconcile.Request{NamespacedName: client.ObjectKeyFromObject(auth)})
}

func getCloudPrincipalAuth(ctx context.Context, key client.ObjectKey) *vedro.CloudPrincipalAuth {
	auth := &vedro.CloudPrincipalAuth{}
	Expect(k8sClient.Get(ctx, key, auth)).To(Succeed())
	return auth
}

func expectPrincipalAuthCondition(
	auth *vedro.CloudPrincipalAuth,
	conditionType string,
	status metav1.ConditionStatus,
	reason string,
) {
	condition := meta.FindStatusCondition(auth.Status.Conditions, conditionType)
	Expect(condition).NotTo(BeNil())
	Expect(condition.Status).To(Equal(status))
	Expect(condition.Reason).To(Equal(reason))
}

func expectCloudPrincipalAuthNotFound(ctx context.Context, key client.ObjectKey) {
	err := k8sClient.Get(ctx, key, &vedro.CloudPrincipalAuth{})
	Expect(apierrors.IsNotFound(err)).To(BeTrue())
}

func cleanupCloudPrincipalAuth(ctx context.Context, key client.ObjectKey) {
	auth := &vedro.CloudPrincipalAuth{}
	err := k8sClient.Get(ctx, key, auth)
	if apierrors.IsNotFound(err) {
		return
	}
	Expect(err).NotTo(HaveOccurred())
	if len(auth.Finalizers) > 0 {
		auth.Finalizers = nil
		Expect(k8sClient.Update(ctx, auth)).To(Succeed())
	}
	err = k8sClient.Delete(ctx, auth)
	if err != nil {
		Expect(apierrors.IsNotFound(err)).To(BeTrue())
	}
}
