package yc

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"reflect"
	"sync"

	"github.com/gammazero/workerpool"
	vedro "github.com/svetoch-dev/vedro/api/v1alpha1"
	"github.com/svetoch-dev/vedro/internal/cloud"
	"github.com/svetoch-dev/vedro/internal/helpers"
	"github.com/svetoch-dev/vedro/internal/validation"
)

type Bucket struct {
	api cloud.BucketAPI
}

func (b *Bucket) ValidateBucketSpec(bckt vedro.Bucket, pType vedro.ProviderType) validation.ValidationResult {
	spec := bckt.Spec

	v := validation.ValidateCloudSpecificConfig(bckt.Spec.CloudSpecificConfig, pType, nil)

	if !v.Valid {
		return v
	}

	v = validation.ValidateNameImmutability(
		bckt.Spec.Name,
		bckt.Status.ExternalName,
		bckt.Name,
	)

	if !v.Valid {
		return v
	}

	v = validation.ValidateLocation(spec.Location, nil)

	if !v.Valid {
		return v
	}

	bucketName := helpers.BucketNameFromCR(bckt)
	v = validation.ValidateBucketName(bucketName, nil)
	if !v.Valid {
		return v
	}

	return validation.Valid()
}

func (b *Bucket) EnsureBucket(ctx context.Context, bckt vedro.Bucket) (*cloud.BucketAttrs, error) {
	spec := bckt.Spec

	bucketName := helpers.BucketNameFromCR(bckt)

	attrs, err := b.api.GetBucket(ctx, bucketName)

	if errors.Is(err, cloud.ErrBucketNotFound) {
		createAttrs := cloud.BucketAttrs{
			Name:     bucketName,
			Location: spec.Location,
			Properties: &vedro.BucketProperties{
				Versioning:   spec.Versioning,
				Lifecycle:    spec.Lifecycle,
				StorageClass: spec.StorageClass,
				Labels:       spec.Labels,
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

	if attrs.Location != spec.Location {
		return nil, fmt.Errorf(
			"bucket %q already exists in location %q, desired location is %q",
			bucketName,
			attrs.Location,
			spec.Location,
		)
	}

	appliedState := &cloud.BucketAttrs{
		Name:     bucketName,
		Location: spec.Location,
		Properties: &vedro.BucketProperties{
			Versioning:   spec.Versioning,
			Lifecycle:    spec.Lifecycle,
			StorageClass: spec.StorageClass,
			Labels:       spec.Labels,
		},
	}

	patch := cloud.BucketPatch{}

	if !maps.Equal(attrs.Properties.Labels, spec.Labels) {
		patch.Labels = helpers.PatchTo(spec.Labels)
	}

	if attrs.Properties.StorageClass != spec.StorageClass {
		patch.StorageClass = helpers.PatchTo(spec.StorageClass)
	}

	if !reflect.DeepEqual(
		attrs.Properties.Versioning,
		spec.Versioning,
	) {
		patch.Versioning = helpers.PatchTo(spec.Versioning)
	}

	if !reflect.DeepEqual(
		attrs.Properties.Lifecycle,
		spec.Lifecycle,
	) {
		patch.Lifecycle = helpers.PatchTo(spec.Lifecycle)
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

	// TODO make this settable via cli args or/and env var
	workers := 32
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
