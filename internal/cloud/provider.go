package cloud

import (
	"context"

	vedrov1alpha1 "github.com/svetoch-dev/vedro/api/v1alpha1"
	"github.com/svetoch-dev/vedro/internal/validation"
)

type BucketState struct {
	ExternalName string
	Location     string
}

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
	PublicAccess                 bool
	StorageClassArchive          bool
	StorageClassInfrequentAccess bool
	Labels                       bool
}

type BucketProvider interface {
	ValidateBucketSpec(spec vedrov1alpha1.BucketSpec) validation.ValidationResult

	EnsureBucket(
		ctx context.Context,
		spec vedrov1alpha1.BucketSpec,
	) (*BucketState, error)

	DeleteBucket(
		ctx context.Context,
		status vedrov1alpha1.BucketStatus,
	) error
}
