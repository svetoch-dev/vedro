package gcp

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	vedro "github.com/svetoch-dev/vedro/api/v1alpha1"
	"github.com/svetoch-dev/vedro/internal/cloud"
	cloudtest "github.com/svetoch-dev/vedro/internal/cloud/test"
	"github.com/svetoch-dev/vedro/internal/helpers"
)

var defaultCloudSpecific = vedro.BucketCloudSpecificConfig{
	Gcp: &vedro.BucketGcpConfig{
		SoftDeletePolicy: defaultSoftDelete,
	},
}

// newBucketCR is a small package-local helper so the GCP-specific specs
// (ValidateBucketSpec, and cloudSpecific case) stay concise.
func newBucketCR(name string, location string, mods ...func(*vedro.Bucket)) vedro.Bucket {
	return cloudtest.NewBucketCR(name, location, mods...)
}

// Provider-agnostic EnsureBucket/DeleteBucket behaviour lives in the shared
// cloudtest package; only GCP specifics are configured here.
var _ = cloudtest.BucketProviderTests(cloudtest.Config{
	Location:                "europe-west1",
	NormalizedLocation:      "EUROPE-WEST1",
	OtherLocation:           "us-central1",
	OtherNormalizedLocation: "US-CENTRAL1",
	ProviderConfigType:      vedro.ProviderTypeGCP,
	DefaultBucketPropertiesMods: []func(*vedro.BucketProperties){
		func(p *vedro.BucketProperties) {
			p.CloudSpecificConfig = &defaultCloudSpecific
		},
	},
	NewBucket: func(api cloud.BucketAPI) cloud.BucketProvider {
		return &Bucket{api: api}
	},
})

var _ = cloudtest.BucketValidationTests(cloudtest.Config{
	Location:                "europe-west1",
	NormalizedLocation:      "EUROPE-WEST1",
	OtherLocation:           "us-central1",
	OtherNormalizedLocation: "US-CENTRAL1",
	ProviderConfigType:      vedro.ProviderTypeGCP,
	NewBucket: func(api cloud.BucketAPI) cloud.BucketProvider {
		return &Bucket{api: api}
	},
})

