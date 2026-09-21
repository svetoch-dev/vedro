package capabilities

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	vedro "github.com/svetoch-dev/vedro/api/v1alpha1"
	"github.com/svetoch-dev/vedro/internal/cloud"
)

var _ = Describe("ValidatePrincipalAuthCapabilities", func() {
	It("valid static credentials", func() {
		principalAuth := vedro.CloudPrincipalAuthSpec{
			Method: vedro.AuthMethodStaticCredentials,
		}
		caps := cloud.PrincipalAuthCapabilities{
			StaticCredentials: true,
			WorkloadIdentity:  true,
			WorkloadIdentityKinds: map[vedro.PrincipalKind]bool{
				vedro.PrincipalKindRole: true,
			},
			StaticCredentialsKinds: map[vedro.PrincipalKind]bool{
				vedro.PrincipalKindServiceAccount: true,
			},
		}
		unsupported := ValidatePrincipalAuthCapabilities(caps, principalAuth, vedro.PrincipalKindServiceAccount)
		Expect(unsupported).To(BeEmpty())
	})
	It("valid workload identity", func() {
		principalAuth := vedro.CloudPrincipalAuthSpec{
			Method: vedro.AuthMethodWorkloadIdentity,
		}
		caps := cloud.PrincipalAuthCapabilities{
			StaticCredentials: true,
			WorkloadIdentity:  true,
			WorkloadIdentityKinds: map[vedro.PrincipalKind]bool{
				vedro.PrincipalKindRole: true,
			},
			StaticCredentialsKinds: map[vedro.PrincipalKind]bool{
				vedro.PrincipalKindServiceAccount: true,
			},
		}
		unsupported := ValidatePrincipalAuthCapabilities(caps, principalAuth, vedro.PrincipalKindRole)
		Expect(unsupported).To(BeEmpty())
	})
	It("invalid workload identity", func() {
		principalAuth := vedro.CloudPrincipalAuthSpec{
			Method: vedro.AuthMethodWorkloadIdentity,
		}
		caps := cloud.PrincipalAuthCapabilities{
			StaticCredentials: true,
			WorkloadIdentity:  false,
			WorkloadIdentityKinds: map[vedro.PrincipalKind]bool{
				vedro.PrincipalKindServiceAccount: true,
			},
			StaticCredentialsKinds: map[vedro.PrincipalKind]bool{
				vedro.PrincipalKindServiceAccount: true,
			},
		}
		want := []vedro.UnsupportedFeature{
			{
				Field:   "method",
				Message: "Method WorkloadIdentity is unsupported",
				Reason:  vedro.PrincipalAuthUnsupportedWorkloadIdentity,
			},
			{
				Field:   "principalRef.name",
				Message: "CloudPrincipal Kind can not authenticate using WorkloadIdentity",
				Reason:  vedro.PrincipalAuthUnsupportedWorkloadIdentityKind,
			},
		}
		unsupported := ValidatePrincipalAuthCapabilities(caps, principalAuth, vedro.PrincipalKindRole)
		Expect(unsupported).NotTo(BeEmpty())
		Expect(unsupported).To(Equal(want))

		caps.WorkloadIdentityKinds[vedro.PrincipalKindRole] = true
		want = []vedro.UnsupportedFeature{
			{
				Field:   "method",
				Message: "Method WorkloadIdentity is unsupported",
				Reason:  vedro.PrincipalAuthUnsupportedWorkloadIdentity,
			},
		}
		unsupported = ValidatePrincipalAuthCapabilities(caps, principalAuth, vedro.PrincipalKindRole)
		Expect(unsupported).NotTo(BeEmpty())
		Expect(unsupported).To(Equal(want))
	})
	It("invalid static credentials", func() {
		principalAuth := vedro.CloudPrincipalAuthSpec{
			Method: vedro.AuthMethodStaticCredentials,
		}
		caps := cloud.PrincipalAuthCapabilities{
			StaticCredentials: false,
			WorkloadIdentity:  false,
			StaticCredentialsKinds: map[vedro.PrincipalKind]bool{
				vedro.PrincipalKindRole: true,
			},
		}
		want := []vedro.UnsupportedFeature{
			{
				Field:   "method",
				Message: "Method StaticCredentials is unsupported",
				Reason:  vedro.PrincipalAuthUnsupportedStaticCredentials,
			},
			{
				Field:   "principalRef.name",
				Message: "StaticCredentials can not be issued for CloudPrincipals Kind",
				Reason:  vedro.PrincipalAuthUnsupportedStaticCredentialsKind,
			},
		}
		unsupported := ValidatePrincipalAuthCapabilities(caps, principalAuth, vedro.PrincipalKindServiceAccount)
		Expect(unsupported).NotTo(BeEmpty())
		Expect(unsupported).To(Equal(want))

		caps.StaticCredentialsKinds[vedro.PrincipalKindServiceAccount] = true
		want = []vedro.UnsupportedFeature{
			{
				Field:   "method",
				Message: "Method StaticCredentials is unsupported",
				Reason:  vedro.PrincipalAuthUnsupportedStaticCredentials,
			},
		}
		unsupported = ValidatePrincipalAuthCapabilities(caps, principalAuth, vedro.PrincipalKindServiceAccount)
		Expect(unsupported).NotTo(BeEmpty())
		Expect(unsupported).To(Equal(want))
	})
})
