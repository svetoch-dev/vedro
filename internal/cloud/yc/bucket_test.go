package yc

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	vedro "github.com/svetoch-dev/vedro/api/v1alpha1"
	"github.com/svetoch-dev/vedro/internal/cloud"
	cloudtest "github.com/svetoch-dev/vedro/internal/cloud/test"
)

// Provider-agnostic EnsureBucket/DeleteBucket behaviour lives in the shared
// cloudtest package; only GCP specifics are configured here.
var _ = cloudtest.BucketProviderTests(cloudtest.Config{
	Location:                "ru-central1",
	NormalizedLocation:      "ru-central1",
	OtherLocation:           "kz1",
	OtherNormalizedLocation: "kz1",
	ProviderConfigType:      vedro.ProviderTypeYandexCloud,
	BucketCaps:              (&Provider{}).Capabilities().Bucket,
	NewBucket: func(api cloud.BucketAPI) cloud.BucketProvider {
		return &Bucket{api: api}
	},
})

var _ = cloudtest.BucketValidationTests(cloudtest.Config{
	Location:                "ru-central1",
	NormalizedLocation:      "ru-central1",
	OtherLocation:           "kz1",
	OtherNormalizedLocation: "kz1",
	ProviderConfigType:      vedro.ProviderTypeYandexCloud,
	NewBucket: func(api cloud.BucketAPI) cloud.BucketProvider {
		return &Bucket{api: api}
	},
})

var _ = Describe("BucketAccess.EnsureBucketAccess", func() {
	It("uses the bucket external ID for YC access operations", func() {
		api := &cloudtest.FakeBucketAPI{}
		provider := &BucketAccess{api: api}
		bucket := vedro.Bucket{
			Status: vedro.BucketStatus{
				ExternalName: "my-bucket",
				ExternalId:   "e3r-resource-id",
			},
		}
		principal := vedro.CloudPrincipal{
			Status: vedro.CloudPrincipalStatus{
				ExternalId: "service-account-id",
			},
		}
		access := vedro.BucketAccess{
			Spec: vedro.BucketAccessSpec{
				Access: vedro.Access{Level: vedro.ObjectReader},
			},
		}

		result, err := provider.EnsureBucketAccess(
			context.Background(),
			bucket,
			principal,
			access,
		)

		Expect(err).NotTo(HaveOccurred())
		Expect(api.HasAccessInputs).To(ConsistOf(cloud.BucketAccessAttrs{
			BucketName:    "my-bucket",
			BucketId:      "e3r-resource-id",
			PrincipalId:   "service-account-id",
			GrantedAccess: vedro.ObjectReader,
		}))
		Expect(api.GrantAccessInputs).To(Equal(api.HasAccessInputs))
		Expect(result.BucketName).To(Equal("my-bucket"))
		Expect(result.BucketId).To(Equal("e3r-resource-id"))
	})
})
