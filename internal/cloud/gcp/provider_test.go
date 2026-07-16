package gcp

import (
	"context"
	"errors"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	vedro "github.com/svetoch-dev/vedro/api/v1alpha1"
	"github.com/svetoch-dev/vedro/internal/cloud"
	cloudtest "github.com/svetoch-dev/vedro/internal/cloud/test"
)

type cleanupPrincipalAPI struct {
	cloud.PrincipalAPI
	closeCalled bool
	closeErr    error
}

func (f *cleanupPrincipalAPI) Close(context.Context) error {
	f.closeCalled = true
	return f.closeErr
}

var _ = Describe("Provider.Cleanup", func() {
	It("closes the principal API when closing the bucket API fails", func() {
		bucketErr := errors.New("close bucket API")
		principalErr := errors.New("close principal API")
		bucketAPI := &cloudtest.FakeBucketAPI{CloseErr: bucketErr}
		principalAPI := &cleanupPrincipalAPI{closeErr: principalErr}
		provider := &Provider{
			bucket:    &Bucket{api: bucketAPI},
			principal: &Principal{api: principalAPI},
		}

		err := provider.Cleanup(context.Background())

		Expect(bucketAPI.CloseCalled).To(BeTrue())
		Expect(principalAPI.closeCalled).To(BeTrue())
		Expect(errors.Is(err, bucketErr)).To(BeTrue())
		Expect(errors.Is(err, principalErr)).To(BeTrue())
	})
})

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
