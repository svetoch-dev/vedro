package capabilities

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	vedro "github.com/svetoch-dev/vedro/api/v1alpha1"
	"github.com/svetoch-dev/vedro/internal/cloud"
)

var bucket = vedro.BucketAccessSpec{
	BucketRef: vedro.BucketReference{
		Name:      "test-bucket",
		Namespace: "default",
	},
	PrincipalRef: vedro.PrincipalReference{
		Name:      "test-principal",
		Namespace: "default",
	},
}

var _ = Describe("ValidateBucketAccessCapabilities", func() {
	It("supported/set BucketAdmin", func() {
		caps := cloud.BucketAccessCapabilities{
			BucketAdmin: true,
		}
		bucket.Access.Level = vedro.BucketAdmin
		unsupported := ValidateBucketAccessCapabilities(caps, bucket)
		Expect(unsupported).To(BeEmpty())
	})
	It("unsupported/set BucketAdmin", func() {
		caps := cloud.BucketAccessCapabilities{
			BucketAdmin: false,
		}
		bucket.Access.Level = vedro.BucketAdmin
		unsupported := ValidateBucketAccessCapabilities(caps, bucket)
		want := []vedro.UnsupportedFeature{
			unsupportedFeatures["BucketAdmin"],
		}
		Expect(unsupported).NotTo(BeEmpty())
		Expect(unsupported).To(Equal(want))
	})
	It("supported/set ObjectReader", func() {
		caps := cloud.BucketAccessCapabilities{
			ObjectReader: true,
		}
		bucket.Access.Level = vedro.ObjectReader
		unsupported := ValidateBucketAccessCapabilities(caps, bucket)
		Expect(unsupported).To(BeEmpty())
	})
	It("unsupported/set ObjectReader", func() {
		caps := cloud.BucketAccessCapabilities{
			ObjectReader: false,
		}
		bucket.Access.Level = vedro.ObjectReader
		unsupported := ValidateBucketAccessCapabilities(caps, bucket)
		want := []vedro.UnsupportedFeature{
			unsupportedFeatures["ObjectReader"],
		}
		Expect(unsupported).NotTo(BeEmpty())
		Expect(unsupported).To(Equal(want))
	})
	It("supported/set ObjectWriter", func() {
		caps := cloud.BucketAccessCapabilities{
			ObjectWriter: true,
		}
		bucket.Access.Level = vedro.ObjectWriter
		unsupported := ValidateBucketAccessCapabilities(caps, bucket)
		Expect(unsupported).To(BeEmpty())
	})
	It("unsupported/set ObjectWriter", func() {
		caps := cloud.BucketAccessCapabilities{
			ObjectWriter: false,
		}
		bucket.Access.Level = vedro.ObjectWriter
		unsupported := ValidateBucketAccessCapabilities(caps, bucket)
		want := []vedro.UnsupportedFeature{
			unsupportedFeatures["ObjectWriter"],
		}
		Expect(unsupported).NotTo(BeEmpty())
		Expect(unsupported).To(Equal(want))
	})
	It("supported/set ObjectAdmin", func() {
		caps := cloud.BucketAccessCapabilities{
			ObjectAdmin: true,
		}
		bucket.Access.Level = vedro.ObjectAdmin
		unsupported := ValidateBucketAccessCapabilities(caps, bucket)
		Expect(unsupported).To(BeEmpty())
	})
	It("unsupported/set ObjectAdmin", func() {
		caps := cloud.BucketAccessCapabilities{
			ObjectAdmin: false,
		}
		bucket.Access.Level = vedro.ObjectAdmin
		unsupported := ValidateBucketAccessCapabilities(caps, bucket)
		want := []vedro.UnsupportedFeature{
			unsupportedFeatures["ObjectAdmin"],
		}
		Expect(unsupported).NotTo(BeEmpty())
		Expect(unsupported).To(Equal(want))
	})

})
