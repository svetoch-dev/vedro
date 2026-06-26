package cloudtest

import (
	"context"
	"errors"

	"cloud.google.com/go/storage"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	vedro "github.com/svetoch-dev/vedro/api/v1alpha1"
	"github.com/svetoch-dev/vedro/internal/cloud"
	"github.com/svetoch-dev/vedro/internal/helpers"
)

// Config parameterizes the shared bucket lifecycle specs with the details
// that differ between providers.
type Config struct {
	// Location is a location the provider accepts in spec.location.
	Location string
	// NormalizedLocation is how the provider stores/returns Location
	// (e.g. GCP upper-cases it: "europe-west1" -> "EUROPE-WEST1").
	NormalizedLocation string

	// OtherLocation is a different valid location, used to test the
	// "bucket already exists in another location" case.
	OtherLocation string
	// OtherNormalizedLocation is the normalized form of OtherLocation.
	OtherNormalizedLocation string

	// NewBucket wires the provider's cloud.BucketProvider to the supplied
	// fake API. Implemented inside each provider's package so it can reach
	// unexported fields.
	NewBucket func(api cloud.BucketAPI) cloud.BucketProvider
}

// BucketProviderTests registers the provider-agnostic EnsureBucket and
// DeleteBucket specs. Call it from each provider's Ginkgo suite, e.g.:
//
//	var _ = cloudtest.BucketProviderTests(cloudtest.Config{...})
func BucketProviderTests(cfg Config) bool {
	newBucketCR := func(mods ...func(*vedro.Bucket)) vedro.Bucket {
		return NewBucketCR("my-bucket", cfg.Location, mods...)
	}

	Describe("BucketProvider.EnsureBucket", func() {
		var (
			ctx    context.Context
			fake   *FakeBucketAPI
			bucket cloud.BucketProvider
		)

		BeforeEach(func() {
			ctx = context.Background()
			fake = &FakeBucketAPI{}
			bucket = cfg.NewBucket(fake)
		})

		It("creates a bucket when it does not exist", func() {
			fake.AttrsErr = cloud.ErrBucketNotFound

			bckt := newBucketCR(func(b *vedro.Bucket) {
				b.Spec.StorageClass = vedro.BucketStorageClassStandard
			})

			attrs, err := bucket.EnsureBucket(ctx, bckt)
			Expect(err).NotTo(HaveOccurred())
			Expect(fake.Created).NotTo(BeNil())
			Expect(fake.Created.Location).To(Equal(cfg.Location))
			Expect(fake.Created.Properties.StorageClass).To(Equal(vedro.BucketStorageClassStandard))
			Expect(attrs.Name).To(Equal("my-bucket"))
			Expect(attrs.Location).To(Equal(cfg.Location))
			Expect(attrs.Properties).NotTo(BeNil())
			Expect(attrs.Properties.StorageClass).To(Equal(vedro.BucketStorageClassStandard))
		})

		It("creates a bucket with all supported options", func() {
			fake.AttrsErr = cloud.ErrBucketNotFound

			publicAccessPrevention := true
			bckt := newBucketCR(func(b *vedro.Bucket) {
				b.Spec.StorageClass = vedro.BucketStorageClassIce
				b.Spec.Labels = map[string]string{"env": "prod"}
				b.Spec.Versioning = &vedro.BucketVersioning{Enabled: true}
				b.Spec.PublicAccessPrevention = &publicAccessPrevention
				b.Spec.Lifecycle = &Lifecycle
			})

			attrs, err := bucket.EnsureBucket(ctx, bckt)
			Expect(err).NotTo(HaveOccurred())

			Expect(fake.Created).NotTo(BeNil())
			Expect(fake.Created.Location).To(Equal(cfg.Location))
			Expect(fake.Created.Properties.StorageClass).To(Equal(vedro.BucketStorageClassIce))
			Expect(fake.Created.Properties.Labels).To(Equal(map[string]string{"env": "prod"}))
			Expect(fake.Created.Properties.Versioning.Enabled).To(BeTrue())
			Expect(fake.Created.Properties.Lifecycle).To(Equal(&Lifecycle))
			Expect(*fake.Created.Properties.PublicAccessPrevention).To(BeTrue())
			Expect(attrs.Properties.StorageClass).To(Equal(vedro.BucketStorageClassIce))
			Expect(attrs.Properties.Labels).To(Equal(map[string]string{"env": "prod"}))
			Expect(attrs.Properties.Versioning.Enabled).To(BeTrue())
			Expect(*attrs.Properties.PublicAccessPrevention).To(BeTrue())
		})

		It("returns an error when creating a bucket fails", func() {
			fake.AttrsErr = cloud.ErrBucketNotFound
			fake.CreateErr = errors.New("network error")

			bckt := newBucketCR(func(b *vedro.Bucket) {
				b.Spec.StorageClass = vedro.BucketStorageClassStandard
			})

			_, err := bucket.EnsureBucket(ctx, bckt)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("create bucket \"my-bucket\""))
			Expect(err.Error()).To(ContainSubstring("network error"))
		})

		It("returns an error when fetching bucket attributes fails", func() {
			fake.AttrsErr = errors.New("permission denied")

			bckt := newBucketCR(func(b *vedro.Bucket) {
				b.Spec.StorageClass = vedro.BucketStorageClassStandard
			})

			_, err := bucket.EnsureBucket(ctx, bckt)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("get bucket attrs \"my-bucket\""))
			Expect(err.Error()).To(ContainSubstring("permission denied"))
		})

		It("returns the existing attrs when the bucket already matches the spec", func() {
			fake.Attrs = NewBucketAttrs(
				"my-bucket", cfg.NormalizedLocation, vedro.BucketStorageClassStandard,
			)

			bckt := newBucketCR(func(b *vedro.Bucket) {
				b.Spec.StorageClass = vedro.BucketStorageClassStandard
			})

			attrs, err := bucket.EnsureBucket(ctx, bckt)
			Expect(err).NotTo(HaveOccurred())
			Expect(fake.Updated).To(BeNil())
			Expect(attrs.Name).To(Equal("my-bucket"))
			Expect(attrs.Location).To(Equal(cfg.NormalizedLocation))
		})

		It("returns an error when the existing bucket is in a different location", func() {
			fake.Attrs = NewBucketAttrs(
				"my-bucket", cfg.OtherNormalizedLocation, vedro.BucketStorageClassStandard,
			)

			bckt := newBucketCR(func(b *vedro.Bucket) {
				b.Spec.StorageClass = vedro.BucketStorageClassStandard
			})

			_, err := bucket.EnsureBucket(ctx, bckt)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("already exists in location \"" + cfg.OtherNormalizedLocation + "\""))
			Expect(err.Error()).To(ContainSubstring("desired location is \"" + cfg.NormalizedLocation + "\""))
		})

		It("updates the storage class when it differs", func() {
			fake.Attrs = NewBucketAttrs(
				"my-bucket", cfg.NormalizedLocation, vedro.BucketStorageClassStandard,
			)

			bckt := newBucketCR(func(b *vedro.Bucket) {
				b.Spec.StorageClass = vedro.BucketStorageClassWarm
			})

			attrs, err := bucket.EnsureBucket(ctx, bckt)
			Expect(err).NotTo(HaveOccurred())
			Expect(fake.Updated).NotTo(BeNil())
			Expect(fake.Updated.StorageClass).To(Equal(
				helpers.PatchTo(vedro.BucketStorageClassWarm),
			))
			Expect(attrs.Properties.StorageClass).To(Equal(vedro.BucketStorageClassWarm))
		})

		It("updates lifecycle when its empty", func() {
			fake.Attrs = NewBucketAttrs(
				"my-bucket", cfg.NormalizedLocation, vedro.BucketStorageClassStandard,
			)

			bckt := newBucketCR(func(b *vedro.Bucket) {
				b.Spec.StorageClass = vedro.BucketStorageClassStandard
				b.Spec.Lifecycle = &Lifecycle
			})

			attrs, err := bucket.EnsureBucket(ctx, bckt)
			Expect(err).NotTo(HaveOccurred())
			Expect(fake.Updated).NotTo(BeNil())
			Expect(fake.Updated.Lifecycle).To(Equal(helpers.PatchTo(&Lifecycle)))
			Expect(attrs.Properties.Lifecycle).To(Equal(&Lifecycle))
		})

		It("updates lifecycle when it differs", func() {
			actualLifecycle := Lifecycle.DeepCopy()
			actualLifecycle.Rules[0].AgeDays = helpers.Ptr(int64(100000))
			fake.Attrs = NewBucketAttrs(
				"my-bucket", cfg.NormalizedLocation, vedro.BucketStorageClassStandard,
				func(p *vedro.BucketProperties) { p.Lifecycle = actualLifecycle },
			)

			bckt := newBucketCR(func(b *vedro.Bucket) {
				b.Spec.StorageClass = vedro.BucketStorageClassStandard
				b.Spec.Lifecycle = &Lifecycle
			})

			attrs, err := bucket.EnsureBucket(ctx, bckt)
			Expect(err).NotTo(HaveOccurred())
			Expect(fake.Updated).NotTo(BeNil())
			Expect(fake.Updated.Lifecycle).To(Equal(helpers.PatchTo(&Lifecycle)))
			Expect(attrs.Properties.Lifecycle).To(Equal(&Lifecycle))
		})

		It("updates versioning when it differs", func() {
			fake.Attrs = NewBucketAttrs(
				"my-bucket", cfg.NormalizedLocation, vedro.BucketStorageClassStandard,
				func(p *vedro.BucketProperties) {
					p.Versioning = &vedro.BucketVersioning{Enabled: false}
				},
			)

			bckt := newBucketCR(func(b *vedro.Bucket) {
				b.Spec.StorageClass = vedro.BucketStorageClassStandard
				b.Spec.Versioning = &vedro.BucketVersioning{Enabled: true}
			})

			attrs, err := bucket.EnsureBucket(ctx, bckt)
			Expect(err).NotTo(HaveOccurred())
			Expect(fake.Updated).NotTo(BeNil())
			Expect(fake.Updated.Versioning).To(Equal(
				helpers.PatchTo(&vedro.BucketVersioning{Enabled: true}),
			))
			Expect(attrs.Properties.Versioning.Enabled).To(BeTrue())
		})

		It("updates labels when they differ", func() {
			fake.Attrs = NewBucketAttrs(
				"my-bucket", cfg.NormalizedLocation, vedro.BucketStorageClassStandard,
				func(p *vedro.BucketProperties) {
					p.Labels = map[string]string{"env": "dev"}
				},
			)

			bckt := newBucketCR(func(b *vedro.Bucket) {
				b.Spec.StorageClass = vedro.BucketStorageClassStandard
				b.Spec.Labels = map[string]string{"env": "prod"}
			})

			attrs, err := bucket.EnsureBucket(ctx, bckt)
			Expect(err).NotTo(HaveOccurred())
			Expect(fake.Updated).NotTo(BeNil())
			Expect(fake.Updated.Labels).To(Equal(helpers.PatchTo(map[string]string{"env": "prod"})))
			Expect(attrs.Properties.Labels).To(Equal(map[string]string{"env": "prod"}))
		})

		It("updates public access prevention when it differs", func() {
			fake.Attrs = NewBucketAttrs(
				"my-bucket", cfg.NormalizedLocation, vedro.BucketStorageClassStandard,
				func(p *vedro.BucketProperties) {
					p.PublicAccessPrevention = helpers.Ptr(false)
				},
			)

			publicAccessPrevention := true
			bckt := newBucketCR(func(b *vedro.Bucket) {
				b.Spec.StorageClass = vedro.BucketStorageClassStandard
				b.Spec.PublicAccessPrevention = &publicAccessPrevention
			})

			attrs, err := bucket.EnsureBucket(ctx, bckt)
			Expect(err).NotTo(HaveOccurred())
			Expect(fake.Updated).NotTo(BeNil())
			Expect(fake.Updated.PublicAccessPrevention).To(Equal(helpers.PatchTo(&publicAccessPrevention)))
			Expect(*attrs.Properties.PublicAccessPrevention).To(BeTrue())
		})

		It("updates labels when spec.Labels is nil labels in status.Applied.Labels", func() {
			fake.Attrs = NewBucketAttrs(
				"my-bucket", cfg.NormalizedLocation, vedro.BucketStorageClassStandard,
				func(p *vedro.BucketProperties) {
					p.Labels = map[string]string{"env": "dev"}
				},
			)

			bckt := newBucketCR(func(b *vedro.Bucket) {
				b.Spec.StorageClass = vedro.BucketStorageClassStandard
				b.Status.Applied = &vedro.BucketProperties{
					StorageClass: vedro.BucketStorageClassStandard,
					Labels:       map[string]string{"env": "dev"},
				}
			})

			attrs, err := bucket.EnsureBucket(ctx, bckt)
			Expect(err).NotTo(HaveOccurred())
			Expect(fake.Updated).NotTo(BeNil())
			Expect(fake.Updated.Labels.Set).To(BeTrue())
			Expect(fake.Updated.Labels.Value).To(BeNil())
			Expect(attrs.Properties.Labels).To(BeEmpty())
		})

		It("returns an error when updating the bucket fails", func() {
			fake.Attrs = NewBucketAttrs(
				"my-bucket", cfg.NormalizedLocation, vedro.BucketStorageClassStandard,
			)
			fake.UpdateErr = errors.New("update failed")

			bckt := newBucketCR(func(b *vedro.Bucket) {
				b.Spec.StorageClass = vedro.BucketStorageClassWarm
			})

			_, err := bucket.EnsureBucket(ctx, bckt)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("update bucket \"my-bucket\""))
			Expect(err.Error()).To(ContainSubstring("update failed"))
		})
		It("returns an error for an unmapped storage class", func() {
			fake.AttrsErr = cloud.ErrBucketNotFound
			fake.CreateErr = errors.New("storage class NoneExistant is not supported")

			bckt := newBucketCR(func(b *vedro.Bucket) {
				b.Spec.StorageClass = vedro.BucketStorageClass("NoneExistant")
			})

			_, err := bucket.EnsureBucket(ctx, bckt)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("NoneExistant"))
		})

	})

	Describe("BucketProvider.DeleteBucket", func() {
		var (
			ctx    context.Context
			fake   *FakeBucketAPI
			bucket cloud.BucketProvider
		)

		BeforeEach(func() {
			ctx = context.Background()
			fake = &FakeBucketAPI{}
			bucket = cfg.NewBucket(fake)
		})

		newDeleteBucketCR := func(mods ...func(*vedro.Bucket)) vedro.Bucket {
			return newBucketCR(append([]func(*vedro.Bucket){
				func(b *vedro.Bucket) {
					b.Spec.DeletionPolicy = vedro.DeletionPolicyDelete
				},
			}, mods...)...)
		}

		It("does nothing when the deletion policy is not Delete", func() {
			bckt := newBucketCR(func(b *vedro.Bucket) {
				b.Spec.DeletionPolicy = vedro.DeletionPolicyRetain
			})

			err := bucket.DeleteBucket(ctx, bckt)
			Expect(err).NotTo(HaveOccurred())
			Expect(fake.Deleted).To(BeFalse())
			Expect(fake.Query).To(BeNil())
		})

		It("deletes an empty bucket", func() {
			bckt := newDeleteBucketCR()

			err := bucket.DeleteBucket(ctx, bckt)
			Expect(err).NotTo(HaveOccurred())
			Expect(fake.Deleted).To(BeTrue())
		})

		It("requests all object versions while listing", func() {
			fake.ObjectIterator = &FakeObjectIterator{}
			bckt := newDeleteBucketCR()

			err := bucket.DeleteBucket(ctx, bckt)
			Expect(err).NotTo(HaveOccurred())
			Expect(fake.Query).NotTo(BeNil())
			Expect(fake.Query.Versions).To(BeTrue())
		})

		It("deletes all objects before deleting the bucket", func() {
			fake.ObjectIterator = &FakeObjectIterator{
				Attrs: []*storage.ObjectAttrs{
					{Name: "obj-a", Generation: 1},
					{Name: "obj-b", Generation: 2},
				},
			}
			bckt := newDeleteBucketCR()

			err := bucket.DeleteBucket(ctx, bckt)
			Expect(err).NotTo(HaveOccurred())
			Expect(fake.Deleted).To(BeTrue())
			Expect(fake.GetDeletedObjects()).To(ConsistOf(
				DeletedObject{Name: "obj-a", Generation: 1},
				DeletedObject{Name: "obj-b", Generation: 2},
			))
		})

		It("returns an error when listing objects fails", func() {
			fake.ObjectIterator = &FakeObjectIterator{Err: errors.New("list failed")}
			bckt := newDeleteBucketCR()

			err := bucket.DeleteBucket(ctx, bckt)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("list failed"))
			Expect(fake.Deleted).To(BeFalse())
		})

		It("returns an error when object deletion fails", func() {
			fake.ObjectIterator = &FakeObjectIterator{
				Attrs: []*storage.ObjectAttrs{
					{Name: "obj-a", Generation: 1},
				},
			}
			fake.ObjectDeleteErr = errors.New("object delete failed")
			bckt := newDeleteBucketCR()

			err := bucket.DeleteBucket(ctx, bckt)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("could not delete bucket because object deletion failed"))
			Expect(err.Error()).To(ContainSubstring("object delete failed"))
			Expect(fake.Deleted).To(BeFalse())
		})

		It("ignores 404 errors while deleting objects", func() {
			fake.ObjectIterator = &FakeObjectIterator{
				Attrs: []*storage.ObjectAttrs{
					{Name: "obj-a", Generation: 1},
				},
			}
			fake.ObjectDeleteErr = cloud.ErrBucketObjectNotFound
			bckt := newDeleteBucketCR()

			err := bucket.DeleteBucket(ctx, bckt)
			Expect(err).NotTo(HaveOccurred())
			Expect(fake.Deleted).To(BeTrue())
		})

		It("ignores 404 errors when deleting the bucket", func() {
			fake.DeleteErr = cloud.ErrBucketNotFound
			bckt := newDeleteBucketCR()

			err := bucket.DeleteBucket(ctx, bckt)
			Expect(err).NotTo(HaveOccurred())
			Expect(fake.Deleted).To(BeTrue())
		})

		It("returns an error when bucket deletion fails", func() {
			fake.DeleteErr = errors.New("bucket delete failed")
			bckt := newDeleteBucketCR()

			err := bucket.DeleteBucket(ctx, bckt)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("bucket delete failed"))
			Expect(fake.Deleted).To(BeTrue())
		})
	})

	return true
}

