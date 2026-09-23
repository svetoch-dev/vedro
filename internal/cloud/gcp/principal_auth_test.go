package gcp

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	vedro "github.com/svetoch-dev/vedro/api/v1alpha1"
	"github.com/svetoch-dev/vedro/internal/cloud"
	cloudtest "github.com/svetoch-dev/vedro/internal/cloud/test"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = cloudtest.PrincipalAuthProviderTests(cloudtest.Config{
	NewPrincipalAuth: func(api cloud.PrincipalAPI) cloud.PrincipalAuthProvider {
		return &PrincipalAuth{api: api}
	},
	SupportsWorkloadIdentity: true,
})

var _ = Describe("GCP PrincipalAuth propagation grace period", func() {
	var (
		ctx       context.Context
		fake      *cloudtest.FakePrincipalAPI
		provider  *PrincipalAuth
		principal vedro.CloudPrincipal
		auth      vedro.CloudPrincipalAuth
	)

	BeforeEach(func() {
		ctx = context.Background()
		fake = &cloudtest.FakePrincipalAPI{GetAuthErr: cloud.ErrAuthNotFound}
		provider = &PrincipalAuth{api: fake}
		principal = cloudtest.NewPrincipalCR("my-principal", func(p *vedro.CloudPrincipal) {
			p.Status.ExternalId = "principal-id"
		})
		auth = vedro.CloudPrincipalAuth{
			Spec: vedro.CloudPrincipalAuthSpec{Method: vedro.AuthMethodStaticCredentials},
			Status: vedro.CloudPrincipalAuthStatus{
				Applied: &vedro.CloudPrincipalAuthProperties{
					Method:        vedro.AuthMethodStaticCredentials,
					CredentialsId: "credentials-id",
					CreatedAt:     metav1.NewTime(time.Now().Add(-30 * time.Second)),
				},
			},
		}
	})

	It("keeps a recently created static key when GCP temporarily reports it missing", func() {
		result, err := provider.EnsureAuthentication(ctx, auth, principal)

		Expect(err).NotTo(HaveOccurred())
		Expect(result).To(Equal(&cloud.PrincipalAuthResult{
			Method:        vedro.AuthMethodStaticCredentials,
			CredentialsID: "credentials-id",
		}))
		Expect(fake.GetAuthInputs).To(HaveLen(1))
		Expect(fake.CreateAuthInputs).To(BeEmpty())
	})

	It("recreates a static key once the grace period has elapsed", func() {
		auth.Status.Applied.CreatedAt = metav1.NewTime(time.Now().Add(-2 * gcpKeyPropagationGracePeriod))
		expected := &cloud.PrincipalAuthResult{CredentialsID: "replacement-id"}
		fake.CreateAuthResult = expected

		result, err := provider.EnsureAuthentication(ctx, auth, principal)

		Expect(err).NotTo(HaveOccurred())
		Expect(result).To(BeIdenticalTo(expected))
		Expect(fake.CreateAuthInputs).To(Equal(fake.GetAuthInputs))
	})

	It("recreates missing workload identity without the static key grace period", func() {
		auth.Namespace = "my-namespace"
		auth.Spec.Method = vedro.AuthMethodWorkloadIdentity
		auth.Spec.WorkloadIdentity = &vedro.WorkloadIdentitySpec{
			ServiceAccountRef: vedro.AuthObjectReference{Name: "my-service-account"},
		}
		auth.Status.Applied.Method = vedro.AuthMethodWorkloadIdentity
		expected := &cloud.PrincipalAuthResult{CredentialsID: "replacement-id"}
		fake.CreateAuthResult = expected

		result, err := provider.EnsureAuthentication(ctx, auth, principal)

		Expect(err).NotTo(HaveOccurred())
		Expect(result).To(BeIdenticalTo(expected))
		Expect(fake.CreateAuthInputs).To(Equal(fake.GetAuthInputs))
		Expect(fake.CreateAuthInputs[0].K8sServiceAccount).To(Equal(&vedro.NamespacedName{
			Name: "my-service-account", Namespace: "my-namespace",
		}))
	})
})
