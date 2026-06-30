package cloud

import (
	"context"
	"errors"

	vedro "github.com/svetoch-dev/vedro/api/v1alpha1"
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
	Lifecycle              LifecycleCapabilities
	StorageClass           StorageClassCapabilities
	Versioning             bool
	PublicAccessPrevention bool
	Labels                 bool
}

type LifecycleCapabilities struct {
	RuleNames      bool
	RuleExpiration bool
}

type StorageClassCapabilities struct {
	Ice  bool
	Cold bool
	Warm bool
}

func (bc BucketCapabilities) LifecycleSupported() bool {
	return bc.Lifecycle.RuleExpiration
}

type BucketAttrs struct {
	Name     string
	Location string

	Properties *vedro.BucketProperties
}

type Change[T any] struct {
	Set   bool
	Value T
}

type ObjectVersion struct {
	Name    string
	Version int64
}

type BucketPatch struct {
	StorageClass           Change[vedro.BucketStorageClass]
	Labels                 Change[map[string]string]
	Versioning             Change[*vedro.BucketVersioning]
	PublicAccessPrevention Change[*bool]
	Lifecycle              Change[*vedro.BucketLifecycle]
	CloudSpecificConfig    Change[*vedro.BucketCloudSpecificConfig]
}

func (p BucketPatch) HasChanges() bool {
	return p.StorageClass.Set ||
		p.Labels.Set ||
		p.Versioning.Set ||
		p.PublicAccessPrevention.Set ||
		p.CloudSpecificConfig.Set ||
		p.Lifecycle.Set
}

type BucketAPI interface {
	GetBucket(ctx context.Context, name string) (*BucketAttrs, error)
	CreateBucket(ctx context.Context, name string, attrs BucketAttrs) error
	UpdateBucket(ctx context.Context, name string, patch BucketPatch) (*BucketAttrs, error)

	ProcessObjects(
		ctx context.Context,
		bucket string,
		process func(object ObjectVersion) error,
	) error

	DeleteObject(
		ctx context.Context,
		bucket string,
		object ObjectVersion,
	) error

	DeleteBucket(ctx context.Context, name string) error
}

type BucketProvider interface {
	ValidateBucketSpec(bckt vedro.Bucket, pType vedro.ProviderType) validation.ValidationResult

	EnsureBucket(
		ctx context.Context,
		spec vedro.Bucket,
	) (*BucketAttrs, error)

	DeleteBucket(
		ctx context.Context,
		bckt vedro.Bucket,
	) error
}
