package cloud

import (
	"context"
	"errors"

	vedrov1alpha1 "github.com/svetoch-dev/vedro/api/v1alpha1"
	"github.com/svetoch-dev/vedro/internal/validation"
)

var (
	ErrBucketNotFound       = errors.New("bucket not found")
	ErrBucketObjectNotFound = errors.New("bucket object not found")
)

type Provider interface {
	Bucket() BucketProvider
	Capabilities() Capabilities
}

type Capabilities struct {
	Bucket BucketCapabilities
}

type BucketCapabilities struct {
	Versioning                   bool
	LifecycleExpiration          bool
	PublicAccessPrevention       bool
	StorageClassArchive          bool
	StorageClassInfrequentAccess bool
	Labels                       bool
}

type BucketAttrs struct {
	Name     string
	Location string

	Properties *vedrov1alpha1.BucketProperties
}

type Change[T any] struct {
	Set   bool
	Value T
}

type ObjectVersion struct {
	Name       string
	Generation int64
}

type BucketPatch struct {
	StorageClass           Change[vedrov1alpha1.BucketStorageClass]
	Labels                 Change[map[string]string]
	Versioning             Change[*vedrov1alpha1.BucketVersioning]
	PublicAccessPrevention Change[*bool]
	Lifecycle              Change[*vedrov1alpha1.BucketLifecycle]
}

func (p BucketPatch) HasChanges() bool {
	return p.StorageClass.Set ||
		p.Labels.Set ||
		p.Versioning.Set ||
		p.PublicAccessPrevention.Set ||
		p.Lifecycle.Set
}

type BucketAPI interface {
	GetBucket(ctx context.Context, name string) (*BucketAttrs, error)
	CreateBucket(ctx context.Context, name string, attrs BucketAttrs) error
	UpdateBucket(ctx context.Context, name string, patch BucketPatch) (*BucketAttrs, error)

	ListObjects(
		ctx context.Context,
		bucket string,
		process func(ObjectVersion) error,
	) error

	DeleteObject(
		ctx context.Context,
		bucket string,
		object ObjectVersion,
	) error

	DeleteBucket(ctx context.Context, name string) error
}

type BucketProvider interface {
	ValidateBucketSpec(spec vedrov1alpha1.Bucket) validation.ValidationResult

	EnsureBucket(
		ctx context.Context,
		spec vedrov1alpha1.Bucket,
	) (*BucketAttrs, error)

	DeleteBucket(
		ctx context.Context,
		bckt vedrov1alpha1.Bucket,
	) error
}
