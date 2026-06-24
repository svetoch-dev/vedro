package gcp

import (
	"context"
	"errors"
	"sync"

	"cloud.google.com/go/storage"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"google.golang.org/api/iterator"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	vedrov1alpha1 "github.com/svetoch-dev/vedro/api/v1alpha1"
	"github.com/svetoch-dev/vedro/internal/cloud"
	"github.com/svetoch-dev/vedro/internal/helpers"
)

func newTestBucket(name, location string, mods ...func(*vedrov1alpha1.Bucket)) vedrov1alpha1.Bucket {
	b := vedrov1alpha1.Bucket{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Spec: vedrov1alpha1.BucketSpec{
			ProviderRef: vedrov1alpha1.ProviderConfigReference{Name: "gcp-dev"},
			Location:    location,
		},
	}
	for _, m := range mods {
		m(&b)
	}
	return b
}

func newBucketAttrs(
	name, location string,
	storageClass vedrov1alpha1.BucketStorageClass,
	mods ...func(*vedrov1alpha1.BucketProperties),
) *cloud.BucketAttrs {
	properties := &vedrov1alpha1.BucketProperties{StorageClass: storageClass}
	for _, mod := range mods {
		mod(properties)
	}
	return &cloud.BucketAttrs{Name: name, Location: location, Properties: properties}
}

var lifecycle = vedrov1alpha1.BucketLifecycle{
	Rules: []vedrov1alpha1.BucketLifecycleRule{
		vedrov1alpha1.BucketLifecycleRule{
			Enabled: true,
			AgeDays: helpers.Ptr(int64(2)),
			Action:  vedrov1alpha1.BucketLifecycleActionDelete,
		},
	},
}