var _ = Describe("BucketProvider.EnsureBucketGCP", func() {
	var (
		ctx    context.Context
		fake   *cloudtest.FakeBucketAPI
		bucket cloud.BucketProvider
	)

	BeforeEach(func() {
		ctx = context.Background()
		fake = &cloudtest.FakeBucketAPI{}
		bucket = &Bucket{api: fake}
	})

	It("creates a bucket with gcp specfic options", func() {
		fake.AttrsErr = cloud.ErrBucketNotFound

		gcpConfig := vedro.BucketCloudSpecificConfig{
			Gcp: &vedro.BucketGcpConfig{
				SoftDeletePolicy: &vedro.SoftDeletePolicy{
					RetentionDuration: v1.Duration{
						Duration: 24 * time.Hour,
					},
				},
			},
		}

		bckt := newBucketCR("my-bucket", "us-central1", func(b *vedro.Bucket) {
			b.Spec.StorageClass = vedro.BucketStorageClassStandard
			b.Spec.CloudSpecificConfig = &gcpConfig
		})

		attrs, err := bucket.EnsureBucket(ctx, bckt)
		Expect(err).NotTo(HaveOccurred())

		Expect(fake.Created).NotTo(BeNil())
		Expect(fake.Created.Location).To(Equal("us-central1"))
		Expect(fake.Created.Properties.StorageClass).To(Equal(vedro.BucketStorageClassStandard))
		Expect(fake.Created.Properties.CloudSpecificConfig).NotTo(BeNil())
		Expect(*fake.Created.Properties.CloudSpecificConfig).To(Equal(gcpConfig))
		Expect(attrs.Properties.CloudSpecificConfig).NotTo(BeNil())
		Expect(*attrs.Properties.CloudSpecificConfig).To(Equal(gcpConfig))
	})
	It("falls back to default cloudSpecificConfig if none cloudSpecific gcp options are passed", func() {
		fake.AttrsErr = cloud.ErrBucketNotFound
		ycConfig := vedro.BucketCloudSpecificConfig{
			Yc: &vedro.BucketYcConfig{},
		}

		bckt := newBucketCR("my-bucket", "us-central1", func(b *vedro.Bucket) {
			b.Spec.StorageClass = vedro.BucketStorageClassStandard
			b.Spec.CloudSpecificConfig = &ycConfig
		})

		attrs, err := bucket.EnsureBucket(ctx, bckt)
		Expect(err).NotTo(HaveOccurred())
		Expect(fake.Created).NotTo(BeNil())
		Expect(fake.Created.Location).To(Equal("us-central1"))
		Expect(fake.Created.Properties.StorageClass).To(Equal(vedro.BucketStorageClassStandard))
		Expect(*fake.Created.Properties.CloudSpecificConfig).To(Equal(defaultCloudSpecific))
		Expect(*attrs.Properties.CloudSpecificConfig).To(Equal(defaultCloudSpecific))
	})

	It("updates cloudSpecificConfig when it differs", func() {
		got := vedro.BucketCloudSpecificConfig{
			Gcp: &vedro.BucketGcpConfig{
				SoftDeletePolicy: &vedro.SoftDeletePolicy{
					RetentionDuration: v1.Duration{
						Duration: 24 * 7 * time.Hour,
					},
				},
			},
		}
		want := got.DeepCopy()
		want.Gcp.SoftDeletePolicy.RetentionDuration = v1.Duration{
			Duration: 0,
		}
		fake.Attrs = cloudtest.NewBucketAttrs(
			"my-bucket", "EU-WEST1",
			vedro.BucketStorageClassStandard,
			func(p *vedro.BucketProperties) {
				p.CloudSpecificConfig = &got
			},
		)

		bckt := newBucketCR(
			"my-bucket", "EU-WEST1",
			func(b *vedro.Bucket) {
				b.Spec.StorageClass = vedro.BucketStorageClassStandard
				b.Spec.CloudSpecificConfig = want
			})

		attrs, err := bucket.EnsureBucket(ctx, bckt)
		Expect(err).NotTo(HaveOccurred())
		Expect(fake.Updated).NotTo(BeNil())
		Expect(fake.Updated.CloudSpecificConfig).To(Equal(helpers.PatchTo(want)))
		Expect(*attrs.Properties.CloudSpecificConfig).To(Equal(*want))
	})
	It("does not update cloudSpecificConfig when its not gcp", func() {
		ycConfig := vedro.BucketCloudSpecificConfig{
			Yc: &vedro.BucketYcConfig{},
		}
		got := vedro.BucketCloudSpecificConfig{
			Gcp: &vedro.BucketGcpConfig{
				SoftDeletePolicy: &vedro.SoftDeletePolicy{
					RetentionDuration: v1.Duration{
						Duration: 24 * 7 * time.Hour,
					},
				},
			},
		}
		fake.Attrs = cloudtest.NewBucketAttrs(
			"my-bucket", "EU-WEST1",
			vedro.BucketStorageClassStandard,
			func(p *vedro.BucketProperties) {
				p.CloudSpecificConfig = &got
			},
		)

		bckt := newBucketCR(
			"my-bucket", "EU-WEST1",
			func(b *vedro.Bucket) {
				b.Spec.StorageClass = vedro.BucketStorageClassStandard
				b.Spec.CloudSpecificConfig = &ycConfig
			})

		_, err := bucket.EnsureBucket(ctx, bckt)
		Expect(err).NotTo(HaveOccurred())
		Expect(fake.Updated).To(BeNil())
	})

	It("does not update cloudSpecificConfig when gcp options are empty", func() {
		emptyGCPConfig := vedro.BucketCloudSpecificConfig{
			Gcp: &vedro.BucketGcpConfig{},
		}
		fake.Attrs = cloudtest.NewBucketAttrs(
			"my-bucket", "EU-WEST1",
			vedro.BucketStorageClassStandard,
			func(p *vedro.BucketProperties) {
				p.CloudSpecificConfig = &defaultCloudSpecific
			},
		)

		bckt := newBucketCR(
			"my-bucket", "EU-WEST1",
			func(b *vedro.Bucket) {
				b.Spec.StorageClass = vedro.BucketStorageClassStandard
				b.Spec.CloudSpecificConfig = &emptyGCPConfig
			})

		_, err := bucket.EnsureBucket(ctx, bckt)
		Expect(err).NotTo(HaveOccurred())
		Expect(fake.Updated).To(BeNil())
	})

})

