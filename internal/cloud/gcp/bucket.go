package gcp

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"reflect"
	"regexp"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/gammazero/workerpool"
	vedro "github.com/svetoch-dev/vedro/api/v1alpha1"
	"github.com/svetoch-dev/vedro/internal/cloud"
	"github.com/svetoch-dev/vedro/internal/helpers"
	"github.com/svetoch-dev/vedro/internal/validation"
)

var (
	dualRegionPattern = regexp.MustCompile(`^[A-Z]+[0-9]+$`)
)

func validateGCSLocation(location string) *validation.ValidationResult {
	normalized := strings.ToUpper(location)

	// Known multi-regions.
	switch normalized {
	case "US", "EU", "ASIA":
		v := validation.Valid()
		return &v
	}

	// Allow predefined dual-region IDs like NAM4, EUR4, ASIA1.
	if dualRegionPattern.MatchString(normalized) {
		v := validation.Valid()
		return &v
	}

	return nil
}

func validateGCSName(name string) *validation.ValidationResult {
	if strings.HasPrefix(name, "goog") || strings.Contains(name, "google") {
		v := validation.Invalid("bucket name must not use reserved Google-related names")
		return &v
	}

	return nil
}

func validateGCSCloudSpecific(cfg *vedro.BucketCloudSpecificConfig) *validation.ValidationResult {
	if cfg != nil && cfg.Gcp != nil &&
		cfg.Gcp.SoftDeletePolicy != nil {
		duration := cfg.Gcp.SoftDeletePolicy.RetentionDuration.Duration

		if duration < 0 {
			v := validation.Invalid("spec.cloudSpecificConfig.gcp.softDeletePolicy.retentionDuration cannot be negative")
			return &v
		}

		if duration != 0 && duration%(24*time.Hour) != 0 {
			v := validation.Invalid("spec.cloudSpecificConfig.gcp.softDeletePolicy.retentionDuration must be a whole number of days")
			return &v
		}
	}

	return nil
}

type Bucket struct {
	api cloud.BucketAPI
}

func (b *Bucket) ValidateBucketSpec(bckt vedro.Bucket, pType vedro.ProviderType) validation.ValidationResult {
	spec := bckt.Spec

	v := validation.ValidateCloudSpecificConfig(bckt.Spec.CloudSpecificConfig, pType, validateGCSCloudSpecific)

	if !v.Valid {
		return v
	}

	v = validation.ValidateBucketNameImmutability(bckt)

	if !v.Valid {
		return v
	}

	v = validation.ValidateBucketLocation(spec.Location, validateGCSLocation)

	if !v.Valid {
		return v
	}

	bucketName := helpers.BucketNameFromCR(bckt)
	v = validation.ValidateBucketName(bucketName, validateGCSName)
	if !v.Valid {
		return v
	}

	return validation.Valid()
}

