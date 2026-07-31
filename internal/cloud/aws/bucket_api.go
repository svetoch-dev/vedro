package aws

import (
	"context"
	"errors"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/smithy-go"
	"github.com/svetoch-dev/vedro/internal/cloud"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

type S3API struct {
	Client *s3.Client
}

func isS3NotFoundLike(err error, codes ...string) bool {
	var apiErr smithy.APIError
	if !errors.As(err, &apiErr) {
		return false
	}

	for _, code := range codes {
		if apiErr.ErrorCode() == code {
			return true
		}
	}

	return false
}

func (s *S3API) GetBucket(
	ctx context.Context,
	name string,
) (*cloud.BucketAttrs, error) {
	return &cloud.BucketAttrs{}, nil
}

func (s *S3API) CreateBucket(ctx context.Context, name string, attrs cloud.BucketAttrs) (*cloud.BucketAttrs, error) {
	return nil, nil
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
	p := s3.NewListObjectVersionsPaginator(s.Client, &s3.ListObjectVersionsInput{
		Bucket: aws.String(bucket),
	})

	for p.HasMorePages() {
		page, err := p.NextPage(ctx)
		if err != nil {
			if isS3NotFoundLike(err, "NoSuchBucket", "NotFound") {
				return cloud.ErrBucketNotFound
			}
			return fmt.Errorf("list object versions for bucket %q: %w", bucket, err)
		}

		for _, v := range page.Versions {
			if err := process(cloud.ObjectVersion{
				Name:    aws.ToString(v.Key),
				Version: v.VersionId,
			}); err != nil {
				return err
			}
		}

		for _, v := range page.DeleteMarkers {
			if err := process(cloud.ObjectVersion{
				Name:    aws.ToString(v.Key),
				Version: v.VersionId,
			}); err != nil {
				return err
			}

		}
	}

	return nil
}

func (s *S3API) DeleteObject(
	ctx context.Context,
	bucket string,
	object cloud.ObjectVersion,
) error {
	_, err := s.Client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket:    aws.String(bucket),
		Key:       aws.String(object.Name),
		VersionId: object.Version,
	})
	if isS3NotFoundLike(err, "NoSuchKey", "NoSuchVersion", "NotFound") {
		return cloud.ErrBucketObjectNotFound
	}
	return err
}

func (s *S3API) DeleteBucket(ctx context.Context, name string) error {
	return nil
}

func (y *S3API) Close(ctx context.Context) error {
	return nil
}
