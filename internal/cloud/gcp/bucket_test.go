package gcp

import (
	"context"
	"errors"

	"cloud.google.com/go/storage"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	vedrov1alpha1 "github.com/svetoch-dev/vedro/api/v1alpha1"
)

var _ = Describe("Bucket.EnsureBucket", func() {
	var (
		ctx       context.Context
		fake      *fakeBucketHandle
		bucket    *Bucket
		projectID = "test-project"
	)

	BeforeEach(func() {
		ctx = context.Background()
		fake = &fakeBucketHandle{}
		bucket = &Bucket{
			client:    &fakeStorageClient{bucket: fake},
			projectId: projectID,
		}
	})

	newBucket := func(name, location string, mods ...func(*vedrov1alpha1.Bucket)) vedrov1alpha1.Bucket {
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

	It("creates a bucket when it does not exist", func() {
		fake.attrsErr = storage.ErrBucketNotExist

		bckt := newBucket("my-bucket", "europe-west1", func(b *vedrov1alpha1.Bucket) {
			b.Spec.StorageClass = vedrov1alpha1.BucketStorageClassStandard
		})

		state, err := bucket.EnsureBucket(ctx, bckt)
		Expect(err).NotTo(HaveOccurred())
		Expect(fake.created).NotTo(BeNil())
		Expect(fake.created.Location).To(Equal("europe-west1"))
		Expect(fake.created.StorageClass).To(Equal("STANDARD"))
		Expect(state.ExternalName).To(Equal("my-bucket"))
		Expect(state.Location).To(Equal("EUROPE-WEST1"))
		Expect(state.Applied).NotTo(BeNil())
		Expect(state.Applied.StorageClass).To(Equal(vedrov1alpha1.BucketStorageClassStandard))
	})

	It("creates a bucket with all supported options", func() {
		fake.attrsErr = storage.ErrBucketNotExist

		publicAccessPrevention := true
		bckt := newBucket("my-bucket", "us-central1", func(b *vedrov1alpha1.Bucket) {
			b.Spec.StorageClass = vedrov1alpha1.BucketStorageClassArchive
			b.Spec.Labels = map[string]string{"env": "prod"}
			b.Spec.Versioning = &vedrov1alpha1.BucketVersioningSpec{Enabled: true}
			b.Spec.PublicAccessPrevention = &publicAccessPrevention
		})

		state, err := bucket.EnsureBucket(ctx, bckt)
		Expect(err).NotTo(HaveOccurred())
		Expect(fake.created).NotTo(BeNil())
		Expect(fake.created.Location).To(Equal("us-central1"))
		Expect(fake.created.StorageClass).To(Equal("ARCHIVE"))
		Expect(fake.created.Labels).To(Equal(map[string]string{"env": "prod"}))
		Expect(fake.created.VersioningEnabled).To(BeTrue())
		Expect(fake.created.PublicAccessPrevention).To(Equal(storage.PublicAccessPreventionEnforced))
		Expect(state.Applied.StorageClass).To(Equal(vedrov1alpha1.BucketStorageClassArchive))
		Expect(state.Applied.Labels).To(Equal(map[string]string{"env": "prod"}))
		Expect(state.Applied.Versioning.Enabled).To(BeTrue())
		Expect(*state.Applied.PublicAccessPrevention).To(BeTrue())
	})

	It("returns an error when creating a bucket fails", func() {
		fake.attrsErr = storage.ErrBucketNotExist
		fake.createErr = errors.New("network error")

		bckt := newBucket("my-bucket", "europe-west1", func(b *vedrov1alpha1.Bucket) {
			b.Spec.StorageClass = vedrov1alpha1.BucketStorageClassStandard
		})

		_, err := bucket.EnsureBucket(ctx, bckt)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("create bucket \"my-bucket\""))
		Expect(err.Error()).To(ContainSubstring("network error"))
	})

	It("returns an error when fetching bucket attributes fails", func() {
		fake.attrsErr = errors.New("permission denied")

		bckt := newBucket("my-bucket", "europe-west1", func(b *vedrov1alpha1.Bucket) {
			b.Spec.StorageClass = vedrov1alpha1.BucketStorageClassStandard
		})

		_, err := bucket.EnsureBucket(ctx, bckt)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("get bucket attrs \"my-bucket\""))
		Expect(err.Error()).To(ContainSubstring("permission denied"))
	})

	It("returns an error for an unmapped storage class", func() {
		fake.attrsErr = storage.ErrBucketNotExist

		bckt := newBucket("my-bucket", "europe-west1", func(b *vedrov1alpha1.Bucket) {
			b.Spec.StorageClass = vedrov1alpha1.BucketStorageClass("Glacier")
		})

		_, err := bucket.EnsureBucket(ctx, bckt)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("Glacier"))
	})

	It("returns the existing state when the bucket already matches the spec", func() {
		fake.attrs = &storage.BucketAttrs{
			Location:     "EUROPE-WEST1",
			StorageClass: "STANDARD",
		}

		bckt := newBucket("my-bucket", "europe-west1", func(b *vedrov1alpha1.Bucket) {
			b.Spec.StorageClass = vedrov1alpha1.BucketStorageClassStandard
		})

		state, err := bucket.EnsureBucket(ctx, bckt)
		Expect(err).NotTo(HaveOccurred())
		Expect(fake.updated).To(BeNil())
		Expect(state.ExternalName).To(Equal("my-bucket"))
		Expect(state.Location).To(Equal("EUROPE-WEST1"))
	})

	It("returns an error when the existing bucket is in a different location", func() {
		fake.attrs = &storage.BucketAttrs{
			Location:     "US-CENTRAL1",
			StorageClass: "STANDARD",
		}

		bckt := newBucket("my-bucket", "europe-west1", func(b *vedrov1alpha1.Bucket) {
			b.Spec.StorageClass = vedrov1alpha1.BucketStorageClassStandard
		})

		_, err := bucket.EnsureBucket(ctx, bckt)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("already exists in location \"US-CENTRAL1\""))
		Expect(err.Error()).To(ContainSubstring("desired location is \"EUROPE-WEST1\""))
	})

	It("updates the storage class when it differs", func() {
		fake.attrs = &storage.BucketAttrs{
			Location:     "EUROPE-WEST1",
			StorageClass: "STANDARD",
		}

		bckt := newBucket("my-bucket", "europe-west1", func(b *vedrov1alpha1.Bucket) {
			b.Spec.StorageClass = vedrov1alpha1.BucketStorageClassInfrequentAccess
		})

		state, err := bucket.EnsureBucket(ctx, bckt)
		Expect(err).NotTo(HaveOccurred())
		Expect(fake.updated).NotTo(BeNil())
		Expect(fake.updated.StorageClass).To(Equal("NEARLINE"))
		Expect(state.Applied.StorageClass).To(Equal(vedrov1alpha1.BucketStorageClassInfrequentAccess))
	})

	It("updates versioning when it differs", func() {
		fake.attrs = &storage.BucketAttrs{
			Location:          "EUROPE-WEST1",
			StorageClass:      "STANDARD",
			VersioningEnabled: false,
		}

		bckt := newBucket("my-bucket", "europe-west1", func(b *vedrov1alpha1.Bucket) {
			b.Spec.StorageClass = vedrov1alpha1.BucketStorageClassStandard
			b.Spec.Versioning = &vedrov1alpha1.BucketVersioningSpec{Enabled: true}
		})

		state, err := bucket.EnsureBucket(ctx, bckt)
		Expect(err).NotTo(HaveOccurred())
		Expect(fake.updated).NotTo(BeNil())
		Expect(fake.updated.VersioningEnabled).To(Equal(interface{}(true)))
		Expect(state.Applied.Versioning.Enabled).To(BeTrue())
	})

	It("updates labels when they differ", func() {
		fake.attrs = &storage.BucketAttrs{
			Location:     "EUROPE-WEST1",
			StorageClass: "STANDARD",
			Labels:       map[string]string{"env": "dev"},
		}

		bckt := newBucket("my-bucket", "europe-west1", func(b *vedrov1alpha1.Bucket) {
			b.Spec.StorageClass = vedrov1alpha1.BucketStorageClassStandard
			b.Spec.Labels = map[string]string{"env": "prod"}
		})

		state, err := bucket.EnsureBucket(ctx, bckt)
		Expect(err).NotTo(HaveOccurred())
		Expect(fake.updated).NotTo(BeNil())
		Expect(state.Applied.Labels).To(Equal(map[string]string{"env": "prod"}))
	})

	It("updates public access prevention when it differs", func() {
		fake.attrs = &storage.BucketAttrs{
			Location:               "EUROPE-WEST1",
			StorageClass:           "STANDARD",
			PublicAccessPrevention: storage.PublicAccessPreventionInherited,
		}

		publicAccessPrevention := true
		bckt := newBucket("my-bucket", "europe-west1", func(b *vedrov1alpha1.Bucket) {
			b.Spec.StorageClass = vedrov1alpha1.BucketStorageClassStandard
			b.Spec.PublicAccessPrevention = &publicAccessPrevention
		})

		state, err := bucket.EnsureBucket(ctx, bckt)
		Expect(err).NotTo(HaveOccurred())
		Expect(fake.updated).NotTo(BeNil())
		Expect(fake.updated.PublicAccessPrevention).To(Equal(storage.PublicAccessPreventionEnforced))
		Expect(*state.Applied.PublicAccessPrevention).To(BeTrue())
	})

	It("updates labels when spec.Labels is nil labels in status.Applied.Labels", func() {
		fake.attrs = &storage.BucketAttrs{
			Location:     "EUROPE-WEST1",
			StorageClass: "STANDARD",
			Labels:       map[string]string{"env": "dev"},
		}

		bckt := newBucket("my-bucket", "europe-west1", func(b *vedrov1alpha1.Bucket) {
			b.Spec.StorageClass = vedrov1alpha1.BucketStorageClassStandard
			b.Status.Applied = &vedrov1alpha1.BucketAppliedState{
				StorageClass: vedrov1alpha1.BucketStorageClassStandard,
				Labels:       map[string]string{"env": "dev"},
			}
		})

		state, err := bucket.EnsureBucket(ctx, bckt)
		Expect(err).NotTo(HaveOccurred())
		Expect(fake.updated).NotTo(BeNil())
		Expect(state.Applied.Labels).To(BeEmpty())
	})

	It("returns an error when updating the bucket fails", func() {
		fake.attrs = &storage.BucketAttrs{
			Location:     "EUROPE-WEST1",
			StorageClass: "STANDARD",
		}
		fake.updateErr = errors.New("update failed")

		bckt := newBucket("my-bucket", "europe-west1", func(b *vedrov1alpha1.Bucket) {
			b.Spec.StorageClass = vedrov1alpha1.BucketStorageClassInfrequentAccess
		})

		_, err := bucket.EnsureBucket(ctx, bckt)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("update bucket \"my-bucket\""))
		Expect(err.Error()).To(ContainSubstring("update failed"))
	})

	It("preserves unmodified applied state from status", func() {
		fake.attrs = &storage.BucketAttrs{
			Location:     "EUROPE-WEST1",
			StorageClass: "STANDARD",
		}

		bckt := newBucket("my-bucket", "europe-west1", func(b *vedrov1alpha1.Bucket) {
			b.Spec.StorageClass = vedrov1alpha1.BucketStorageClassStandard
			b.Status.Applied = &vedrov1alpha1.BucketAppliedState{
				StorageClass: vedrov1alpha1.BucketStorageClassStandard,
				//		Labels:       map[string]string{"team": "platform"},
			}
		})

		state, err := bucket.EnsureBucket(ctx, bckt)
		Expect(err).NotTo(HaveOccurred())
		Expect(fake.updated).To(BeNil())
		Expect(state.Applied.StorageClass).To(Equal(vedrov1alpha1.BucketStorageClassStandard))
		//	Expect(state.Applied.Labels).To(Equal(map[string]string{"team": "platform"}))
	})
})