func BucketValidationTests(cfg Config) bool {
	newBucketCR := func(mods ...func(*vedro.Bucket)) vedro.Bucket {
		return NewBucketCR("my-bucket", cfg.Location, mods...)
	}
	Describe("Bucket.ValidateBucketSpec", func() {
		var (
			fake   *FakeBucketAPI
			bucket cloud.BucketProvider
		)

		BeforeEach(func() {
			fake = &FakeBucketAPI{}
			bucket = cfg.NewBucket(fake)
		})

		It("returns valid for a correct bucket spec", func() {
			bckt := newBucketCR()

			result := bucket.ValidateBucketSpec(bckt)
			Expect(result.Valid).To(BeTrue())
		})

		It("returns valid when spec.name is used", func() {
			bckt := newBucketCR(func(b *vedro.Bucket) {
				b.Spec.Name = "actual-bucket-name"
			})

			result := bucket.ValidateBucketSpec(bckt)
			Expect(result.Valid).To(BeTrue())
		})

		It("returns an error when spec.name is changed after creation", func() {
			bckt := newBucketCR(func(b *vedro.Bucket) {
				b.Spec.Name = "new-name"
				b.Status.ExternalName = "old-name"
			})

			result := bucket.ValidateBucketSpec(bckt)
			Expect(result.Valid).To(BeFalse())
			Expect(result.Message).To(ContainSubstring("spec.name cannot be changed"))
		})

		It("returns an error when metadata.name is used after spec.name was used", func() {
			bckt := newBucketCR(func(b *vedro.Bucket) {
				b.Spec.Name = ""
				b.Status.ExternalName = "old-spec-name"
			})

			result := bucket.ValidateBucketSpec(bckt)
			Expect(result.Valid).To(BeFalse())
			Expect(result.Message).To(ContainSubstring("metadata.name cannot be used"))
		})

		It("returns an error when location is empty", func() {
			bckt := newBucketCR(func(b *vedro.Bucket) {
				b.Spec.Location = ""
			})

			result := bucket.ValidateBucketSpec(bckt)
			Expect(result.Valid).To(BeFalse())
			Expect(result.Message).To(ContainSubstring("location is an empty string"))
		})

		It("returns an error for an unsupported location", func() {
			bckt := newBucketCR(func(b *vedro.Bucket) {
				b.Spec.Location = "bad"
			})

			result := bucket.ValidateBucketSpec(bckt)
			Expect(result.Valid).To(BeFalse())
			Expect(result.Message).To(ContainSubstring("unsupported bucket location"))
		})

		It("returns an error when the bucket name is too short", func() {
			bckt := newBucketCR(func(b *vedro.Bucket) {
				b.Name = "b"
			})

			result := bucket.ValidateBucketSpec(bckt)
			Expect(result.Valid).To(BeFalse())
			Expect(result.Message).To(ContainSubstring("bucket name must be 3-63 characters"))
		})

		It("returns an error when the bucket name contains uppercase letters", func() {
			bckt := newBucketCR(func(b *vedro.Bucket) {
				b.Name = "My-Bucket"
			})

			result := bucket.ValidateBucketSpec(bckt)
			Expect(result.Valid).To(BeFalse())
			Expect(result.Message).To(ContainSubstring("bucket name must be 3-63 characters"))
		})

		It("returns an error when the bucket name contains consecutive dots", func() {
			bckt := newBucketCR(func(b *vedro.Bucket) {
				b.Name = "my..bucket"
			})

			result := bucket.ValidateBucketSpec(bckt)
			Expect(result.Valid).To(BeFalse())
			Expect(result.Message).To(ContainSubstring("consecutive dots"))
		})

		It("returns an error when the bucket name has dots next to dashes", func() {
			bckt := newBucketCR(func(b *vedro.Bucket) {
				b.Name = "my.-bucket"
			})

			result := bucket.ValidateBucketSpec(bckt)
			Expect(result.Valid).To(BeFalse())
			Expect(result.Message).To(ContainSubstring("dots next to dashes"))
		})
		It("returns an error when spec.name is invalid", func() {
			bckt := newBucketCR(func(b *vedro.Bucket) {
				b.Spec.Name = "INVALID"
			})

			result := bucket.ValidateBucketSpec(bckt)
			Expect(result.Valid).To(BeFalse())
			Expect(result.Message).To(ContainSubstring("bucket name must be 3-63 characters"))
		})
	})

	return true
}
