package capabilities

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	vedro "github.com/svetoch-dev/vedro/api/v1alpha1"
	"github.com/svetoch-dev/vedro/internal/cloud"
)

var _ = Describe("ValidatePrincipalCapabilities", func() {
	It("supported kind for managed", func() {
		principal := vedro.CloudPrincipalSpec{
			ProviderRef: vedro.ProviderConfigReference{
				Name: "some-provider",
			},
			Kind:             vedro.PrincipalKindServiceAccount,
			ManagementPolicy: vedro.PrincipalManagementPolicyManaged,
		}
		caps := cloud.PrincipalCapabilities{
			ManagedKinds: map[vedro.PrincipalKind]bool{
				vedro.PrincipalKindServiceAccount: true,
			},
		}
		unsupported := ValidatePrincipalCapabilities(caps, principal)
		Expect(unsupported).To(BeEmpty())
	})
	It("unsupported kind for managed", func() {
		principal := vedro.CloudPrincipalSpec{
			ProviderRef: vedro.ProviderConfigReference{
				Name: "some-provider",
			},
			Kind:             vedro.PrincipalKindServiceAccount,
			ManagementPolicy: vedro.PrincipalManagementPolicyManaged,
		}
		caps := cloud.PrincipalCapabilities{
			ManagedKinds: map[vedro.PrincipalKind]bool{
				vedro.PrincipalKindServiceAccount: false,
			},
		}
		want := []vedro.UnsupportedFeature{
			unsupportedFeatures["ManagedServiceAccount"],
		}
		unsupported := ValidatePrincipalCapabilities(caps, principal)
		Expect(unsupported).NotTo(BeEmpty())
		Expect(unsupported).To(Equal(want))
	})
	It("missing kind for managed", func() {
		principal := vedro.CloudPrincipalSpec{
			ProviderRef: vedro.ProviderConfigReference{
				Name: "some-provider",
			},
			Kind:             vedro.PrincipalKindRole,
			ManagementPolicy: vedro.PrincipalManagementPolicyManaged,
		}
		caps := cloud.PrincipalCapabilities{
			ManagedKinds: map[vedro.PrincipalKind]bool{
				vedro.PrincipalKindServiceAccount: true,
			},
		}
		want := []vedro.UnsupportedFeature{
			unsupportedFeatures["ManagedRole"],
		}
		unsupported := ValidatePrincipalCapabilities(caps, principal)
		Expect(unsupported).NotTo(BeEmpty())
		Expect(unsupported).To(Equal(want))
	})
	It("supported kind for reference", func() {
		principal := vedro.CloudPrincipalSpec{
			ProviderRef: vedro.ProviderConfigReference{
				Name: "some-provider",
			},
			Kind:             vedro.PrincipalKindUser,
			ManagementPolicy: vedro.PrincipalManagementPolicyReference,
		}
		caps := cloud.PrincipalCapabilities{
			ReferencedKinds: map[vedro.PrincipalKind]bool{
				vedro.PrincipalKindServiceAccount: true,
				vedro.PrincipalKindUser:           true,
			},
		}
		unsupported := ValidatePrincipalCapabilities(caps, principal)
		Expect(unsupported).To(BeEmpty())
	})
	It("unsupported kind for reference", func() {
		principal := vedro.CloudPrincipalSpec{
			ProviderRef: vedro.ProviderConfigReference{
				Name: "some-provider",
			},
			Kind:             vedro.PrincipalKindUser,
			ManagementPolicy: vedro.PrincipalManagementPolicyReference,
		}
		caps := cloud.PrincipalCapabilities{
			ReferencedKinds: map[vedro.PrincipalKind]bool{
				vedro.PrincipalKindServiceAccount: true,
				vedro.PrincipalKindUser:           false,
			},
		}

		want := []vedro.UnsupportedFeature{
			unsupportedFeatures["ReferencedUser"],
		}
		unsupported := ValidatePrincipalCapabilities(caps, principal)
		Expect(unsupported).NotTo(BeEmpty())
		Expect(unsupported).To(Equal(want))
	})
	It("missing kind for reference", func() {
		principal := vedro.CloudPrincipalSpec{
			ProviderRef: vedro.ProviderConfigReference{
				Name: "some-provider",
			},
			Kind:             vedro.PrincipalKindGroup,
			ManagementPolicy: vedro.PrincipalManagementPolicyReference,
		}
		caps := cloud.PrincipalCapabilities{
			ReferencedKinds: map[vedro.PrincipalKind]bool{
				vedro.PrincipalKindServiceAccount: true,
				vedro.PrincipalKindUser:           true,
			},
		}

		want := []vedro.UnsupportedFeature{
			unsupportedFeatures["ReferencedGroup"],
		}
		unsupported := ValidatePrincipalCapabilities(caps, principal)
		Expect(unsupported).NotTo(BeEmpty())
		Expect(unsupported).To(Equal(want))
	})

})
