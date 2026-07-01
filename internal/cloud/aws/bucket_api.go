package aws

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/svetoch-dev/vedro/internal/cloud"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

type AccessKey struct {
	AccessKeyID     string
	SecretAccessKey string
}

type AwsAPI struct {
	Client *s3.Client
}

func (a *AwsAPI) GetBucket(
	ctx context.Context,
	name string,
) (*cloud.BucketAttrs, error) {

	return &cloud.BucketAttrs{}, nil
}

func (a *AwsAPI) CreateBucket(ctx context.Context, name string, attrs cloud.BucketAttrs) error {
	return nil
}

func (a *AwsAPI) UpdateBucket(ctx context.Context, name string, patch cloud.BucketPatch) (*cloud.BucketAttrs, error) {
	log.FromContext(ctx).V(1).Info("Updating bucket")

	return &cloud.BucketAttrs{}, nil
}

func (a *AwsAPI) ProcessObjects(
	ctx context.Context,
	bucket string,
	process func(cloud.ObjectVersion) error,
) error {
	return nil
}

func (a *AwsAPI) DeleteObject(
	ctx context.Context,
	bucket string,
	object cloud.ObjectVersion,
) error {
	return nil
}

func (a *AwsAPI) DeleteBucket(ctx context.Context, name string) error {
	return nil
}
