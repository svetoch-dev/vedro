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
				Expect(result.Message).To(ContainSubstring("principal name must be 3-63 characters"))
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
})
