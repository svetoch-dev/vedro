package gcp

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"time"

	"cloud.google.com/go/storage"
	vedro "github.com/svetoch-dev/vedro/api/v1alpha1"
	"github.com/svetoch-dev/vedro/internal/cloud"
	"github.com/svetoch-dev/vedro/internal/helpers"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/iterator"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

var (
	gcsStorageClassMapping = map[string]vedro.BucketStorageClass{
		"STANDARD": vedro.BucketStorageClassStandard,
		"NEARLINE": vedro.BucketStorageClassWarm,
		"COLDLINE": vedro.BucketStorageClassCold,
		"ARCHIVE":  vedro.BucketStorageClassIce,
	}
	storageClassMapping = map[vedro.BucketStorageClass]string{
		vedro.BucketStorageClassStandard: "STANDARD",
		vedro.BucketStorageClassWarm:     "NEARLINE",
		vedro.BucketStorageClassCold:     "COLDLINE",
		vedro.BucketStorageClassIce:      "ARCHIVE",
	}
	gcsPublicAccessPreventionMapping = map[storage.PublicAccessPrevention]*bool{
		storage.PublicAccessPreventionInherited: helpers.Ptr(false),
		storage.PublicAccessPreventionEnforced:  helpers.Ptr(true),
		storage.PublicAccessPreventionUnknown:   nil,
	}
	gcsLifeCycleActionMapping = map[string]vedro.BucketLifecycleAction{
		storage.DeleteAction: vedro.BucketLifecycleActionDelete,
	}
	lifeCycleActionMapping = map[vedro.BucketLifecycleAction]string{
		vedro.BucketLifecycleActionDelete: storage.DeleteAction,
	}
	defaultSoftDeleteDuration = 7 * 24 * time.Hour
	defaultSoftDelete         = &vedro.SoftDeletePolicy{
		RetentionDuration: v1.Duration{
			Duration: defaultSoftDeleteDuration,
		},
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

func versioningMapping(v *vedro.BucketVersioning) bool {
	if v == nil {
		return false
	}

	if v.Enabled {
		return true
	}

	return false
}

func normalizedCloudSpecific(cfg *vedro.BucketCloudSpecificConfig) *vedro.BucketCloudSpecificConfig {
	if cfg == nil || cfg.Gcp == nil || cfg.Gcp.SoftDeletePolicy == nil {
		return &vedro.BucketCloudSpecificConfig{
			Gcp: &vedro.BucketGcpConfig{
				SoftDeletePolicy: defaultSoftDelete,
			},
		}
	}

	return cfg
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

	if maps.Equal(desiredLabels, actualLabels) {
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

func toGCSLifeCycle(lifeCycle *vedro.BucketLifecycle) (storage.Lifecycle, error) {
	gcsLifeCycle := storage.Lifecycle{}

	if lifeCycle == nil {
		return gcsLifeCycle, nil
	}

	for index, rule := range lifeCycle.Rules {
		if !rule.Enabled {
			continue
		}
		actionType, ok := lifeCycleActionMapping[rule.Action]
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

		if rule.AgeDays != nil && *rule.AgeDays > 0 {
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

func fromGCSLifeCycle(lifecycle storage.Lifecycle) vedro.BucketLifecycle {
	var bucketLifeCycle vedro.BucketLifecycle

	for _, rule := range lifecycle.Rules {
		if rule.Condition.AgeInDays > 0 && rule.Action.Type == storage.DeleteAction {
			bucketLifeCycle.Rules = append(bucketLifeCycle.Rules, vedro.BucketLifecycleRule{
				AgeDays: helpers.Ptr(rule.Condition.AgeInDays),
				Action:  gcsLifeCycleActionMapping[rule.Action.Type],
				Enabled: true,
			})
		}
	}

	return bucketLifeCycle

}

func toCloudSpecific(sdp *storage.SoftDeletePolicy) *vedro.BucketCloudSpecificConfig {
	gcpConfig := &vedro.BucketGcpConfig{}

	if sdp == nil {
		gcpConfig.SoftDeletePolicy = defaultSoftDelete
	} else {
		gcpConfig.SoftDeletePolicy = &vedro.SoftDeletePolicy{
			RetentionDuration: v1.Duration{
				Duration: sdp.RetentionDuration,
			},
		}
	}

	return &vedro.BucketCloudSpecificConfig{
		Gcp: gcpConfig,
	}
}

func fromCloudSpecific(cfg *vedro.BucketCloudSpecificConfig) *storage.SoftDeletePolicy {
	sdp := &storage.SoftDeletePolicy{
		RetentionDuration: defaultSoftDeleteDuration,
	}

	if cfg == nil || cfg.Gcp == nil {
		return sdp
	}

	cfgSdp := cfg.Gcp.SoftDeletePolicy

	if cfgSdp != nil {
		sdp.RetentionDuration = cfgSdp.RetentionDuration.Duration
	}

	return sdp
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

	lifeCycle := fromGCSLifeCycle(attrs.Lifecycle)

	return &cloud.BucketAttrs{
		Name:     attrs.Name,
		Location: attrs.Location,
		Properties: &vedro.BucketProperties{
			PublicAccessPrevention: pap,
			Versioning: &vedro.BucketVersioning{
				Enabled: attrs.VersioningEnabled,
			},
			StorageClass:        sc,
			Labels:              attrs.Labels,
			Lifecycle:           &lifeCycle,
			CloudSpecificConfig: toCloudSpecific(attrs.SoftDeletePolicy),
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

	sdp := fromCloudSpecific(attrs.Properties.CloudSpecificConfig)

	gcsAttrs.SoftDeletePolicy = sdp

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

	if patch.CloudSpecificConfig.Set {
		sdp := fromCloudSpecific(patch.CloudSpecificConfig.Value)
		update.SoftDeletePolicy = sdp
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
	log.FromContext(ctx).V(1).Info("Updating bucket")

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
	err := bh.Object(object.Name).Generation(object.Version).Delete(ctx)
	if isGoogleAPINotFound(err) {
		return cloud.ErrBucketObjectNotFound
	}
	return err
}

func (a *bucketAPI) DeleteBucket(ctx context.Context, name string) error {
	bh := a.client.Bucket(name)
	err := bh.Delete(ctx)
	if isGoogleAPINotFound(err) {
		return cloud.ErrBucketNotFound
	}
	return err
}

func isGoogleAPINotFound(err error) bool {
	var gErr *googleapi.Error
	return errors.As(err, &gErr) && gErr.Code == 404
}