var _ = Describe("Bucket.EnsureBucket", func() {
	var (
		ctx    context.Context
		fake   *fakeBucketHandle
		bucket *Bucket
	)

	BeforeEach(func() {
		ctx = context.Background()
		fake = &fakeBucketHandle{}
		bucket = &Bucket{
			api: fake,
		}
	})

	It("creates a bucket when it does not exist", func() {
		fake.attrsErr = cloud.ErrBucketNotFound

		bckt := newTestBucket("my-bucket", "europe-west1", func(b *vedrov1alpha1.Bucket) {
			b.Spec.StorageClass = vedrov1alpha1.BucketStorageClassStandard
		})

		attrs, err := bucket.EnsureBucket(ctx, bckt)
		Expect(err).NotTo(HaveOccurred())
		Expect(fake.created).NotTo(BeNil())
		Expect(fake.created.Location).To(Equal("europe-west1"))
		Expect(fake.created.Properties.StorageClass).To(Equal(vedrov1alpha1.BucketStorageClassStandard))
		Expect(attrs.Name).To(Equal("my-bucket"))
		Expect(attrs.Location).To(Equal("europe-west1"))
		Expect(attrs.Properties).NotTo(BeNil())
		Expect(attrs.Properties.StorageClass).To(Equal(vedrov1alpha1.BucketStorageClassStandard))
	})

	It("creates a bucket with all supported options", func() {
		fake.attrsErr = cloud.ErrBucketNotFound

		publicAccessPrevention := true
		bckt := newTestBucket("my-bucket", "us-central1", func(b *vedrov1alpha1.Bucket) {
			b.Spec.StorageClass = vedrov1alpha1.BucketStorageClassArchive
			b.Spec.Labels = map[string]string{"env": "prod"}
			b.Spec.Versioning = &vedrov1alpha1.BucketVersioning{Enabled: true}
			b.Spec.PublicAccessPrevention = &publicAccessPrevention
			b.Spec.Lifecycle = &lifecycle
		})

		attrs, err := bucket.EnsureBucket(ctx, bckt)
		Expect(err).NotTo(HaveOccurred())

		Expect(fake.created).NotTo(BeNil())
		Expect(fake.created.Location).To(Equal("us-central1"))
		Expect(fake.created.Properties.StorageClass).To(Equal(vedrov1alpha1.BucketStorageClassArchive))
		Expect(fake.created.Properties.Labels).To(Equal(map[string]string{"env": "prod"}))
		Expect(fake.created.Properties.Versioning.Enabled).To(BeTrue())
		Expect(fake.created.Properties.Lifecycle).To(Equal(&lifecycle))
		Expect(*fake.created.Properties.PublicAccessPrevention).To(BeTrue())
		Expect(attrs.Properties.StorageClass).To(Equal(vedrov1alpha1.BucketStorageClassArchive))
		Expect(attrs.Properties.Labels).To(Equal(map[string]string{"env": "prod"}))
		Expect(attrs.Properties.Versioning.Enabled).To(BeTrue())
		Expect(*attrs.Properties.PublicAccessPrevention).To(BeTrue())
	})

	It("returns an error when creating a bucket fails", func() {
		fake.attrsErr = cloud.ErrBucketNotFound
		fake.createErr = errors.New("network error")

		bckt := newTestBucket("my-bucket", "europe-west1", func(b *vedrov1alpha1.Bucket) {
			b.Spec.StorageClass = vedrov1alpha1.BucketStorageClassStandard
		})

		_, err := bucket.EnsureBucket(ctx, bckt)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("create bucket \"my-bucket\""))
		Expect(err.Error()).To(ContainSubstring("network error"))
	})

	It("returns an error when fetching bucket attributes fails", func() {
		fake.attrsErr = errors.New("permission denied")

		bckt := newTestBucket("my-bucket", "europe-west1", func(b *vedrov1alpha1.Bucket) {
			b.Spec.StorageClass = vedrov1alpha1.BucketStorageClassStandard
		})

		_, err := bucket.EnsureBucket(ctx, bckt)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("get bucket attrs \"my-bucket\""))
		Expect(err.Error()).To(ContainSubstring("permission denied"))
	})

	It("returns an error for an unmapped storage class", func() {
		fake.attrsErr = cloud.ErrBucketNotFound
		fake.createErr = errors.New("storage class Glacier is not supported")

		bckt := newTestBucket("my-bucket", "europe-west1", func(b *vedrov1alpha1.Bucket) {
			b.Spec.StorageClass = vedrov1alpha1.BucketStorageClass("Glacier")
		})

		_, err := bucket.EnsureBucket(ctx, bckt)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("Glacier"))
	})

	It("returns the existing attrs when the bucket already matches the spec", func() {
		fake.attrs = newBucketAttrs(
			"my-bucket", "EUROPE-WEST1", vedrov1alpha1.BucketStorageClassStandard,
		)

		bckt := newTestBucket("my-bucket", "europe-west1", func(b *vedrov1alpha1.Bucket) {
			b.Spec.StorageClass = vedrov1alpha1.BucketStorageClassStandard
		})

		attrs, err := bucket.EnsureBucket(ctx, bckt)
		Expect(err).NotTo(HaveOccurred())
		Expect(fake.updated).To(BeNil())
		Expect(attrs.Name).To(Equal("my-bucket"))
		Expect(attrs.Location).To(Equal("EUROPE-WEST1"))
	})

	It("returns an error when the existing bucket is in a different location", func() {
		fake.attrs = newBucketAttrs(
			"my-bucket", "US-CENTRAL1", vedrov1alpha1.BucketStorageClassStandard,
		)

		bckt := newTestBucket("my-bucket", "europe-west1", func(b *vedrov1alpha1.Bucket) {
			b.Spec.StorageClass = vedrov1alpha1.BucketStorageClassStandard
		})

		_, err := bucket.EnsureBucket(ctx, bckt)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("already exists in location \"US-CENTRAL1\""))
		Expect(err.Error()).To(ContainSubstring("desired location is \"EUROPE-WEST1\""))
	})

	It("updates the storage class when it differs", func() {
		fake.attrs = newBucketAttrs(
			"my-bucket", "EUROPE-WEST1", vedrov1alpha1.BucketStorageClassStandard,
		)

		bckt := newTestBucket("my-bucket", "europe-west1", func(b *vedrov1alpha1.Bucket) {
			b.Spec.StorageClass = vedrov1alpha1.BucketStorageClassInfrequentAccess
		})

		attrs, err := bucket.EnsureBucket(ctx, bckt)
		Expect(err).NotTo(HaveOccurred())
		Expect(fake.updated).NotTo(BeNil())
		Expect(fake.updated.StorageClass).To(Equal(
			helpers.PatchTo(vedrov1alpha1.BucketStorageClassInfrequentAccess),
		))
		Expect(attrs.Properties.StorageClass).To(Equal(vedrov1alpha1.BucketStorageClassInfrequentAccess))
	})
	It("updates lifecycle when its empty", func() {
		fake.attrs = newBucketAttrs(
			"my-bucket", "EUROPE-WEST1", vedrov1alpha1.BucketStorageClassStandard,
		)

		bckt := newTestBucket("my-bucket", "europe-west1", func(b *vedrov1alpha1.Bucket) {
			b.Spec.StorageClass = vedrov1alpha1.BucketStorageClassStandard
			b.Spec.Lifecycle = &lifecycle
		})

		attrs, err := bucket.EnsureBucket(ctx, bckt)
		Expect(err).NotTo(HaveOccurred())
		Expect(fake.updated).NotTo(BeNil())
		Expect(fake.updated.Lifecycle).To(Equal(helpers.PatchTo(&lifecycle)))
		Expect(attrs.Properties.Lifecycle).To(Equal(&lifecycle))
	})
	It("updates lifecycle when it differs", func() {
		actualLifecycle := lifecycle.DeepCopy()
		actualLifecycle.Rules[0].AgeDays = helpers.Ptr(int64(100000))
		fake.attrs = newBucketAttrs(
			"my-bucket", "EUROPE-WEST1", vedrov1alpha1.BucketStorageClassStandard,
			func(p *vedrov1alpha1.BucketProperties) { p.Lifecycle = actualLifecycle },
		)

		bckt := newTestBucket("my-bucket", "europe-west1", func(b *vedrov1alpha1.Bucket) {
			b.Spec.StorageClass = vedrov1alpha1.BucketStorageClassStandard
			b.Spec.Lifecycle = &lifecycle
		})

		attrs, err := bucket.EnsureBucket(ctx, bckt)
		Expect(err).NotTo(HaveOccurred())
		Expect(fake.updated).NotTo(BeNil())
		Expect(fake.updated.Lifecycle).To(Equal(helpers.PatchTo(&lifecycle)))
		Expect(attrs.Properties.Lifecycle).To(Equal(&lifecycle))
	})

	It("updates versioning when it differs", func() {
		fake.attrs = newBucketAttrs(
			"my-bucket", "EUROPE-WEST1", vedrov1alpha1.BucketStorageClassStandard,
			func(p *vedrov1alpha1.BucketProperties) {
				p.Versioning = &vedrov1alpha1.BucketVersioning{Enabled: false}
			},
		)

		bckt := newTestBucket("my-bucket", "europe-west1", func(b *vedrov1alpha1.Bucket) {
			b.Spec.StorageClass = vedrov1alpha1.BucketStorageClassStandard
			b.Spec.Versioning = &vedrov1alpha1.BucketVersioning{Enabled: true}
		})

		attrs, err := bucket.EnsureBucket(ctx, bckt)
		Expect(err).NotTo(HaveOccurred())
		Expect(fake.updated).NotTo(BeNil())
		Expect(fake.updated.Versioning).To(Equal(
			helpers.PatchTo(&vedrov1alpha1.BucketVersioning{Enabled: true}),
		))
		Expect(attrs.Properties.Versioning.Enabled).To(BeTrue())
	})

	It("updates labels when they differ", func() {
		fake.attrs = newBucketAttrs(
			"my-bucket", "EUROPE-WEST1", vedrov1alpha1.BucketStorageClassStandard,
			func(p *vedrov1alpha1.BucketProperties) {
				p.Labels = map[string]string{"env": "dev"}
			},
		)

		bckt := newTestBucket("my-bucket", "europe-west1", func(b *vedrov1alpha1.Bucket) {
			b.Spec.StorageClass = vedrov1alpha1.BucketStorageClassStandard
			b.Spec.Labels = map[string]string{"env": "prod"}
		})

		attrs, err := bucket.EnsureBucket(ctx, bckt)
		Expect(err).NotTo(HaveOccurred())
		Expect(fake.updated).NotTo(BeNil())
		Expect(fake.updated.Labels).To(Equal(helpers.PatchTo(map[string]string{"env": "prod"})))
		Expect(attrs.Properties.Labels).To(Equal(map[string]string{"env": "prod"}))
	})

	It("updates public access prevention when it differs", func() {
		fake.attrs = newBucketAttrs(
			"my-bucket", "EUROPE-WEST1", vedrov1alpha1.BucketStorageClassStandard,
			func(p *vedrov1alpha1.BucketProperties) {
				p.PublicAccessPrevention = helpers.Ptr(false)
			},
		)

		publicAccessPrevention := true
		bckt := newTestBucket("my-bucket", "europe-west1", func(b *vedrov1alpha1.Bucket) {
			b.Spec.StorageClass = vedrov1alpha1.BucketStorageClassStandard
			b.Spec.PublicAccessPrevention = &publicAccessPrevention
		})

		attrs, err := bucket.EnsureBucket(ctx, bckt)
		Expect(err).NotTo(HaveOccurred())
		Expect(fake.updated).NotTo(BeNil())
		Expect(fake.updated.PublicAccessPrevention).To(Equal(helpers.PatchTo(&publicAccessPrevention)))
		Expect(*attrs.Properties.PublicAccessPrevention).To(BeTrue())
	})

	It("updates labels when spec.Labels is nil labels in status.Applied.Labels", func() {
		fake.attrs = newBucketAttrs(
			"my-bucket", "EUROPE-WEST1", vedrov1alpha1.BucketStorageClassStandard,
			func(p *vedrov1alpha1.BucketProperties) {
				p.Labels = map[string]string{"env": "dev"}
			},
		)

		bckt := newTestBucket("my-bucket", "europe-west1", func(b *vedrov1alpha1.Bucket) {
			b.Spec.StorageClass = vedrov1alpha1.BucketStorageClassStandard
			b.Status.Applied = &vedrov1alpha1.BucketProperties{
				StorageClass: vedrov1alpha1.BucketStorageClassStandard,
				Labels:       map[string]string{"env": "dev"},
			}
		})

		attrs, err := bucket.EnsureBucket(ctx, bckt)
		Expect(err).NotTo(HaveOccurred())
		Expect(fake.updated).NotTo(BeNil())
		Expect(fake.updated.Labels.Set).To(BeTrue())
		Expect(fake.updated.Labels.Value).To(BeNil())
		Expect(attrs.Properties.Labels).To(BeEmpty())
	})

	It("returns an error when updating the bucket fails", func() {
		fake.attrs = newBucketAttrs(
			"my-bucket", "EUROPE-WEST1", vedrov1alpha1.BucketStorageClassStandard,
		)
		fake.updateErr = errors.New("update failed")

		bckt := newTestBucket("my-bucket", "europe-west1", func(b *vedrov1alpha1.Bucket) {
			b.Spec.StorageClass = vedrov1alpha1.BucketStorageClassInfrequentAccess
		})

		_, err := bucket.EnsureBucket(ctx, bckt)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("update bucket \"my-bucket\""))
		Expect(err.Error()).To(ContainSubstring("update failed"))
	})
})