func (b *Bucket) EnsureBucket(ctx context.Context, bckt vedro.Bucket) (*cloud.BucketAttrs, error) {
	spec := bckt.Spec

	bucketName := helpers.BucketNameFromCR(bckt)
	normalizedLocation := strings.ToUpper(spec.Location)

	p := &Provider{}
	caps := p.Capabilities().Bucket

	attrs, err := b.api.GetBucket(ctx, bucketName)

	if errors.Is(err, cloud.ErrBucketNotFound) {
		createAttrs := cloud.BucketAttrs{
			Name:     bucketName,
			Location: spec.Location,
			Properties: &vedro.BucketProperties{
				PublicAccessPrevention: helpers.NormalizedBucketPAP(spec.PublicAccessPrevention),
				Versioning:             helpers.NormalizedBucketVersioning(spec.Versioning),
				Lifecycle:              helpers.NormalizedBucketLifecycle(spec.Lifecycle, caps),
				StorageClass:           spec.StorageClass,
				Labels:                 spec.Labels,
				CloudSpecificConfig:    normalizedCloudSpecific(spec.CloudSpecificConfig),
			},
		}

		if err := b.api.CreateBucket(ctx, bucketName, createAttrs); err != nil {
			return nil, fmt.Errorf("create bucket %q: %w", bucketName, err)
		}

		return &createAttrs, nil
	}

	if err != nil {
		return nil, fmt.Errorf("get bucket attrs %q: %w", bucketName, err)
	}

	if attrs.Location != normalizedLocation {
		return nil, fmt.Errorf(
			"bucket %q already exists in location %q, desired location is %q",
			bucketName,
			attrs.Location,
			normalizedLocation,
		)
	}

	appliedState := helpers.AppliedState(attrs.Location, bckt, caps)

	patch := cloud.BucketPatch{}

	if !maps.Equal(attrs.Properties.Labels, spec.Labels) {
		patch.Labels = helpers.PatchTo(spec.Labels)
	}

	if attrs.Properties.StorageClass != spec.StorageClass {
		patch.StorageClass = helpers.PatchTo(spec.StorageClass)
	}

	desiredVersioning := helpers.NormalizedBucketVersioning(spec.Versioning)

	if !reflect.DeepEqual(
		attrs.Properties.Versioning,
		desiredVersioning,
	) {
		patch.Versioning = helpers.PatchTo(desiredVersioning)
	}

	desiredPAP := helpers.NormalizedBucketPAP(spec.PublicAccessPrevention)

	if !reflect.DeepEqual(
		attrs.Properties.PublicAccessPrevention,
		desiredPAP,
	) {
		patch.PublicAccessPrevention = helpers.PatchTo(desiredPAP)
	}

	desiredLifecycle := helpers.NormalizedBucketLifecycle(spec.Lifecycle, caps)

	if !reflect.DeepEqual(
		attrs.Properties.Lifecycle,
		desiredLifecycle,
	) {
		patch.Lifecycle = helpers.PatchTo(desiredLifecycle)
	}

	desiredCloudSpecificConfig := normalizedCloudSpecific(spec.CloudSpecificConfig)

	if !reflect.DeepEqual(
		attrs.Properties.CloudSpecificConfig,
		desiredCloudSpecificConfig,
	) {
		patch.CloudSpecificConfig = helpers.PatchTo(spec.CloudSpecificConfig)
	}

	if patch.HasChanges() {
		updateAttrs, updateErr := b.api.UpdateBucket(ctx, bucketName, patch)
		if updateErr != nil {
			return nil, fmt.Errorf("update bucket %q: %w", bucketName, updateErr)
		}

		return updateAttrs, nil
	}
	return appliedState, nil
}

func (b *Bucket) DeleteBucket(ctx context.Context, bckt vedro.Bucket) error {
	bucketName := helpers.BucketNameFromCR(bckt)

	if bckt.Spec.DeletionPolicy != vedro.DeletionPolicyDelete {
		return nil
	}

	// err that we will return if object deletion fails
	var deleteObjectError error
	// error mutex for syncing concurrent changes to error var
	var errM sync.Mutex

	// New pool with workers equal number of CPU
	workers := runtime.NumCPU() - 1
	if workers < 1 {
		workers = 1
	}
	wp := workerpool.New(workers)

	// Semaphore channel allowing up to 2000 uncompleted deletion tasks in the queue
	sem := make(chan struct{}, 2000)

	err := b.api.ProcessObjects(ctx, bucketName, func(object cloud.ObjectVersion) error {
		// queue task
		sem <- struct{}{}

		wp.Submit(func() {
			// dequeue task. Defer only accepts callable so wrap it in func
			defer func() { <-sem }()
			err := b.api.DeleteObject(ctx, bucketName, object)
			if err != nil {
				errM.Lock()
				defer errM.Unlock()
				if errors.Is(err, cloud.ErrBucketObjectNotFound) {
					return
				}
				if deleteObjectError == nil {
					deleteObjectError = err
				}
			}
		})

		return nil
	})

	wp.StopWait()

	if err != nil {
		return err
	}

	if deleteObjectError != nil {
		return fmt.Errorf("could not delete bucket because object deletion failed: %w", deleteObjectError)
	}

	err = b.api.DeleteBucket(ctx, bucketName)
	if errors.Is(err, cloud.ErrBucketNotFound) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("could not delete bucket because of error: %w", err)
	}

	return nil
}