type fakeStorageClient struct {
	bucket bucketHandle
}

func (f *fakeStorageClient) Bucket(name string) bucketHandle {
	return f.bucket
}

type fakeBucketHandle struct {
	attrs     *storage.BucketAttrs
	attrsErr  error
	createErr error
	updateErr error
	created   *storage.BucketAttrs
	updated   *storage.BucketAttrsToUpdate
}

func (f *fakeBucketHandle) Attrs(ctx context.Context) (*storage.BucketAttrs, error) {
	if f.attrsErr != nil {
		return nil, f.attrsErr
	}
	return f.attrs, nil
}

func (f *fakeBucketHandle) Create(ctx context.Context, projectID string, attrs *storage.BucketAttrs) error {
	f.created = attrs
	if f.createErr != nil {
		return f.createErr
	}
	f.attrs = attrs
	return nil
}

func (f *fakeBucketHandle) Update(ctx context.Context, uattrs storage.BucketAttrsToUpdate) (*storage.BucketAttrs, error) {
	f.updated = &uattrs
	if f.updateErr != nil {
		return nil, f.updateErr
	}

	if f.attrs != nil {
		if uattrs.StorageClass != "" {
			f.attrs.StorageClass = uattrs.StorageClass
		}
		if uattrs.VersioningEnabled != nil {
			if v, ok := uattrs.VersioningEnabled.(bool); ok {
				f.attrs.VersioningEnabled = v
			}
		}
		if uattrs.PublicAccessPrevention != storage.PublicAccessPreventionUnknown {
			f.attrs.PublicAccessPrevention = uattrs.PublicAccessPrevention
		}
	}

	return f.attrs, nil
}
