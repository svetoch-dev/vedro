package gcp

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	vedro "github.com/svetoch-dev/vedro/api/v1alpha1"
)

var _ = Describe("Provider.ValidateProviderConfigSpec", func() {
	newProviderConfig := func(projectID string) vedro.ProviderConfig {
		return vedro.ProviderConfig{
			Spec: vedro.ProviderConfigSpec{
				ProjectId: projectID,
				Region:    "us-central1",
			},
		}
	}

	It("returns valid for a valid GCP project ID", func() {
		result := (&Provider{}).ValidateProviderConfigSpec(newProviderConfig("test-project-1"))

		Expect(result.Valid).To(BeTrue())
	})

	It("returns invalid when the GCP project ID is empty", func() {
		result := (&Provider{}).ValidateProviderConfigSpec(newProviderConfig(""))

		Expect(result.Valid).To(BeFalse())
		Expect(result.Message).To(ContainSubstring("spec.projectId"))
	})

	It("returns invalid when the GCP project ID has an unsupported format", func() {
		result := (&Provider{}).ValidateProviderConfigSpec(newProviderConfig("Test_Project"))

		Expect(result.Valid).To(BeFalse())
		Expect(result.Message).To(ContainSubstring("lowercase"))
	})
})
