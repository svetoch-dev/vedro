package aws

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/svetoch-dev/vedro/internal/cloud"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

type S3API struct {
	Client *s3.Client
}

// func isS3NotFoundLike(err error, codes ...string) bool {
// 	var apiErr smithy.APIError
// 	if !errors.As(err, &apiErr) {
// 		return false
// 	}

// 	for _, code := range codes {
// 		if apiErr.ErrorCode() == code {
// 			return true
// 		}
// 	}

// 	return false
// }

// func getS3Location(ctx context.Context, client *s3.Client, name string) (*s3.GetBucketLocationOutput, error) {
// 	return client.GetBucketLocation(ctx, &s3.GetBucketLocationInput{
// 		Bucket: aws.String(name),
// 	})
// }

// func getS3BucketVersioning(ctx context.Context, client *s3.Client, name string) (*s3.GetBucketVersioningOutput, error) {
// 	return client.GetBucketVersioning(ctx, &s3.GetBucketVersioningInput{
// 		Bucket: aws.String(name),
// 	})
// }

// func getS3BucketLifecycle(ctx context.Context, client *s3.Client, name string) (*s3.GetBucketLifecycleConfigurationOutput, error) {
// 	return client.GetBucketLifecycleConfiguration(ctx, &s3.GetBucketLifecycleConfigurationInput{
// 		Bucket: aws.String(name),
// 	})
// }
// func getS3BucketTags(ctx context.Context, client *s3.Client, name string) (*s3.GetBucketTaggingOutput, error) {
// 	return client.GetBucketTagging(ctx, &s3.GetBucketTaggingInput{
// 		Bucket: aws.String(name),
// 	})
// }

// func fromS3Location(out *s3.GetBucketLocationOutput, err error) (string, error) {
// 	if err != nil {
// 		return "", fmt.Errorf("get bucket location: %w", err)
// 	}

// 	if out == nil {
// 		return "", fmt.Errorf("no output for bucket location")
// 	}

// 	// Empty / zero location means us-east-1 for classic S3 buckets.
// 	if out.LocationConstraint == "" {
// 		return "us-east-1", nil
// 	}

// 	return string(out.LocationConstraint), nil

// }

// func fromS3Versioning(out *s3.GetBucketVersioningOutput, err error) (*vedro.BucketVersioning, error) {
// 	if err != nil {
// 		return nil, fmt.Errorf("get bucket versioning: %w", err)
// 	}
// 	if out == nil || out.Status == "" {
// 		return nil, fmt.Errorf("no output for bucket versioning")
// 	}

// 	switch out.Status {
// 	case types.BucketVersioningStatusEnabled:
// 		return &vedro.BucketVersioning{
// 			Enabled: true,
// 		}

// 	case types.BucketVersioningStatusSuspended:
// 		return &vedro.BucketVersioning{
// 			Enabled: false,
// 		}

// 	default:
// 		return &vedro.BucketVersioning{
// 			Enabled: false,
// 		}
// 	}

// }

// func fromS3Lifecycle(out *s3.GetBucketLifecycleConfigurationOutput, err error) (*vedro.BucketLifecycle, error) {
// 	if !isS3NotFoundLike(err, "NoSuchLifecycleConfiguration") {
// 		return &vedro.BucketLifecycle{}, nil
// 	}

// 	if err != nil {
// 		return nil, fmt.Errorf("get bucket lifecycle: %w", err)
// 	}

// 	lifecycle := &vedro.BucketLifecycle{}

// 	for _, rule := range out.Rules {
// 		lifecycle.Rules = append(lifecycle.Rules, vedro.BucketLifecycleRule{
// 			Name:    rule.ID,
// 			Enabled: true,
// 			AgeDays: helpers.Ptr(int64(*rule.Expiration.Days)),
// 			Action:  vedro.BucketLifecycleActionDelete,
// 		})
// 	}

// }

func (s *S3API) GetBucket(
	ctx context.Context,
	name string,
) (*cloud.BucketAttrs, error) {
	// cfg := &cloud.BucketAttrs{
	// 	Name: name,
	// }

	// location, err := getS3Location(ctx, s.Client)
	// vlocation, err := fromS3Location(location, err)
	// if err != nil {
	// 	return nil, err
	// }
	// cfg.Location = vlocation

	// props := &vedro.BucketProperties{}

	// versioning, err := getS3BucketVersioning(ctx, s.Client, name)
	// vVersioning, err := fromS3Versioning(versioning, err)
	// if err != nil {
	// 	return nil, err

	// }
	// props.Versioning = vVersioning

	// lifecycle, err := getS3BucketLifecycle(ctx, s.Client, name)

	// tagging, err := getS3BucketTags(ctx, s.Client, name)
	// if err == nil {
	// 	for _, tag := range tagging.TagSet {
	// 		cfg.Tags[aws.ToString(tag.Key)] = aws.ToString(tag.Value)
	// 	}
	// } else if !isS3NotFoundLike(err, "NoSuchTagSet") {
	// 	return nil, fmt.Errorf("get bucket tagging: %w", err)
	// }

	// _ = s3types.BucketVersioningStatusEnabled // keeps s3types import useful if you want typed comparisons

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
	p := s3.NewListObjectVersionsPaginator(s.Client, &s3.ListObjectVersionsInput{
		Bucket: aws.String(bucket),
	})

	for p.HasMorePages() {
		page, err := p.NextPage(ctx)
		if err != nil {
			return err
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
	return err
}

func (s *S3API) DeleteBucket(ctx context.Context, name string) error {
	return nil
}
