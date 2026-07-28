package cloudtest

import (
	"context"
	"errors"

	. "github.com/onsi/ginkgo/v2" //nolint:staticcheck
	. "github.com/onsi/gomega"    //nolint:staticcheck

	vedro "github.com/svetoch-dev/vedro/api/v1alpha1"
	"github.com/svetoch-dev/vedro/internal/cloud"
	"github.com/svetoch-dev/vedro/internal/helpers"
)

// BucketProviderTests registers the provider-agnostic EnsureBucket and
// DeleteBucket specs. Call it from each provider's Ginkgo suite, e.g.:
//
//	var _ = cloudtest.BucketProviderTests(cloudtest.Config{...})
func BucketProviderTests(cfg Config) bool {
	newBucketCR := func(mods ...func(*vedro.Bucket)) vedro.Bucket {
		return NewBucketCR("my-bucket", cfg.Location, mods...)
	}
	newBucketAttrs := func(
		name string,
		location string,
		storageClass vedro.BucketStorageClass,
		mods ...func(*vedro.BucketProperties),
	) *cloud.BucketAttrs {
		allMods := append([]func(*vedro.BucketProperties){}, cfg.BucketPropertiesMods...)
		allMods = append(allMods, mods...)
		return NewBucketAttrs(name, location, storageClass, allMods...)
	}
	lifecycleCapability := cfg.BucketCaps.Lifecycle.RuleExpiration

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

			bckt := newBucketCR(func(b *vedro.Bucket) {
				b.Spec.StorageClass = vedro.BucketStorageClassStandard
				if cfg.BucketCaps.Labels {
					b.Spec.Labels = map[string]string{"env": "prod"}
				}
				if cfg.BucketCaps.PublicAccessPrevention {
					b.Spec.PublicAccessPrevention = helpers.Ptr(true)
				}
				if cfg.BucketCaps.Versioning {
					b.Spec.Versioning = &vedro.BucketVersioning{Enabled: true}
				}
				if lifecycleCapability {
					b.Spec.Lifecycle = &Lifecycle
				}
			})

			attrs, err := bucket.EnsureBucket(ctx, bckt)
			Expect(err).NotTo(HaveOccurred())

			Expect(fake.Created).NotTo(BeNil())
			Expect(fake.Created.Location).To(Equal(cfg.Location))
			Expect(fake.Created.Properties.StorageClass).To(Equal(vedro.BucketStorageClassStandard))
			Expect(attrs.Properties.StorageClass).To(Equal(vedro.BucketStorageClassStandard))

			if cfg.BucketCaps.Labels {
				Expect(fake.Created.Properties.Labels).To(Equal(map[string]string{"env": "prod"}))
			}
			if cfg.BucketCaps.Versioning {
				Expect(fake.Created.Properties.Versioning.Enabled).To(BeTrue())
			}
			if lifecycleCapability {
				Expect(fake.Created.Properties.Lifecycle).To(Equal(&Lifecycle))
			}
			if cfg.BucketCaps.PublicAccessPrevention {
				Expect(*fake.Created.Properties.PublicAccessPrevention).To(BeTrue())
			}
			if cfg.BucketCaps.Labels {
				Expect(attrs.Properties.Labels).To(Equal(map[string]string{"env": "prod"}))
			}
			if cfg.BucketCaps.Versioning {
				Expect(attrs.Properties.Versioning.Enabled).To(BeTrue())
			}
			if cfg.BucketCaps.PublicAccessPrevention {
				Expect(*attrs.Properties.PublicAccessPrevention).To(BeTrue())
			}
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
			fake.Attrs = newBucketAttrs(
				"my-bucket", cfg.NormalizedLocation, vedro.BucketStorageClassStandard,
			)
			fake.Attrs.Id = "external-bucket-id"

			bckt := newBucketCR(func(b *vedro.Bucket) {
				b.Spec.StorageClass = vedro.BucketStorageClassStandard
			})

			attrs, err := bucket.EnsureBucket(ctx, bckt)
			Expect(err).NotTo(HaveOccurred())
			Expect(fake.Updated).To(BeNil())
			Expect(attrs.Name).To(Equal("my-bucket"))
			Expect(attrs.Id).To(Equal("external-bucket-id"))
			Expect(attrs.Location).To(Equal(cfg.NormalizedLocation))
		})

		It("returns an error when the existing bucket is in a different location", func() {
			fake.Attrs = newBucketAttrs(
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
			fake.Attrs = newBucketAttrs(
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

		if lifecycleCapability {
			It("updates lifecycle when its empty", func() {
				fake.Attrs = newBucketAttrs(
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
				fake.Attrs = newBucketAttrs(
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
		}

		if cfg.BucketCaps.Versioning {

			It("updates versioning when it differs", func() {
				fake.Attrs = newBucketAttrs(
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
		}

		if cfg.BucketCaps.Labels {

			It("updates labels when they differ", func() {
				fake.Attrs = newBucketAttrs(
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

			It("updates labels when spec.Labels is nil labels in status.Applied.Labels", func() {
				fake.Attrs = newBucketAttrs(
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
		}

		if cfg.BucketCaps.PublicAccessPrevention {
			It("updates public access prevention when it differs", func() {
				fake.Attrs = newBucketAttrs(
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
		}

		It("returns an error when updating the bucket fails", func() {
			fake.Attrs = newBucketAttrs(
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
			Expect(fake.ProcessObjectsCalled).To(BeFalse())
		})

		It("deletes an empty bucket", func() {
			bckt := newDeleteBucketCR()

			err := bucket.DeleteBucket(ctx, bckt)
			Expect(err).NotTo(HaveOccurred())
			Expect(fake.Deleted).To(BeTrue())
		})

		It("processes objects before deleting the bucket", func() {
			fake.ObjectIterator = &FakeObjectIterator{}
			bckt := newDeleteBucketCR()

			err := bucket.DeleteBucket(ctx, bckt)
			Expect(err).NotTo(HaveOccurred())
			Expect(fake.ProcessObjectsCalled).To(BeTrue())
		})

		It("deletes all objects before deleting the bucket", func() {
			fake.ObjectIterator = &FakeObjectIterator{
				Objects: []cloud.ObjectVersion{
					{Name: "obj-a", Version: helpers.Ptr("1")},
					{Name: "obj-b", Version: helpers.Ptr("2")},
				},
			}
			bckt := newDeleteBucketCR()

			err := bucket.DeleteBucket(ctx, bckt)
			Expect(err).NotTo(HaveOccurred())
			Expect(fake.Deleted).To(BeTrue())
			Expect(fake.GetDeletedObjects()).To(ConsistOf(
				DeletedObject{Name: "obj-a", Version: helpers.Ptr("1")},
				DeletedObject{Name: "obj-b", Version: helpers.Ptr("2")},
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
				Objects: []cloud.ObjectVersion{
					{Name: "obj-a", Version: helpers.Ptr("1")},
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
				Objects: []cloud.ObjectVersion{
					{Name: "obj-a", Version: helpers.Ptr("1")},
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

			result := bucket.ValidateBucketSpec(bckt, cfg.ProviderConfigType)
			Expect(result.Valid).To(BeTrue())
		})

		It("returns valid when spec.name is used", func() {
			bckt := newBucketCR(func(b *vedro.Bucket) {
				b.Spec.Name = "actual-bucket-name"
			})

			result := bucket.ValidateBucketSpec(bckt, cfg.ProviderConfigType)
			Expect(result.Valid).To(BeTrue())
		})

		It("returns an error when spec.name is changed after creation", func() {
			bckt := newBucketCR(func(b *vedro.Bucket) {
				b.Spec.Name = "new-name"
				b.Status.ExternalName = "old-name"
			})

			result := bucket.ValidateBucketSpec(bckt, cfg.ProviderConfigType)
			Expect(result.Valid).To(BeFalse())
			Expect(result.Message).To(ContainSubstring("spec.name cannot be changed"))
		})

		It("returns an error when metadata.name is used after spec.name was used", func() {
			bckt := newBucketCR(func(b *vedro.Bucket) {
				b.Spec.Name = ""
				b.Status.ExternalName = "old-spec-name"
			})

			result := bucket.ValidateBucketSpec(bckt, cfg.ProviderConfigType)
			Expect(result.Valid).To(BeFalse())
			Expect(result.Message).To(ContainSubstring("metadata.name cannot be used"))
		})

		It("returns an error when location is empty", func() {
			bckt := newBucketCR(func(b *vedro.Bucket) {
				b.Spec.Location = ""
			})

			result := bucket.ValidateBucketSpec(bckt, cfg.ProviderConfigType)
			Expect(result.Valid).To(BeFalse())
			Expect(result.Message).To(ContainSubstring("location is an empty string"))
		})

		It("returns an error for an unsupported location", func() {
			bckt := newBucketCR(func(b *vedro.Bucket) {
				b.Spec.Location = "bad"
			})

			result := bucket.ValidateBucketSpec(bckt, cfg.ProviderConfigType)
			Expect(result.Valid).To(BeFalse())
			Expect(result.Message).To(ContainSubstring("unsupported bucket location"))
		})

		It("returns an error when the bucket name is too short", func() {
			bckt := newBucketCR(func(b *vedro.Bucket) {
				b.Name = "b"
			})

			result := bucket.ValidateBucketSpec(bckt, cfg.ProviderConfigType)
			Expect(result.Valid).To(BeFalse())
			Expect(result.Message).To(ContainSubstring("bucket name must be 3-63 characters"))
		})

		It("returns an error when the bucket name contains uppercase letters", func() {
			bckt := newBucketCR(func(b *vedro.Bucket) {
				b.Name = "My-Bucket"
			})

			result := bucket.ValidateBucketSpec(bckt, cfg.ProviderConfigType)
			Expect(result.Valid).To(BeFalse())
			Expect(result.Message).To(ContainSubstring("bucket name must be 3-63 characters"))
		})

		It("returns an error when the bucket name contains consecutive dots", func() {
			bckt := newBucketCR(func(b *vedro.Bucket) {
				b.Name = "my..bucket"
			})

			result := bucket.ValidateBucketSpec(bckt, cfg.ProviderConfigType)
			Expect(result.Valid).To(BeFalse())
			Expect(result.Message).To(ContainSubstring("consecutive dots"))
		})

		It("returns an error when the bucket name has dots next to dashes", func() {
			bckt := newBucketCR(func(b *vedro.Bucket) {
				b.Name = "my.-bucket"
			})

			result := bucket.ValidateBucketSpec(bckt, cfg.ProviderConfigType)
			Expect(result.Valid).To(BeFalse())
			Expect(result.Message).To(ContainSubstring("dots next to dashes"))
		})
		It("returns an error when spec.name is invalid", func() {
			bckt := newBucketCR(func(b *vedro.Bucket) {
				b.Spec.Name = "INVALID"
			})

			result := bucket.ValidateBucketSpec(bckt, cfg.ProviderConfigType)
			Expect(result.Valid).To(BeFalse())
			Expect(result.Message).To(ContainSubstring("bucket name must be 3-63 characters"))
		})
	})

	return true
}