var _ = Describe("Bucket.DeleteBucket", func() {
	var (
		ctx    context.Context
		fake   *fakeBucketHandle
		bucket *Bucket
	)

	BeforeEach(func() {
		ctx = context.Background()
		fake = &fakeBucketHandle{}
		bucket = &Bucket{
			api: fake,
		}
	})

	newDeleteBucket := func(mods ...func(*vedrov1alpha1.Bucket)) vedrov1alpha1.Bucket {
		return newTestBucket("my-bucket", "europe-west1", append([]func(*vedrov1alpha1.Bucket){
			func(b *vedrov1alpha1.Bucket) {
				b.Spec.DeletionPolicy = vedrov1alpha1.DeletionPolicyDelete
			},
		}, mods...)...)
	}

	It("does nothing when the deletion policy is not Delete", func() {
		bckt := newTestBucket("my-bucket", "europe-west1", func(b *vedrov1alpha1.Bucket) {
			b.Spec.DeletionPolicy = vedrov1alpha1.DeletionPolicyRetain
		})

		err := bucket.DeleteBucket(ctx, bckt)
		Expect(err).NotTo(HaveOccurred())
		Expect(fake.deleted).To(BeFalse())
		Expect(fake.query).To(BeNil())
	})

	It("deletes an empty bucket", func() {
		bckt := newDeleteBucket()

		err := bucket.DeleteBucket(ctx, bckt)
		Expect(err).NotTo(HaveOccurred())
		Expect(fake.deleted).To(BeTrue())
	})

	It("requests all object versions while listing", func() {
		fake.objectIterator = &fakeObjectIterator{}
		bckt := newDeleteBucket()

		err := bucket.DeleteBucket(ctx, bckt)
		Expect(err).NotTo(HaveOccurred())
		Expect(fake.query).NotTo(BeNil())
		Expect(fake.query.Versions).To(BeTrue())
	})

	It("deletes all objects before deleting the bucket", func() {
		fake.objectIterator = &fakeObjectIterator{
			attrs: []*storage.ObjectAttrs{
				{Name: "obj-a", Generation: 1},
				{Name: "obj-b", Generation: 2},
			},
		}
		bckt := newDeleteBucket()

		err := bucket.DeleteBucket(ctx, bckt)
		Expect(err).NotTo(HaveOccurred())
		Expect(fake.deleted).To(BeTrue())
		Expect(fake.getDeletedObjects()).To(ConsistOf(
			deletedObject{name: "obj-a", generation: 1},
			deletedObject{name: "obj-b", generation: 2},
		))
	})

	It("returns an error when listing objects fails", func() {
		fake.objectIterator = &fakeObjectIterator{err: errors.New("list failed")}
		bckt := newDeleteBucket()

		err := bucket.DeleteBucket(ctx, bckt)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("list failed"))
		Expect(fake.deleted).To(BeFalse())
	})

	It("returns an error when object deletion fails", func() {
		fake.objectIterator = &fakeObjectIterator{
			attrs: []*storage.ObjectAttrs{
				{Name: "obj-a", Generation: 1},
			},
		}
		fake.objectDeleteErr = errors.New("object delete failed")
		bckt := newDeleteBucket()

		err := bucket.DeleteBucket(ctx, bckt)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("could not delete bucket because object deletion failed"))
		Expect(err.Error()).To(ContainSubstring("object delete failed"))
		Expect(fake.deleted).To(BeFalse())
	})

	It("ignores 404 errors while deleting objects", func() {
		fake.objectIterator = &fakeObjectIterator{
			attrs: []*storage.ObjectAttrs{
				{Name: "obj-a", Generation: 1},
			},
		}
		fake.objectDeleteErr = cloud.ErrBucketObjectNotFound
		bckt := newDeleteBucket()

		err := bucket.DeleteBucket(ctx, bckt)
		Expect(err).NotTo(HaveOccurred())
		Expect(fake.deleted).To(BeTrue())
	})

	It("ignores 404 errors when deleting the bucket", func() {
		fake.deleteErr = cloud.ErrBucketNotFound
		bckt := newDeleteBucket()

		err := bucket.DeleteBucket(ctx, bckt)
		Expect(err).NotTo(HaveOccurred())
		Expect(fake.deleted).To(BeTrue())
	})

	It("returns an error when bucket deletion fails", func() {
		fake.deleteErr = errors.New("bucket delete failed")
		bckt := newDeleteBucket()

		err := bucket.DeleteBucket(ctx, bckt)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("bucket delete failed"))
		Expect(fake.deleted).To(BeTrue())
	})
})

