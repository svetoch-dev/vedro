package yc

import (
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	vedro "github.com/svetoch-dev/vedro/api/v1alpha1"
)

var _ = Describe("Principal.ValidatePrincipalSpec", func() {
	newPrincipal := func(metadataName, specName, externalName string) vedro.CloudPrincipal {
		return vedro.CloudPrincipal{
			ObjectMeta: metav1.ObjectMeta{Name: metadataName},
			Spec:       vedro.CloudPrincipalSpec{Name: specName},
			Status:     vedro.CloudPrincipalStatus{ExternalName: externalName},
		}
	}

	DescribeTable("validates YC service account names",
		func(metadataName, specName string, valid bool) {
			result := (&Principal{}).ValidatePrincipalSpec(newPrincipal(metadataName, specName, ""))

			Expect(result.Valid).To(Equal(valid))
			if !valid {
				Expect(result.Message).To(ContainSubstring("principal name must be 3-63 characters"))
			}
		},
		Entry("metadata name", "service-account", "", true),
		Entry("spec name override", "invalid.metadata.name", "service-account", true),
		Entry("minimum length", "abc", "", true),
		Entry("maximum length", "", "a"+strings.Repeat("b", 62), true),
		Entry("empty name", "", "", false),
		Entry("too short", "ab", "", false),
		Entry("too long", "", "a"+strings.Repeat("b", 63), false),
		Entry("starts with a number", "1service", "", false),
		Entry("starts with a dash", "-service", "", false),
		Entry("ends with a dash", "service-", "", false),
		Entry("contains uppercase letters", "Service", "", false),
		Entry("contains an underscore", "service_account", "", false),
		Entry("contains a dot", "service.account", "", false),
	)
})
