package gcp

import (
	"context"

	"cloud.google.com/go/storage"
	vedrov1alpha1 "github.com/svetoch-dev/vedro/api/v1alpha1"
	"github.com/svetoch-dev/vedro/internal/cloud"
)

type Bucket struct {
	client *storage.Client
}

func (p *Bucket) ValidateBucketSpec(spec vedrov1alpha1.BucketSpec) cloud.ValidationResult {
	return cloud.Valid()
}

func (p *Bucket) EnsureBucket(ctx context.Context, spec vedrov1alpha1.BucketSpec) (*cloud.BucketState, error) {
	return nil, nil
}

func (p *Bucket) DeleteBucket(ctx context.Context, status vedrov1alpha1.BucketStatus) error {
	return nil
}
