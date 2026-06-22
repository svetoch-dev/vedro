package gcp

import (
	"context"
	"errors"
	"fmt"

	"cloud.google.com/go/storage"
	vedrov1alpha1 "github.com/svetoch-dev/vedro/api/v1alpha1"
	"github.com/svetoch-dev/vedro/internal/cloud"
	"github.com/svetoch-dev/vedro/internal/helpers"
)

var (
	gcsStorageClassMapping = map[string]vedrov1alpha1.BucketStorageClass{
		"STANDARD": vedrov1alpha1.BucketStorageClassStandard,
		"NEARLINE": vedrov1alpha1.BucketStorageClassInfrequentAccess,
		"COLDLINE": vedrov1alpha1.BucketStorageClassArchive,
		"ARCHIVE":  vedrov1alpha1.BucketStorageClassArchive,
	}
	gcsPublicAccessPreventionMapping = map[storage.PublicAccessPrevention]*bool{
		storage.PublicAccessPreventionInherited: helpers.Ptr(false),
		storage.PublicAccessPreventionEnforced:  helpers.Ptr(true),
		storage.PublicAccessPreventionUnknown:   nil,
	}
	publicAccessPreventionMapping = map[bool]storage.PublicAccessPrevention{
		false: storage.PublicAccessPreventionInherited,
		true:  storage.PublicAccessPreventionEnforced,
	}
	gcsLifeCycleMapping = map[string]vedrov1alpha1.BucketLifecycleAction{
		storage.DeleteAction: vedrov1alpha1.BucketLifecycleActionDelete,
	}
)

type bucketAPI struct {
	client    *storage.Client
	projectID string
}

func setGCSLabels(
	desiredLabels map[string]string,
	actualLabels map[string]string,
	attrs *storage.BucketAttrsToUpdate,
) {
	if attrs == nil {
		return
	}

	for k, v := range desiredLabels {
		attrs.SetLabel(k, v)
	}

	for k := range actualLabels {
		if _, ok := desiredLabels[k]; !ok {
			attrs.DeleteLabel(k)
		}
	}
}

func toGCSLifeCycle(lifeCycle vedrov1alpha1.BucketLifecycle) (storage.Lifecycle, error) {
	gcsLifeCycle := storage.Lifecycle{}

	for index, rule := range lifeCycle.Rules {
		if !rule.Enabled {
			continue
		}

		actionType, ok := helpers.FindKeyByValue(gcsLifeCycleMapping, rule.Action)
		if !ok {
			return gcsLifeCycle, fmt.Errorf(
				"lifecycle.rules[%d].action %s doesn't map to any GCS lifecycle action",
				index,
				rule.Action,
			)
		}

		gcsAction := storage.LifecycleAction{
			Type: actionType,
		}

		var condition *storage.LifecycleCondition

		if rule.AgeDays != nil {
			condition = &storage.LifecycleCondition{
				AgeInDays: *rule.AgeDays,
			}
		}

		if condition != nil {
			gcsLifeCycle.Rules = append(gcsLifeCycle.Rules, storage.LifecycleRule{
				Action:    gcsAction,
				Condition: *condition,
			})
		}
	}

	return gcsLifeCycle, nil
}

func fromGCSLifeCycle(lifecycle storage.Lifecycle) vedrov1alpha1.BucketLifecycle {
	var bucketLifeCycle vedrov1alpha1.BucketLifecycle

	for _, rule := range lifecycle.Rules {
		if rule.Condition.AgeInDays > 0 && rule.Action.Type == storage.DeleteAction {
			bucketLifeCycle.Rules = append(bucketLifeCycle.Rules, vedrov1alpha1.BucketLifecycleRule{
				Enabled: true,
				AgeDays: helpers.Ptr(rule.Condition.AgeInDays),
				Action:  gcsLifeCycleMapping[rule.Action.Type],
			})
		}
	}

	return bucketLifeCycle

}

func fromGCSBucketAttrs(attrs storage.BucketAttrs) (*cloud.BucketAttrs, error) {
	pap, ok := gcsPublicAccessPreventionMapping[attrs.PublicAccessPrevention]
	if !ok {
		return nil, fmt.Errorf("gcs PublicAccessPrevention %s doesnt map to any bucket StorageClass", attrs.PublicAccessPrevention)
	}

	sc, ok := gcsStorageClassMapping[attrs.StorageClass]
	if !ok {
		return nil, fmt.Errorf("gcs StorageClass %s doesnt map to any bucket StorageClass", attrs.StorageClass)
	}

	gcsLifeCycle := fromGCSLifeCycle(attrs.Lifecycle)

	return &cloud.BucketAttrs{
		Name:     attrs.Name,
		Location: attrs.Location,
		Properties: &vedrov1alpha1.BucketProperties{
			PublicAccessPrevention: pap,
			Versioning: &vedrov1alpha1.BucketVersioning{
				Enabled: attrs.VersioningEnabled,
			},
			StorageClass: sc,
			Labels:       attrs.Labels,
			Lifecycle:    &gcsLifeCycle,
		},
	}, nil
}

