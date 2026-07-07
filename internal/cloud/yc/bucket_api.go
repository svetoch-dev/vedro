package yc

import (
	"context"
	"fmt"

	vedro "github.com/svetoch-dev/vedro/api/v1alpha1"
	"github.com/svetoch-dev/vedro/internal/cloud"
	"github.com/svetoch-dev/vedro/internal/cloud/aws"
	"github.com/svetoch-dev/vedro/internal/helpers"
	storageapi "github.com/yandex-cloud/go-genproto/yandex/cloud/storage/v1"
	storagesdk "github.com/yandex-cloud/go-sdk/services/storage/v1"
	ycsdk "github.com/yandex-cloud/go-sdk/v2"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/fieldmaskpb"
	"google.golang.org/protobuf/types/known/wrapperspb"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

var (
	ycStorageClassMapping = map[string]vedro.BucketStorageClass{
		"STANDARD":    vedro.BucketStorageClassStandard,
		"NEARLINE":    vedro.BucketStorageClassCold,
		"COLD":        vedro.BucketStorageClassCold,
		"STANDARD_IA": vedro.BucketStorageClassCold,
		"ICE":         vedro.BucketStorageClassIce,
		"GLACIER":     vedro.BucketStorageClassIce,
	}
	storageClassMapping = map[vedro.BucketStorageClass]string{
		vedro.BucketStorageClassStandard: "STANDARD",
		vedro.BucketStorageClassCold:     "COLD",
		vedro.BucketStorageClassIce:      "ICE",
	}
)

type staticS3AccessKey struct {
	accessKeyID     string
	secretAccessKey string
	id              string
}

type ycAPI struct {
	sdk       *ycsdk.SDK
	accessKey *staticS3AccessKey
	awsAPI    *aws.S3API
	saId      string
	folderId  string
	location  string
}

func isNotFound(err error) bool {
	if err == nil {
		return false
	}

	if st, ok := status.FromError(err); ok {
		return st.Code() == codes.NotFound
	}

	return false
}

func fromYcVersioning(versioning storageapi.Versioning) *vedro.BucketVersioning {
	switch versioning {
	case storageapi.Versioning_VERSIONING_ENABLED:
		return &vedro.BucketVersioning{
			Enabled: true,
		}

	case storageapi.Versioning_VERSIONING_SUSPENDED:
		return &vedro.BucketVersioning{
			Enabled: false,
		}
	default:
		return nil
	}
}

func toYcVersioning(versioning *vedro.BucketVersioning) storageapi.Versioning {
	if versioning == nil {
		return storageapi.Versioning_VERSIONING_DISABLED
	}

	if versioning.Enabled {
		return storageapi.Versioning_VERSIONING_ENABLED
	} else {
		return storageapi.Versioning_VERSIONING_SUSPENDED
	}
}

func fromYcLifecycle(lifecycleRules []*storageapi.LifecycleRule) *vedro.BucketLifecycle {
	lifecycle := &vedro.BucketLifecycle{}

	for _, rule := range lifecycleRules {
		if rule == nil {
			continue
		}
		if rule.Expiration != nil {
			var name *string
			if rule.Id != nil {
				name = helpers.Ptr(rule.Id.Value)
			}
			if rule.Expiration.Days != nil {
				lifecycle.Rules = append(lifecycle.Rules, vedro.BucketLifecycleRule{
					Name:    name,
					Enabled: rule.Enabled,
					AgeDays: helpers.Ptr(rule.Expiration.Days.Value),
					Action:  vedro.BucketLifecycleActionDelete,
				})
			}
		}
	}

	if len(lifecycle.Rules) == 0 {
		return nil
	}

	return lifecycle
}

func toYcLifecycle(lifecycle *vedro.BucketLifecycle) []*storageapi.LifecycleRule {
	rules := []*storageapi.LifecycleRule{}

	if lifecycle == nil || len(lifecycle.Rules) == 0 {
		return rules
	}

	for _, rule := range lifecycle.Rules {
		var id *wrapperspb.StringValue
		if rule.Name != nil {
			id = &wrapperspb.StringValue{
				Value: *rule.Name,
			}
		}
		if rule.AgeDays != nil {
			if rule.Action == vedro.BucketLifecycleActionDelete {
				rules = append(rules, &storageapi.LifecycleRule{
					Id:      id,
					Enabled: rule.Enabled,
					Expiration: &storageapi.LifecycleRule_Expiration{
						Days: &wrapperspb.Int64Value{
							Value: *rule.AgeDays,
						},
					},
				})
			}
		}
	}
	return rules
}

func fromYcTags(tags []*storageapi.Tag) map[string]string {
	dict := map[string]string{}

	for _, tag := range tags {
		dict[tag.Key] = tag.Value
	}
	return dict
}

func toYcTags(labels map[string]string) []*storageapi.Tag {
	tags := []*storageapi.Tag{}

	for key, value := range labels {
		tags = append(tags, &storageapi.Tag{
			Key:   key,
			Value: value,
		})
	}
	return tags
}

func fromYcBucket(bucket *storageapi.Bucket, location string) (*cloud.BucketAttrs, error) {
	if bucket == nil {
		return nil, fmt.Errorf("yc storageapi.Bucket is nil")
	}

	sc, ok := ycStorageClassMapping[bucket.DefaultStorageClass]
	if !ok {
		return nil, fmt.Errorf("yc StorageClass %s doesnt map to any bucket StorageClass", bucket.DefaultStorageClass)
	}

	return &cloud.BucketAttrs{
		Name:     bucket.Name,
		Location: location,
		Properties: &vedro.BucketProperties{
			Versioning:   fromYcVersioning(bucket.Versioning),
			Lifecycle:    fromYcLifecycle(bucket.LifecycleRules),
			Labels:       fromYcTags(bucket.Tags),
			StorageClass: sc,
		},
	}, nil

}

func toCreateBucketRequest(attrs cloud.BucketAttrs, folderId string) (*storageapi.CreateBucketRequest, error) {
	request := &storageapi.CreateBucketRequest{
		Name:           attrs.Name,
		FolderId:       folderId,
		Tags:           toYcTags(attrs.Properties.Labels),
		Versioning:     toYcVersioning(attrs.Properties.Versioning),
		LifecycleRules: toYcLifecycle(attrs.Properties.Lifecycle),
	}

	storageClass, ok := storageClassMapping[attrs.Properties.StorageClass]
	if !ok {
		return nil, fmt.Errorf(
			"bucket storage class %q does not map to YC",
			attrs.Properties.StorageClass,
		)
	}

	request.DefaultStorageClass = storageClass

	return request, nil
}

func patchYcBucketAttrs(patch cloud.BucketPatch, name string) (*storageapi.UpdateBucketRequest, error) {
	update := &storageapi.UpdateBucketRequest{
		Name:       name,
		UpdateMask: &fieldmaskpb.FieldMask{},
	}

	if patch.StorageClass.Set {
		storageClass, ok := storageClassMapping[patch.StorageClass.Value]
		if !ok {
			return nil, fmt.Errorf(
				"bucket storage class %q does not map to YC",
				patch.StorageClass.Value,
			)
		}
		update.DefaultStorageClass = storageClass
		update.UpdateMask.Paths = append(update.UpdateMask.Paths, "default_storage_class")
	}

	if patch.Labels.Set {
		update.Tags = toYcTags(patch.Labels.Value)
		update.UpdateMask.Paths = append(update.UpdateMask.Paths, "tags")
	}

	if patch.Versioning.Set {
		update.Versioning = toYcVersioning(patch.Versioning.Value)
		// cant set versioning back to disabled so we just suspend it
		if update.Versioning == storageapi.Versioning_VERSIONING_DISABLED {
			update.Versioning = storageapi.Versioning_VERSIONING_SUSPENDED
		}
		update.UpdateMask.Paths = append(update.UpdateMask.Paths, "versioning")

	}
	if patch.Lifecycle.Set {
		update.LifecycleRules = toYcLifecycle(patch.Lifecycle.Value)
		update.UpdateMask.Paths = append(update.UpdateMask.Paths, "lifecycle_rules")
	}

	return update, nil
}

func (y *ycAPI) GetBucket(
	ctx context.Context,
	name string,
) (*cloud.BucketAttrs, error) {
	bucketClient := storagesdk.NewBucketClient(y.sdk)
	bucket, err := bucketClient.Get(ctx, &storageapi.GetBucketRequest{
		Name: name,
		View: storageapi.GetBucketRequest_VIEW_FULL,
	})
	if err != nil {
		if isNotFound(err) {
			return nil, cloud.ErrBucketNotFound
		}

		return nil, fmt.Errorf("get yc bucket %q: %w", name, err)
	}
	return fromYcBucket(bucket, y.location)

}

func (y *ycAPI) CreateBucket(ctx context.Context, name string, attrs cloud.BucketAttrs) error {
	bucketClient := storagesdk.NewBucketClient(y.sdk)
	request, err := toCreateBucketRequest(attrs, y.folderId)
	if err != nil {
		return err
	}
	op, err := bucketClient.Create(ctx, request)
	if err != nil {
		return fmt.Errorf("yc create bucket operation: %v", err)
	}

	_, err = op.Wait(ctx)
	if err != nil {
		return fmt.Errorf("yc wait create bucket: %v", err)
	}

	return nil
}

func (y *ycAPI) UpdateBucket(ctx context.Context, name string, patch cloud.BucketPatch) (*cloud.BucketAttrs, error) {
	log.FromContext(ctx).V(1).Info("Updating bucket")
	bucketClient := storagesdk.NewBucketClient(y.sdk)

	request, err := patchYcBucketAttrs(patch, name)
	if err != nil {
		return nil, err
	}

	op, err := bucketClient.Update(ctx, request)
	if err != nil {
		return nil, fmt.Errorf("update yc bucket default storage class: %w", err)
	}

	bucket, err := op.Wait(ctx)
	if err != nil {
		return nil, fmt.Errorf("wait yc bucket storage class update: %w", err)
	}

	return fromYcBucket(bucket, y.location)
}

func (y *ycAPI) newAWSAPI(ctx context.Context) error {
	accessKey, err := createStaticS3AccessKey(ctx, y.sdk, y.saId)

	if err != nil {
		return err
	}

	y.accessKey = accessKey

	s3Client, err := newS3Client(ctx, accessKey, y.location)
	if err != nil {
		cleanupErr := deleteStaticS3AccessKey(ctx, y.sdk, y.accessKey.id)
		if cleanupErr != nil {
			return fmt.Errorf("create yc s3 client: %w; cleanup static access key: %v", err, cleanupErr)
		}
		y.accessKey = nil

		return err
	}

	y.awsAPI = &aws.S3API{
		Client: s3Client,
	}
	return nil
}

func (y *ycAPI) ProcessObjects(
	ctx context.Context,
	bucket string,
	process func(cloud.ObjectVersion) error,
) error {

	if y.awsAPI == nil {
		err := y.newAWSAPI(ctx)
		if err != nil {
			return err
		}
	}

	err := y.awsAPI.ProcessObjects(ctx, bucket, process)
	if err != nil {
		return err
	}

	return nil
}

func (y *ycAPI) DeleteObject(
	ctx context.Context,
	bucket string,
	object cloud.ObjectVersion,
) error {
	if y.awsAPI == nil {
		err := y.newAWSAPI(ctx)
		if err != nil {
			return err
		}
	}

	err := y.awsAPI.DeleteObject(ctx, bucket, object)
	if err != nil {
		return err
	}

	return nil
}

func (y *ycAPI) DeleteBucket(ctx context.Context, name string) error {
	return nil
}

func (y *ycAPI) Close(ctx context.Context) error {
	if y.sdk == nil {
		return nil
	}

	if y.accessKey != nil {
		err := deleteStaticS3AccessKey(ctx, y.sdk, y.accessKey.id)
		if err != nil {
			return err
		}
	}

	return y.sdk.Shutdown(ctx)
}
