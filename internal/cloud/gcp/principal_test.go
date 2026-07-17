package gcp

import (
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

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

var _ = Describe("Principal.ValidateGCPPrincipalSpec", func() {
	newPrincipal := func(metadataName, specName, externalName string) vedro.CloudPrincipal {
		return vedro.CloudPrincipal{
			ObjectMeta: metav1.ObjectMeta{Name: metadataName},
			Spec:       vedro.CloudPrincipalSpec{Name: specName},
			Status:     vedro.CloudPrincipalStatus{ExternalName: externalName},
		}
	}

	DescribeTable("validates GCP service account names",
		func(metadataName, specName string, valid bool) {
			result := (&Principal{}).ValidatePrincipalSpec(newPrincipal(metadataName, specName, ""))

			Expect(result.Valid).To(Equal(valid))
			if !valid {
				Expect(result.Message).To(ContainSubstring("principal name must be 6-30 characters"))
			}
		},
		Entry("metadata name", "service-account", "", true),
		Entry("spec name override", "invalid.metadata.name", "service-account", true),
		Entry("minimum length", "abcdef", "", true),
		Entry("maximum length", "", "a"+strings.Repeat("b", 29), true),
		Entry("empty name", "", "", false),
		Entry("too short", "abcde", "", false),
		Entry("too long", "", "a"+strings.Repeat("b", 30), false),
		Entry("starts with a number", "1service", "", false),
		Entry("starts with a dash", "-service", "", false),
		Entry("ends with a dash", "service-", "", false),
		Entry("contains uppercase letters", "Service", "", false),
		Entry("contains an underscore", "service_account", "", false),
		Entry("contains a dot", "service.account", "", false),
	)

	It("rejects a changed spec name after creation", func() {
		result := (&Principal{}).ValidatePrincipalSpec(
			newPrincipal("principal", "new-principal", "old-principal"),
		)

		Expect(result.Valid).To(BeFalse())
		Expect(result.Message).To(Equal("spec.name cannot be changed after creation"))
	})

	It("rejects switching from spec.name to metadata.name after creation", func() {
		result := (&Principal{}).ValidatePrincipalSpec(
			newPrincipal("metadata-principal", "", "spec-principal"),
		)

		Expect(result.Valid).To(BeFalse())
		Expect(result.Message).To(Equal(
			"metadata.name cannot be used as the name source if spec.name was used and CR is created",
		))
	})
})