func toGCSBucketAttrs(attrs cloud.BucketAttrs) (*storage.BucketAttrs, error) {
	gcsAttrs := &storage.BucketAttrs{
		Name:     attrs.Name,
		Location: attrs.Location,
	}

	if attrs.Properties == nil {
		return gcsAttrs, nil
	}

	sc, ok := helpers.FindKeyByValue(gcsStorageClassMapping, attrs.Properties.StorageClass)
	if !ok {
		return nil, fmt.Errorf("gcs StorageClass %s doesnt map to any bucket StorageClass", attrs.Properties.StorageClass)
	}

	gcsAttrs.StorageClass = sc
	gcsAttrs.Labels = attrs.Properties.Labels

	if attrs.Properties.PublicAccessPrevention != nil {
		gcsAttrs.PublicAccessPrevention = publicAccessPreventionMapping[*attrs.Properties.PublicAccessPrevention]
	}

	if attrs.Properties.Versioning != nil {
		gcsAttrs.VersioningEnabled = attrs.Properties.Versioning.Enabled
	}

	if attrs.Properties.Lifecycle != nil {
		lifecycle, err := toGCSLifeCycle(*attrs.Properties.Lifecycle)
		if err != nil {
			return nil, err
		}
		gcsAttrs.Lifecycle = lifecycle
	}

	return gcsAttrs, nil
}

func toGCSBucketUpdateAttrs(attrs cloud.BucketAttrs) (*storage.BucketAttrsToUpdate, error) {
	gcsAttrs, err := toGCSBucketAttrs(attrs)

	if err != nil {
		return nil, err
	}

	return &storage.BucketAttrsToUpdate{
		VersioningEnabled:      gcsAttrs.VersioningEnabled,
		PublicAccessPrevention: gcsAttrs.PublicAccessPrevention,
		Lifecycle:              &gcsAttrs.Lifecycle,
		StorageClass:           gcsAttrs.StorageClass,
	}, nil

}

func (a *bucketAPI) GetBucket(
	ctx context.Context,
	name string,
) (*cloud.BucketAttrs, error) {
	gcsAttrs, err := a.client.Bucket(name).Attrs(ctx)

	if errors.Is(err, storage.ErrBucketNotExist) {
		return nil, cloud.ErrBucketNotFound
	}
	if err != nil {
		return nil, err
	}

	bucketAttrs, err := fromGCSBucketAttrs(*gcsAttrs)

	if err != nil {
		return nil, err
	}

	return bucketAttrs, nil
}

func (a *bucketAPI) CreateBucket(ctx context.Context, name string, attrs cloud.BucketAttrs) error {
	createAttrs, err := toGCSBucketAttrs(attrs)
	if err != nil {
		return err
	}
	if err := a.client.Bucket(name).Create(ctx, a.projectID, createAttrs); err != nil {
		return err
	}
	return nil
}

func (a *bucketAPI) UpdateBucket(ctx context.Context, name string, patch cloud.BucketPatch) (*cloud.BucketAttrs, error) {
	bucketAttrs := &cloud.BucketAttrs{}

	if patch.Lifecycle.Set {
		bucketAttrs.Properties.Lifecycle = patch.Lifecycle.Value
	}

	if patch.PublicAccessPrevention.Set {
		bucketAttrs.Properties.PublicAccessPrevention = patch.PublicAccessPrevention.Value
	}

	if patch.StorageClass.Set {
		bucketAttrs.Properties.StorageClass = patch.StorageClass.Value
	}

	if patch.Versioning.Set {
		bucketAttrs.Properties.Versioning = patch.Versioning.Value
	}

	gcsUAttrs, err := toGCSBucketUpdateAttrs(*bucketAttrs)
	if err != nil {
		return nil, err
	}

	gcsAttrs, err := a.client.Bucket(name).Update(ctx, *gcsUAttrs)
	if err != nil {
		return nil, err
	}

	if patch.Labels.Set {
		attrs, err := a.GetBucket(ctx, name)
		if err != nil {
			return nil, err
		}
		setGCSLabels(patch.Labels.Value, attrs.Properties.Labels, gcsUAttrs)
	}

	bucketAttrs, err = fromGCSBucketAttrs(*gcsAttrs)

	if err != nil {
		return nil, err
	}

	return bucketAttrs, nil
}

func (a *bucketAPI) ListObjects(
	ctx context.Context,
	bucket string,
	process func(cloud.ObjectVersion) error,
) error {
	return nil
}

func (a *bucketAPI) DeleteObject(
	ctx context.Context,
	bucket string,
	object cloud.ObjectVersion,
) error {
	return nil
}

func (a *bucketAPI) DeleteBucket(ctx context.Context, name string) error {
	return nil
}
