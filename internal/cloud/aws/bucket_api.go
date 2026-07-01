package aws

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/svetoch-dev/vedro/internal/cloud"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

type S3API struct {
	Client *s3.Client
}

func (s *S3API) GetBucket(
	ctx context.Context,
	name string,
) (*cloud.BucketAttrs, error) {

	return &cloud.BucketAttrs{}, nil
}

func (s *S3API) CreateBucket(ctx context.Context, name string, attrs cloud.BucketAttrs) error {
	return nil
}

func (s *S3API) UpdateBucket(ctx context.Context, name string, patch cloud.BucketPatch) (*cloud.BucketAttrs, error) {
	log.FromContext(ctx).V(1).Info("Updating bucket")

	return &cloud.BucketAttrs{}, nil
}

func (s *S3API) ProcessObjects(
	ctx context.Context,
	bucket string,
	process func(cloud.ObjectVersion) error,
) error {
	return nil
}

func (s *S3API) DeleteObject(
	ctx context.Context,
	bucket string,
	object cloud.ObjectVersion,
) error {
	return nil
}

func (s *S3API) DeleteBucket(ctx context.Context, name string) error {
	return nil
}
