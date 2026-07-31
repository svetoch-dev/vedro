package yc

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	vedro "github.com/svetoch-dev/vedro/api/v1alpha1"
	"github.com/svetoch-dev/vedro/internal/cloud"
	cloudtest "github.com/svetoch-dev/vedro/internal/cloud/test"
)

var _ = cloudtest.BucketAccessProviderTests(cloudtest.Config{
	Location: "ru-central1",
	NewBucketAccess: func(api cloud.BucketAPI) cloud.BucketAccessProvider {
		return &BucketAccess{api: api}
	},
})

var _ = Describe("BucketAccess.EnsureBucketAccess", func() {
	It("uses the bucket external ID for YC access operations", func() {
		api := &cloudtest.FakeBucketAPI{}
		provider := &BucketAccess{api: api}
		bucket := cloudtest.NewBucketCR("test-bucket", "ru-central1", func(b *vedro.Bucket) {
			b.Status.ExternalName = "my-bucket"
			b.Status.ExternalId = "e3r-resource-id"
		})
		principal := cloudtest.NewPrincipalCR("test-principal", func(p *vedro.CloudPrincipal) {
			p.Status.ExternalId = "service-account-id"
		})
		access := cloudtest.NewBucketAccessCR("bucket-access")

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
