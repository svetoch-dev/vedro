package gcp

import (
	"context"
	"errors"
	"fmt"

	"cloud.google.com/go/storage"
	vedrov1alpha1 "github.com/svetoch-dev/vedro/api/v1alpha1"
	"github.com/svetoch-dev/vedro/internal/cloud"
	"github.com/svetoch-dev/vedro/internal/helpers"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/iterator"
)

var (
	gcsStorageClassMapping = map[string]vedrov1alpha1.BucketStorageClass{
		"STANDARD": vedrov1alpha1.BucketStorageClassStandard,
		"NEARLINE": vedrov1alpha1.BucketStorageClassInfrequentAccess,
		"COLDLINE": vedrov1alpha1.BucketStorageClassArchive,
		"ARCHIVE":  vedrov1alpha1.BucketStorageClassArchive,
	}
	storageClassMapping = map[vedrov1alpha1.BucketStorageClass]string{
		vedrov1alpha1.BucketStorageClassStandard:         "STANDARD",
		vedrov1alpha1.BucketStorageClassInfrequentAccess: "NEARLINE",
		vedrov1alpha1.BucketStorageClassArchive:          "ARCHIVE",
	}
	gcsPublicAccessPreventionMapping = map[storage.PublicAccessPrevention]*bool{
		storage.PublicAccessPreventionInherited: helpers.Ptr(false),
		storage.PublicAccessPreventionEnforced:  helpers.Ptr(true),
		storage.PublicAccessPreventionUnknown:   nil,
	}
	gcsLifeCycleMapping = map[string]vedrov1alpha1.BucketLifecycleAction{
		storage.DeleteAction: vedrov1alpha1.BucketLifecycleActionDelete,
	}
	lifeCycleMapping = map[vedrov1alpha1.BucketLifecycleAction]string{
		vedrov1alpha1.BucketLifecycleActionDelete: storage.DeleteAction,
	}
)

func publicAccessPreventionMapping(v *bool) storage.PublicAccessPrevention {
	if v == nil {
		return storage.PublicAccessPreventionInherited
	}

	if *v {
		return storage.PublicAccessPreventionEnforced
	}

	return storage.PublicAccessPreventionInherited
}

func versioningMapping(v *vedrov1alpha1.BucketVersioning) bool {
	if v == nil {
		return false
	}

	if v.Enabled {
		return true
	}

	return false
}

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

func toGCSLifeCycle(lifeCycle *vedrov1alpha1.BucketLifecycle) (storage.Lifecycle, error) {
	gcsLifeCycle := storage.Lifecycle{}

	if lifeCycle == nil {
		return gcsLifeCycle, nil
	}

	for index, rule := range lifeCycle.Rules {
		if !rule.Enabled {
			continue
		}

		actionType, ok := lifeCycleMapping[rule.Action]
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

	sc, ok := storageClassMapping[attrs.Properties.StorageClass]
	if !ok {
		return nil, fmt.Errorf("gcs StorageClass %s doesnt map to any bucket StorageClass", attrs.Properties.StorageClass)
	}

	gcsAttrs.StorageClass = sc
	gcsAttrs.Labels = attrs.Properties.Labels

	gcsAttrs.PublicAccessPrevention = publicAccessPreventionMapping(attrs.Properties.PublicAccessPrevention)

	gcsAttrs.VersioningEnabled = versioningMapping(attrs.Properties.Versioning)

	lifecycle, err := toGCSLifeCycle(attrs.Properties.Lifecycle)
	if err != nil {
		return nil, err
	}
	gcsAttrs.Lifecycle = lifecycle

	return gcsAttrs, nil
}

func patchGCSBucketAttrs(patch cloud.BucketPatch, currentAttrs *cloud.BucketAttrs) (*storage.BucketAttrsToUpdate, error) {
	update := &storage.BucketAttrsToUpdate{}

	if patch.StorageClass.Set {
		storageClass, ok := storageClassMapping[patch.StorageClass.Value]
		if !ok {
			return nil, fmt.Errorf(
				"bucket storage class %q does not map to GCS",
				patch.StorageClass.Value,
			)
		}
		update.StorageClass = storageClass
	}

	if patch.Versioning.Set {
		update.VersioningEnabled = versioningMapping(patch.Versioning.Value)
	}

	if patch.PublicAccessPrevention.Set {
		update.PublicAccessPrevention = publicAccessPreventionMapping(patch.PublicAccessPrevention.Value)
	}

	if patch.Lifecycle.Set {
		lifecycle, err := toGCSLifeCycle(patch.Lifecycle.Value)
		if err != nil {
			return nil, err
		}
		update.Lifecycle = &lifecycle
	}

	if patch.Labels.Set {
		setGCSLabels(
			patch.Labels.Value,
			currentAttrs.Properties.Labels,
			update,
		)
	}

	return update, nil
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

	return fromGCSBucketAttrs(*gcsAttrs)
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
	currentAttrs, err := a.GetBucket(ctx, name)
	if err != nil {
		return nil, fmt.Errorf("get bucket before patch: %w", err)
	}

	gcsUAttrs, err := patchGCSBucketAttrs(patch, currentAttrs)
	if err != nil {
		return nil, err
	}

	gcsAttrs, err := a.client.Bucket(name).Update(ctx, *gcsUAttrs)
	if err != nil {
		return nil, err
	}

	return fromGCSBucketAttrs(*gcsAttrs)
}

func (a *bucketAPI) ProcessObjects(
	ctx context.Context,
	bucket string,
	process func(cloud.ObjectVersion) error,
) error {
	it := a.client.Bucket(bucket).Objects(ctx, &storage.Query{
		Versions: true,
	})

	for {
		attrs, err := it.Next()
		if errors.Is(err, iterator.Done) {
			return nil
		}
		if err != nil {
			return fmt.Errorf("list objects: %w", err)
		}

		if err := process(cloud.ObjectVersion{
			Name:    attrs.Name,
			Version: attrs.Generation,
		}); err != nil {
			return err
		}
	}
}

func (a *bucketAPI) DeleteObject(
	ctx context.Context,
	bucket string,
	object cloud.ObjectVersion,
) error {
	bh := a.client.Bucket(bucket)
	return bh.Object(object.Name).Generation(object.Version).Delete(ctx)
}

func (a *bucketAPI) DeleteBucket(ctx context.Context, name string) error {
	bh := a.client.Bucket(name)
	err := bh.Delete(ctx)
	if err != nil {
		var gErr *googleapi.Error
		if errors.As(err, &gErr) && gErr.Code == 404 {
			return nil
		}

		return fmt.Errorf("could not delete bucket because of error: %w", err)
	}

	return nil
}
