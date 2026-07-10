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
	Principal() PrincipalProvider
	Capabilities() Capabilities
	Cleanup(ctx context.Context) error
	ValidateProviderConfigSpec(cfg vedro.ProviderConfig) validation.ValidationResult
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

type PrincipalAttrs struct {
	Name string
	Id   string
}

type Change[T any] struct {
	Set   bool
	Value T
}

type ObjectVersion struct {
	Name    string
	Version *string
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

	// Close releases resources owned by the BucketAPI implementation.
	// It may perform remote cleanup, such as deleting temporary provider credentials.
	Close(ctx context.Context) error
}

type PrincipalAPI interface {
	GetPrincipal(ctx context.Context, name string) (*PrincipalAttrs, error)
	CreatePrincipal(ctx context.Context, name string, attrs PrincipalAttrs) error
	DeletePrincipal(ctx context.Context, name string) error
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

type PrincipalProvider interface {
	ValidatePrincipalSpec(principal vedro.CloudPrincipal) validation.ValidationResult

	EnsurePrincipal(
		ctx context.Context,
		principal vedro.CloudPrincipal,
	) (*PrincipalAttrs, error)

	DeletePrincipal(
		ctx context.Context,
		principal vedro.CloudPrincipal,
	) error
}
