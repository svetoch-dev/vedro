package yc

import (
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	vedro "github.com/svetoch-dev/vedro/api/v1alpha1"
	"github.com/svetoch-dev/vedro/internal/cloud"
	cloudtest "github.com/svetoch-dev/vedro/internal/cloud/test"
)

var _ = cloudtest.PrincipalProviderTests(cloudtest.Config{
	NewPrincipal: func(api cloud.PrincipalAPI) cloud.PrincipalProvider {
		return &Principal{api: api}
	},
})

var _ = cloudtest.PrincipalValidationTests(cloudtest.Config{
	NewPrincipal: func(api cloud.PrincipalAPI) cloud.PrincipalProvider {
		return &Principal{api: api}
	},
})

var _ = Describe("Principal.ValidateYcPrincipalSpec", func() {
	newPrincipal := func(managedName, externalName string) vedro.CloudPrincipal {
		return cloudtest.NewPrincipalCR(
			"service-account",
			func(cp *vedro.CloudPrincipal) {
				cp.Spec.Managed.Name = managedName
				cp.Status = vedro.CloudPrincipalStatus{ExternalName: externalName}
			},
		)
	}

	DescribeTable("validates YC service account names",
		func(specName string, valid bool) {
			result := (&Principal{}).ValidatePrincipalSpec(newPrincipal(specName, ""))

			Expect(result.Valid).To(Equal(valid))
			if !valid {
				Expect(result.Message).To(ContainSubstring("name must be 3-63 characters"))
			}
		},
		Entry("managed name", "service-account", true),
		Entry("minimum length", "abc", true),
		Entry("maximum length", "a"+strings.Repeat("b", 62), true),
		Entry("empty name", "", false),
		Entry("too short", "ab", false),
		Entry("too long", "a"+strings.Repeat("b", 63), false),
		Entry("starts with a number", "1service", false),
		Entry("starts with a dash", "-service", false),
		Entry("ends with a dash", "service-", false),
		Entry("contains uppercase letters", "Service", false),
		Entry("contains an underscore", "service_account", false),
		Entry("contains a dot", "service.account", false),
	)

	newPrincipalWithKind := func(
		kind vedro.PrincipalKind,
		policy vedro.PrincipalManagementPolicy,
		name string,
		externalName string,
	) vedro.CloudPrincipal {
		return cloudtest.NewPrincipalCR(
			"service-account",
			func(cp *vedro.CloudPrincipal) {
				cp.Spec.Kind = kind
				cp.Spec.ManagementPolicy = policy
				if policy == vedro.PrincipalManagementPolicyManaged {
					cp.Spec.Managed = &vedro.ManagedPrincipalSpec{Name: name}
				} else {
					cp.Spec.Managed = nil
					cp.Spec.Reference = &vedro.ReferencedPrincipalSpec{Name: name}
				}
				cp.Status = vedro.CloudPrincipalStatus{ExternalName: externalName}
			},
		)
	}

	DescribeTable("validates referenced YC service account names",
		func(name string, valid bool) {
			result := (&Principal{}).ValidatePrincipalSpec(newPrincipalWithKind(
				vedro.PrincipalKindServiceAccount,
				vedro.PrincipalManagementPolicyReference,
				name,
				"",
			))

			Expect(result.Valid).To(Equal(valid))
			if !valid {
				Expect(result.Message).To(ContainSubstring(
					"must match <folder-name>:<service-account-name>",
				))
			}
		},
		Entry("folder and service account", "prod:some-sa", true),
		Entry("minimum lengths", "abc:def", true),
		Entry("empty name", "", false),
		Entry("no colon", "prod-sa", false),
		Entry("empty service account", "prod:", false),
		Entry("empty folder", ":some-sa", false),
		Entry("extra colon", "prod:some-sa:extra", false),
		Entry("uppercase folder", "PROD:some-sa", false),
		Entry("underscore in service account", "prod:some_sa", false),
		Entry("folder too short", "a:sa", false),
		Entry("service account too short", "prod:sa", false),
		Entry("email format", "some-sa@some-project.iam.gserviceaccount.com", false),
	)

	DescribeTable("validates YC user names",
		func(policy vedro.PrincipalManagementPolicy, name string, valid bool) {
			result := (&Principal{}).ValidatePrincipalSpec(newPrincipalWithKind(
				vedro.PrincipalKindUser,
				policy,
				name,
				"",
			))

			Expect(result.Valid).To(Equal(valid))
			if !valid {
				Expect(result.Message).To(ContainSubstring("must be valid emails"))
			}
		},
		Entry("referenced user email", vedro.PrincipalManagementPolicyReference, "user@example.com", true),
		Entry("referenced user not an email", vedro.PrincipalManagementPolicyReference, "user", false),
		Entry("referenced user with subject prefix", vedro.PrincipalManagementPolicyReference, "userAccount:abc123", false),
		Entry("managed user email", vedro.PrincipalManagementPolicyManaged, "user@example.com", true),
		Entry("managed user not an email", vedro.PrincipalManagementPolicyManaged, "user", false),
	)

	It("does not check name immutability for referenced principals", func() {
		result := (&Principal{}).ValidatePrincipalSpec(newPrincipalWithKind(
			vedro.PrincipalKindServiceAccount,
			vedro.PrincipalManagementPolicyReference,
			"prod:new-sa",
			"old-name",
		))

		Expect(result.Valid).To(BeTrue())
	})
})