var _ = Describe("Bucket.ValidateBucketSpec", func() {
	var bucket *Bucket

	BeforeEach(func() {
		bucket = &Bucket{}
	})

	It("returns valid for a correct bucket spec", func() {
		bckt := newTestBucket("my-bucket", "europe-west-1")

		result := bucket.ValidateBucketSpec(bckt)
		Expect(result.Valid).To(BeTrue())
	})

	It("returns valid when spec.name is used", func() {
		bckt := newTestBucket("cr-name", "us-central1", func(b *vedrov1alpha1.Bucket) {
			b.Spec.Name = "actual-bucket-name"
		})

		result := bucket.ValidateBucketSpec(bckt)
		Expect(result.Valid).To(BeTrue())
	})

	It("returns an error when spec.name is changed after creation", func() {
		bckt := newTestBucket("cr-name", "us-central1", func(b *vedrov1alpha1.Bucket) {
			b.Spec.Name = "new-name"
			b.Status.ExternalName = "old-name"
		})

		result := bucket.ValidateBucketSpec(bckt)
		Expect(result.Valid).To(BeFalse())
		Expect(result.Message).To(ContainSubstring("spec.name cannot be changed"))
	})

	It("returns an error when metadata.name is used after spec.name was used", func() {
		bckt := newTestBucket("cr-name", "us-central1", func(b *vedrov1alpha1.Bucket) {
			b.Spec.Name = ""
			b.Status.ExternalName = "old-spec-name"
		})

		result := bucket.ValidateBucketSpec(bckt)
		Expect(result.Valid).To(BeFalse())
		Expect(result.Message).To(ContainSubstring("metadata.name cannot be used"))
	})

	It("returns an error when location is empty", func() {
		bckt := newTestBucket("my-bucket", "")

		result := bucket.ValidateBucketSpec(bckt)
		Expect(result.Valid).To(BeFalse())
		Expect(result.Message).To(ContainSubstring("location is an empty string"))
	})

	It("returns an error for an unsupported location", func() {
		bckt := newTestBucket("my-bucket", "mars")

		result := bucket.ValidateBucketSpec(bckt)
		Expect(result.Valid).To(BeFalse())
		Expect(result.Message).To(ContainSubstring("unsupported bucket location"))
	})

	It("accepts multi-region locations", func() {
		bckt := newTestBucket("my-bucket", "us")

		result := bucket.ValidateBucketSpec(bckt)
		Expect(result.Valid).To(BeTrue())
	})

	It("accepts dual-region locations", func() {
		bckt := newTestBucket("my-bucket", "NAM4")

		result := bucket.ValidateBucketSpec(bckt)
		Expect(result.Valid).To(BeTrue())
	})

	It("returns an error when the bucket name is too short", func() {
		bckt := newTestBucket("ab", "europe-west1")

		result := bucket.ValidateBucketSpec(bckt)
		Expect(result.Valid).To(BeFalse())
		Expect(result.Message).To(ContainSubstring("bucket name must be 3-63 characters"))
	})

	It("returns an error when the bucket name contains uppercase letters", func() {
		bckt := newTestBucket("My-Bucket", "europe-west1")

		result := bucket.ValidateBucketSpec(bckt)
		Expect(result.Valid).To(BeFalse())
		Expect(result.Message).To(ContainSubstring("bucket name must be 3-63 characters"))
	})

	It("returns an error when the bucket name contains consecutive dots", func() {
		bckt := newTestBucket("my..bucket", "europe-west1")

		result := bucket.ValidateBucketSpec(bckt)
		Expect(result.Valid).To(BeFalse())
		Expect(result.Message).To(ContainSubstring("consecutive dots"))
	})

	It("returns an error when the bucket name has dots next to dashes", func() {
		bckt := newTestBucket("my.-bucket", "europe-west1")

		result := bucket.ValidateBucketSpec(bckt)
		Expect(result.Valid).To(BeFalse())
		Expect(result.Message).To(ContainSubstring("dots next to dashes"))
	})

	It("returns an error when the bucket name uses a reserved google prefix", func() {
		bckt := newTestBucket("google-bucket", "europe-west1")

		result := bucket.ValidateBucketSpec(bckt)
		Expect(result.Valid).To(BeFalse())
		Expect(result.Message).To(ContainSubstring("reserved Google-related names"))
	})

	It("returns an error when spec.name is invalid", func() {
		bckt := newTestBucket("cr-name", "europe-west1", func(b *vedrov1alpha1.Bucket) {
			b.Spec.Name = "INVALID"
		})

		result := bucket.ValidateBucketSpec(bckt)
		Expect(result.Valid).To(BeFalse())
		Expect(result.Message).To(ContainSubstring("bucket name must be 3-63 characters"))
	})
})

