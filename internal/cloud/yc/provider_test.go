package yc

import (
	"context"
	"errors"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	vedro "github.com/svetoch-dev/vedro/api/v1alpha1"
	cloudtest "github.com/svetoch-dev/vedro/internal/cloud/test"
)

type cleanupSDK struct {
	shutdownCalled bool
	shutdownErr    error
}

func (f *cleanupSDK) Shutdown(context.Context) error {
	f.shutdownCalled = true
	return f.shutdownErr
}

var _ = Describe("Provider.Cleanup", func() {
	It("shuts down the SDK when closing the bucket API fails", func() {
		bucketErr := errors.New("close bucket API")
		shutdownErr := errors.New("shut down SDK")
		bucketAPI := &cloudtest.FakeBucketAPI{CloseErr: bucketErr}
		sdk := &cleanupSDK{shutdownErr: shutdownErr}
		provider := &Provider{
			bucket: &Bucket{api: bucketAPI},
			sdk:    sdk,
		}

		err := provider.Cleanup(context.Background())

		Expect(bucketAPI.CloseCalled).To(BeTrue())
		Expect(sdk.shutdownCalled).To(BeTrue())
		Expect(errors.Is(err, bucketErr)).To(BeTrue())
		Expect(errors.Is(err, shutdownErr)).To(BeTrue())
	})
})

var _ = Describe("Provider.ValidateProviderConfigSpec", func() {
	newProviderConfig := func(projectID string) vedro.ProviderConfig {
		return vedro.ProviderConfig{
			Spec: vedro.ProviderConfigSpec{
				ProjectId: projectID,
				Region:    "ru-central1",
			},
		}
	}

	It("returns valid for a valid YC project ID", func() {
		result := (&Provider{}).ValidateProviderConfigSpec(newProviderConfig("b1g123456789abcdefdd"))

		Expect(result.Valid).To(BeTrue())
	})

	It("returns invalid when the YC project ID is empty", func() {
		result := (&Provider{}).ValidateProviderConfigSpec(newProviderConfig(""))

		Expect(result.Valid).To(BeFalse())
		Expect(result.Message).To(ContainSubstring("spec.projectId"))
	})

	It("returns invalid when the YC project ID has an unsupported format", func() {
		result := (&Provider{}).ValidateProviderConfigSpec(newProviderConfig("B1G_123"))

		Expect(result.Valid).To(BeFalse())
		Expect(result.Message).To(ContainSubstring("lowercase"))
	})
})
