package yc

import (
	"context"

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

	v = validation.ValidateBucketNameImmutability(bckt)

	if !v.Valid {
		return v
	}

	v = validation.ValidateBucketLocation(spec.Location, nil)

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
	p := &Provider{}
	caps := p.Capabilities().Bucket

	fake := cloud.BucketAttrs{
		Name:     bucketName,
		Location: spec.Location,
		Properties: &vedro.BucketProperties{
			Versioning:   helpers.NormalizedBucketVersioning(spec.Versioning),
			Lifecycle:    helpers.NormalizedBucketLifecycle(spec.Lifecycle, caps),
			StorageClass: spec.StorageClass,
			Labels:       spec.Labels,
		},
	}

	return &fake, nil

	// normalizedLocation := strings.ToUpper(spec.Location)

	// attrs, err := b.api.GetBucket(ctx, bucketName)

	// if errors.Is(err, cloud.ErrBucketNotFound) {
	// 	createAttrs := cloud.BucketAttrs{
	// 		Name:     bucketName,
	// 		Location: spec.Location,
	// 		Properties: &vedro.BucketProperties{
	// 			PublicAccessPrevention: helpers.NormalizedBucketPAP(spec.PublicAccessPrevention),
	// 			Versioning:             helpers.NormalizedBucketVersioning(spec.Versioning),
	// 			Lifecycle:              helpers.NormalizedBucketLifecycle(spec.Lifecycle, caps),
	// 			StorageClass:           spec.StorageClass,
	// 			Labels:                 spec.Labels,
	// 		},
	// 	}

	// 	if err := b.api.CreateBucket(ctx, bucketName, createAttrs); err != nil {
	// 		return nil, fmt.Errorf("create bucket %q: %w", bucketName, err)
	// 	}

	// 	return &createAttrs, nil
	// }

	// if err != nil {
	// 	return nil, fmt.Errorf("get bucket attrs %q: %w", bucketName, err)
	// }

	// if attrs.Location != normalizedLocation {
	// 	return nil, fmt.Errorf(
	// 		"bucket %q already exists in location %q, desired location is %q",
	// 		bucketName,
	// 		attrs.Location,
	// 		normalizedLocation,
	// 	)
	// }

	// appliedState := helpers.AppliedState(attrs.Location, bckt, caps)

	// patch := cloud.BucketPatch{}

	// if !maps.Equal(attrs.Properties.Labels, spec.Labels) {
	// 	patch.Labels = helpers.PatchTo(spec.Labels)
	// }

	// if attrs.Properties.StorageClass != spec.StorageClass {
	// 	patch.StorageClass = helpers.PatchTo(spec.StorageClass)
	// }

	// desiredVersioning := helpers.NormalizedBucketVersioning(spec.Versioning)

	// if !reflect.DeepEqual(
	// 	attrs.Properties.Versioning,
	// 	desiredVersioning,
	// ) {
	// 	patch.Versioning = helpers.PatchTo(desiredVersioning)
	// }

	// desiredPAP := helpers.NormalizedBucketPAP(spec.PublicAccessPrevention)

	// if !reflect.DeepEqual(
	// 	attrs.Properties.PublicAccessPrevention,
	// 	desiredPAP,
	// ) {
	// 	patch.PublicAccessPrevention = helpers.PatchTo(desiredPAP)
	// }

	// desiredLifecycle := helpers.NormalizedBucketLifecycle(spec.Lifecycle, caps)

	// if !reflect.DeepEqual(
	// 	attrs.Properties.Lifecycle,
	// 	desiredLifecycle,
	// ) {
	// 	patch.Lifecycle = helpers.PatchTo(desiredLifecycle)
	// }

	// if patch.HasChanges() {
	// 	updateAttrs, updateErr := b.api.UpdateBucket(ctx, bucketName, patch)
	// 	if updateErr != nil {
	// 		return nil, fmt.Errorf("update bucket %q: %w", bucketName, updateErr)
	// 	}

	// 	return updateAttrs, nil
	// }
	// return appliedState, nil
}

func (b *Bucket) DeleteBucket(ctx context.Context, bckt vedro.Bucket) error {
	return nil
}