type fakeBucketHandle struct {
	attrs     *cloud.BucketAttrs
	attrsErr  error
	createErr error
	updateErr error
	deleteErr error
	created   *cloud.BucketAttrs
	updated   *cloud.BucketPatch

	deleted          bool
	query            *storage.Query
	objectIterator   *fakeObjectIterator
	objectDeleteErr  error
	deletedObjectsMu sync.Mutex
	deletedObjects   []deletedObject
}

type deletedObject struct {
	name       string
	generation int64
}

func (f *fakeBucketHandle) DeleteBucket(ctx context.Context, _ string) error {
	f.deleted = true
	return f.deleteErr
}

func (f *fakeBucketHandle) ProcessObjects(
	ctx context.Context,
	_ string,
	process func(cloud.ObjectVersion) error,
) error {
	f.query = &storage.Query{Versions: true}
	if f.objectIterator != nil {
		for {
			attrs, err := f.objectIterator.Next()
			if errors.Is(err, iterator.Done) {
				return nil
			}
			if err != nil {
				return err
			}
			if err := process(cloud.ObjectVersion{Name: attrs.Name, Version: attrs.Generation}); err != nil {
				return err
			}
		}
	}
	return nil
}

func (f *fakeBucketHandle) DeleteObject(
	ctx context.Context,
	_ string,
	object cloud.ObjectVersion,
) error {
	f.recordDeletedObject(object.Name, object.Version)
	return f.objectDeleteErr
}

