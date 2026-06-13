package cloud

import (
	"context"

	vedrov1alpha1 "github.com/svetoch-dev/vedro/api/v1alpha1"
)

type BucketState struct {
	ExternalName string
	ExternalId   string
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

type ValidationResult struct {
	Valid   bool
	Reason  string
	Message string
	Errors  []ValidationError
}

type ValidationError struct {
	Field   string
	Reason  string
	Message string
}

func Valid() ValidationResult {
	return ValidationResult{Valid: true}
}

func Invalid(reason string, message string) ValidationResult {
	return ValidationResult{
		Valid:   false,
		Reason:  reason,
		Message: message,
	}
}

type BucketProvider interface {
	ValidateBucketSpec(spec vedrov1alpha1.BucketSpec) ValidationResult

	EnsureBucket(
		ctx context.Context,
		spec vedrov1alpha1.BucketSpec,
	) (*BucketState, error)

	DeleteBucket(
		ctx context.Context,
		status vedrov1alpha1.BucketStatus,
	) error
}
