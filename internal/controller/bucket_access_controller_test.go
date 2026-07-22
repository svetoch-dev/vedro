package controller

import (
	"context"

	vedro "github.com/svetoch-dev/vedro/api/v1alpha1"
	"github.com/svetoch-dev/vedro/internal/cloud"
)

type fakeBucketAccess struct {
}

func (b *fakeBucketAccess) EnsureBucketAccess(
	ctx context.Context,
	bucket vedro.Bucket,
	principal *vedro.CloudPrincipal,
	access vedro.BucketAccess,
) (*cloud.BucketAccessAttrs, error) {
	return nil, nil
}

func (b *fakeBucketAccess) DeleteBucketAccess(ctx context.Context, access vedro.BucketAccess) error {
	return nil
}