func (f *fakeBucketHandle) recordDeletedObject(name string, generation int64) {
	f.deletedObjectsMu.Lock()
	defer f.deletedObjectsMu.Unlock()
	f.deletedObjects = append(f.deletedObjects, deletedObject{name: name, generation: generation})
}

func (f *fakeBucketHandle) getDeletedObjects() []deletedObject {
	f.deletedObjectsMu.Lock()
	defer f.deletedObjectsMu.Unlock()
	out := make([]deletedObject, len(f.deletedObjects))
	copy(out, f.deletedObjects)
	return out
}

type fakeObjectIterator struct {
	attrs []*storage.ObjectAttrs
	index int
	err   error
}

func (f *fakeObjectIterator) Next() (*storage.ObjectAttrs, error) {
	if f.err != nil {
		return nil, f.err
	}
	if f.index >= len(f.attrs) {
		return nil, iterator.Done
	}
	attrs := f.attrs[f.index]
	f.index++
	return attrs, nil
}

func (f *fakeBucketHandle) GetBucket(ctx context.Context, _ string) (*cloud.BucketAttrs, error) {
	if f.attrsErr != nil {
		return nil, f.attrsErr
	}
	return f.attrs, nil
}

func (f *fakeBucketHandle) CreateBucket(
	ctx context.Context,
	_ string,
	attrs cloud.BucketAttrs,
) error {
	f.created = &attrs
	if f.createErr != nil {
		return f.createErr
	}
	f.attrs = &attrs
	return nil
}

func (f *fakeBucketHandle) UpdateBucket(
	ctx context.Context,
	_ string,
	patch cloud.BucketPatch,
) (*cloud.BucketAttrs, error) {
	f.updated = &patch
	if f.updateErr != nil {
		return nil, f.updateErr
	}

	if f.attrs.Properties == nil {
		f.attrs.Properties = &vedrov1alpha1.BucketProperties{}
	}
	if patch.StorageClass.Set {
		f.attrs.Properties.StorageClass = patch.StorageClass.Value
	}
	if patch.Labels.Set {
		f.attrs.Properties.Labels = patch.Labels.Value
	}
	if patch.Versioning.Set {
		f.attrs.Properties.Versioning = patch.Versioning.Value
	}
	if patch.PublicAccessPrevention.Set {
		f.attrs.Properties.PublicAccessPrevention = patch.PublicAccessPrevention.Value
	}
	if patch.Lifecycle.Set {
		f.attrs.Properties.Lifecycle = patch.Lifecycle.Value
	}

	return f.attrs, nil
}
