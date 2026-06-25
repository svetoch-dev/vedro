package gcp

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	vedrov1alpha1 "github.com/svetoch-dev/vedro/api/v1alpha1"
	"github.com/svetoch-dev/vedro/internal/cloud"
	cloudtest "github.com/svetoch-dev/vedro/internal/cloud/test"
)

// newBucketCR is a small package-local helper so the GCP-specific specs
// (ValidateBucketSpec, and the unmapped storage class case) stay concise.
func newBucketCR(name string, location string, mods ...func(*vedrov1alpha1.Bucket)) vedrov1alpha1.Bucket {
	return cloudtest.NewBucketCR(name, location, mods...)
}

// Provider-agnostic EnsureBucket/DeleteBucket behaviour lives in the shared
// cloudtest package; only GCP specifics are configured here.
var _ = cloudtest.BucketProviderTests(cloudtest.Config{
	Location:                "europe-west1",
	NormalizedLocation:      "EUROPE-WEST1",
	OtherLocation:           "us-central1",
	OtherNormalizedLocation: "US-CENTRAL1",
	NewBucket: func(api cloud.BucketAPI) cloud.BucketProvider {
		return &Bucket{api: api}
	},
})

var _ = cloudtest.BucketValidationTests(cloudtest.Config{
	Location:                "europe-west1",
	NormalizedLocation:      "EUROPE-WEST1",
	OtherLocation:           "us-central1",
	OtherNormalizedLocation: "US-CENTRAL1",
	NewBucket: func(api cloud.BucketAPI) cloud.BucketProvider {
		return &Bucket{api: api}
	},
})

var _ = Describe("Bucket.ValidateBucketSpec", func() {
	var bucket *Bucket

	BeforeEach(func() {
		bucket = &Bucket{}
	})
	It("accepts multi-region locations", func() {
		bckt := newBucketCR("my-bucket", "us")

		result := bucket.ValidateBucketSpec(bckt)
		Expect(result.Valid).To(BeTrue())
	})

	It("accepts dual-region locations", func() {
		bckt := newBucketCR("my-bucket", "NAM4")

		result := bucket.ValidateBucketSpec(bckt)
		Expect(result.Valid).To(BeTrue())
	})
	It("returns an error when the bucket name uses a reserved google prefix", func() {
		bckt := newBucketCR("google-bucket", "europe-west1")

		result := bucket.ValidateBucketSpec(bckt)
		Expect(result.Valid).To(BeFalse())
		Expect(result.Message).To(ContainSubstring("reserved Google-related names"))
	})
})