var _ = Describe("Bucket.ValidateBucketSpecGCP", func() {
	var bucket *Bucket

	BeforeEach(func() {
		bucket = &Bucket{}
	})
	It("accepts multi-region locations", func() {
		bckt := newBucketCR("my-bucket", "us")

		result := bucket.ValidateBucketSpec(bckt, vedro.ProviderTypeGCP)
		Expect(result.Valid).To(BeTrue())
	})

	It("accepts dual-region locations", func() {
		bckt := newBucketCR("my-bucket", "NAM4")

		result := bucket.ValidateBucketSpec(bckt, vedro.ProviderTypeGCP)
		Expect(result.Valid).To(BeTrue())
	})
	It("returns an Invalid when the bucket name uses a reserved google prefix", func() {
		bckt := newBucketCR("google-bucket", "europe-west1")

		result := bucket.ValidateBucketSpec(bckt, vedro.ProviderTypeGCP)
		Expect(result.Valid).To(BeFalse())
		Expect(result.Message).To(ContainSubstring("reserved Google-related names"))
	})
	It("returns an Invalid when cloudSpecificConfigs is not gcp", func() {
		bckt := newBucketCR("my-bucket", "europe-west1", func(v *vedro.Bucket) {
			v.Spec.CloudSpecificConfig = &vedro.BucketCloudSpecificConfig{
				Yc: &vedro.BucketYcConfig{},
			}
		})

		result := bucket.ValidateBucketSpec(bckt, vedro.ProviderTypeGCP)
		Expect(result.Valid).To(BeFalse())
		Expect(result.Message).To(ContainSubstring("can only be used with provider type yc"))
	})

	It("returns an Invalid when soft delete retention is below the GCS minimum", func() {
		bckt := newBucketCR("my-bucket", "europe-west1", func(v *vedro.Bucket) {
			v.Spec.CloudSpecificConfig = &vedro.BucketCloudSpecificConfig{
				Gcp: &vedro.BucketGcpConfig{
					SoftDeletePolicy: &vedro.SoftDeletePolicy{
						RetentionDuration: v1.Duration{Duration: 24 * time.Hour},
					},
				},
			}
		})

		result := bucket.ValidateBucketSpec(bckt, vedro.ProviderTypeGCP)
		Expect(result.Valid).To(BeFalse())
		Expect(result.Message).To(ContainSubstring("must be 0 or between 7 and 90 days"))
	})

	It("returns an Invalid when soft delete retention is above the GCS maximum", func() {
		bckt := newBucketCR("my-bucket", "europe-west1", func(v *vedro.Bucket) {
			v.Spec.CloudSpecificConfig = &vedro.BucketCloudSpecificConfig{
				Gcp: &vedro.BucketGcpConfig{
					SoftDeletePolicy: &vedro.SoftDeletePolicy{
						RetentionDuration: v1.Duration{Duration: 91 * 24 * time.Hour},
					},
				},
			}
		})

		result := bucket.ValidateBucketSpec(bckt, vedro.ProviderTypeGCP)
		Expect(result.Valid).To(BeFalse())
		Expect(result.Message).To(ContainSubstring("must be 0 or between 7 and 90 days"))
	})

})
