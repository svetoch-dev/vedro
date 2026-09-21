package cloudtest

import (
	"context"
	"errors"

	. "github.com/onsi/ginkgo/v2" //nolint:staticcheck
	. "github.com/onsi/gomega"    //nolint:staticcheck
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	vedro "github.com/svetoch-dev/vedro/api/v1alpha1"
	"github.com/svetoch-dev/vedro/internal/cloud"
)

// PrincipalAuthProviderTests registers the provider-agnostic
// EnsureAuthentication and DeleteAuthentication specs. Call it from each
// provider's Ginkgo suite, e.g.:
//
//	var _ = cloudtest.PrincipalAuthProviderTests(cloudtest.Config{...})
func PrincipalAuthProviderTests(cfg Config) bool {
	newPrincipal := func() vedro.CloudPrincipal {
		return NewPrincipalCR("my-principal", func(principal *vedro.CloudPrincipal) {
			principal.Status.ExternalId = "principal-id"
		})
	}
	newPrincipalAuth := func(mods ...func(*vedro.CloudPrincipalAuth)) vedro.CloudPrincipalAuth {
		auth := vedro.CloudPrincipalAuth{
			ObjectMeta: metav1.ObjectMeta{Name: "my-auth", Namespace: "my-namespace"},
			Spec: vedro.CloudPrincipalAuthSpec{
				Method: vedro.AuthMethodStaticCredentials,
				PrincipalRef: vedro.PrincipalReference{
					Name: "my-principal", Namespace: "my-namespace",
				},
				StaticCredentials: &vedro.StaticCredentialsSpec{
					SecretRef: vedro.AuthObjectReference{Name: "my-auth-secret"},
				},
			},
		}
		for _, mod := range mods {
			mod(&auth)
		}
		return auth
	}
	newAppliedPrincipalAuth := func() vedro.CloudPrincipalAuth {
		return newPrincipalAuth(func(auth *vedro.CloudPrincipalAuth) {
			auth.Status.Applied = &vedro.CloudPrincipalAuthProperties{
				Method:        vedro.AuthMethodStaticCredentials,
				CredentialsId: "credentials-id",
				PrincipalId:   "principal-id",
			}
		})
	}

	Describe("PrincipalAuthProvider.EnsureAuthentication", func() {
		var (
			ctx           context.Context
			fake          *FakePrincipalAPI
			principalAuth cloud.PrincipalAuthProvider
		)

		BeforeEach(func() {
			ctx = context.Background()
			fake = &FakePrincipalAPI{}
			principalAuth = cfg.NewPrincipalAuth(fake)
		})

		It("creates authentication material when no credentials were applied", func() {
			expected := &cloud.PrincipalAuthResult{CredentialsID: "new-credentials-id"}
			fake.CreateAuthResult = expected

			result, err := principalAuth.EnsureAuthentication(ctx, newPrincipalAuth(), newPrincipal())

			Expect(err).NotTo(HaveOccurred())
			Expect(result).To(BeIdenticalTo(expected))
			Expect(fake.GetAuthInputs).To(BeEmpty())
			Expect(fake.CreateAuthInputs).To(Equal([]cloud.PrincipalAuthSetup{{
				Method:           vedro.AuthMethodStaticCredentials,
				ServiceAccountID: "principal-id",
			}}))
		})

		It("returns an error when authentication material creation fails", func() {
			fake.CreateAuthErr = errors.New("create failed")

			result, err := principalAuth.EnsureAuthentication(ctx, newPrincipalAuth(), newPrincipal())

			Expect(err).To(MatchError(ContainSubstring("create failed")))
			Expect(result).To(BeNil())
			Expect(fake.CreateAuthInputs).To(HaveLen(1))
		})

		It("returns existing authentication material", func() {
			expected := &cloud.PrincipalAuthResult{CredentialsID: "credentials-id"}
			fake.GetAuthResult = expected

			result, err := principalAuth.EnsureAuthentication(ctx, newAppliedPrincipalAuth(), newPrincipal())

			Expect(err).NotTo(HaveOccurred())
			Expect(result).To(BeIdenticalTo(expected))
			Expect(fake.GetAuthInputs).To(Equal([]cloud.PrincipalAuthSetup{{
				Method:           vedro.AuthMethodStaticCredentials,
				ServiceAccountID: "principal-id",
				CredentialsID:    "credentials-id",
			}}))
			Expect(fake.CreateAuthInputs).To(BeEmpty())
		})

		It("recreates authentication material that no longer exists", func() {
			expected := &cloud.PrincipalAuthResult{CredentialsID: "replacement-id"}
			fake.GetAuthErr = cloud.ErrAuthNotFound
			fake.CreateAuthResult = expected

			result, err := principalAuth.EnsureAuthentication(ctx, newAppliedPrincipalAuth(), newPrincipal())

			Expect(err).NotTo(HaveOccurred())
			Expect(result).To(BeIdenticalTo(expected))
			Expect(fake.GetAuthInputs).To(HaveLen(1))
			Expect(fake.CreateAuthInputs).To(Equal(fake.GetAuthInputs))
		})

		It("returns an error when recreating missing authentication material fails", func() {
			fake.GetAuthErr = cloud.ErrAuthNotFound
			fake.CreateAuthErr = errors.New("replacement failed")

			result, err := principalAuth.EnsureAuthentication(ctx, newAppliedPrincipalAuth(), newPrincipal())

			Expect(err).To(MatchError(ContainSubstring("replacement failed")))
			Expect(result).To(BeNil())
			Expect(fake.GetAuthInputs).To(HaveLen(1))
			Expect(fake.CreateAuthInputs).To(HaveLen(1))
		})

		It("returns an error when existing authentication material cannot be fetched", func() {
			fake.GetAuthErr = errors.New("get failed")

			result, err := principalAuth.EnsureAuthentication(ctx, newAppliedPrincipalAuth(), newPrincipal())

			Expect(err).To(MatchError(ContainSubstring("get failed")))
			Expect(result).To(BeNil())
			Expect(fake.CreateAuthInputs).To(BeEmpty())
		})

		if cfg.SupportsWorkloadIdentity {
			It("passes the Kubernetes ServiceAccount for workload identity", func() {
				auth := newPrincipalAuth(func(auth *vedro.CloudPrincipalAuth) {
					auth.Spec.Method = vedro.AuthMethodWorkloadIdentity
					auth.Spec.StaticCredentials = nil
					auth.Spec.WorkloadIdentity = &vedro.WorkloadIdentitySpec{
						ServiceAccountRef: vedro.AuthObjectReference{Name: "my-service-account"},
					}
				})

				_, err := principalAuth.EnsureAuthentication(ctx, auth, newPrincipal())

				Expect(err).NotTo(HaveOccurred())
				Expect(fake.CreateAuthInputs).To(Equal([]cloud.PrincipalAuthSetup{{
					Method:           vedro.AuthMethodWorkloadIdentity,
					ServiceAccountID: "principal-id",
					K8sServiceAccount: &vedro.NamespacedName{
						Name: "my-service-account", Namespace: "my-namespace",
					},
				}}))
			})
		}
	})

	Describe("PrincipalAuthProvider.DeleteAuthentication", func() {
		var (
			ctx           context.Context
			fake          *FakePrincipalAPI
			principalAuth cloud.PrincipalAuthProvider
		)

		BeforeEach(func() {
			ctx = context.Background()
			fake = &FakePrincipalAPI{}
			principalAuth = cfg.NewPrincipalAuth(fake)
		})

		It("rejects deletion when no applied status exists", func() {
			err := principalAuth.DeleteAuthentication(ctx, newPrincipalAuth())

			Expect(err).To(MatchError(ContainSubstring("credentialsId")))
			Expect(fake.DeleteAuthInputs).To(BeEmpty())
		})

		It("rejects deletion when the applied credentials ID is empty", func() {
			auth := newAppliedPrincipalAuth()
			auth.Status.Applied.CredentialsId = ""

			err := principalAuth.DeleteAuthentication(ctx, auth)

			Expect(err).To(MatchError(ContainSubstring("credentialsId")))
			Expect(fake.DeleteAuthInputs).To(BeEmpty())
		})

		It("deletes the applied authentication material", func() {
			err := principalAuth.DeleteAuthentication(ctx, newAppliedPrincipalAuth())

			Expect(err).NotTo(HaveOccurred())
			Expect(fake.DeleteAuthInputs).To(Equal([]cloud.PrincipalAuthSetup{{
				Method:           vedro.AuthMethodStaticCredentials,
				ServiceAccountID: "principal-id",
				CredentialsID:    "credentials-id",
			}}))
		})

		It("returns errors from authentication material deletion", func() {
			fake.DeleteAuthErr = errors.New("delete failed")

			err := principalAuth.DeleteAuthentication(ctx, newAppliedPrincipalAuth())

			Expect(err).To(MatchError("delete failed"))
			Expect(fake.DeleteAuthInputs).To(HaveLen(1))
		})
	})

	return true
}
