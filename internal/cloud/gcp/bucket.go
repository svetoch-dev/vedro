package gcp

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"

	"cloud.google.com/go/storage"
	vedrov1alpha1 "github.com/svetoch-dev/vedro/api/v1alpha1"
	"github.com/svetoch-dev/vedro/internal/cloud"
	"github.com/svetoch-dev/vedro/internal/validation"
)

var (
	invalid             = validation.Invalid
	valid               = validation.Valid
	regionalPattern     = regexp.MustCompile(`^[A-Z]+-[A-Z]+[0-9]+$`)
	dualRegionPattern   = regexp.MustCompile(`^[A-Z]+[0-9]+$`)
	bucketNamePattern   = regexp.MustCompile(`^[a-z0-9][a-z0-9._-]{1,61}[a-z0-9]$`)
	storageClassMapping = map[vedrov1alpha1.BucketStorageClass]string{
		vedrov1alpha1.BucketStorageClassStandard:         "STANDARD",
		vedrov1alpha1.BucketStorageClassInfrequentAccess: "NEARLINE",
		vedrov1alpha1.BucketStorageClassArchive:          "ARCHIVE",
	}
	publicAccessPreventionMapping = map[bool]storage.PublicAccessPrevention{
		false: storage.PublicAccessPreventionInherited,
		true:  storage.PublicAccessPreventionEnforced,
	}
)

func validateBucketLocation(location string) validation.ValidationResult {
	if location == "" {
		return invalid("location is an empty string")
	}

	normalized := strings.ToUpper(location)

	// Known multi-regions.
	switch normalized {
	case "US", "EU", "ASIA":
		return valid()
	}

	// Allow normal regional names like europe-west1, us-central1.
	if regionalPattern.MatchString(normalized) {
		return valid()
	}

	// Allow predefined dual-region IDs like NAM4, EUR4, ASIA1.
	if dualRegionPattern.MatchString(normalized) {
		return valid()
	}

	return invalid("unsupported bucket location")
}

func validateBucketName(name string) validation.ValidationResult {
	if !bucketNamePattern.MatchString(name) {
		return invalid(
			"bucket name must be 3-63 characters, contain only lowercase letters, numbers, dots, underscores, and dashes, and start/end with a letter or number",
		)
	}

	if strings.Contains(name, "..") {
		return invalid("bucket name must not contain consecutive dots")
	}

	if strings.Contains(name, ".-") || strings.Contains(name, "-.") {
		return invalid("bucket name must not contain dots next to dashes")
	}

	if strings.HasPrefix(name, "goog") || strings.Contains(name, "google") {
		return invalid("bucket name must not use reserved Google-related names")
	}

	return valid()
}

type Bucket struct {
	client    *storage.Client
	projectId string
}

func (b *Bucket) ValidateBucketSpec(spec vedrov1alpha1.BucketSpec) validation.ValidationResult {
	v := validateBucketLocation(spec.Location)

	if !v.Valid {
		return v
	}

	v = validateBucketName(spec.Name)
	if !v.Valid {
		return v
	}

	return valid()
}

func (b *Bucket) EnsureBucket(ctx context.Context, cr vedrov1alpha1.Bucket) (*cloud.BucketState, error) {
	spec := cr.Spec
	status := cr.Status
	bucket := b.client.Bucket(spec.Name)

	normalizedLocation := strings.ToUpper(spec.Location)

	storageClass, ok := storageClassMapping[spec.StorageClass]
	if !ok {
		return nil, fmt.Errorf("spec.StorageClass %s doesnt map to any bucket StorageClass", spec.StorageClass)
	}

	var publicAccessPrevention storage.PublicAccessPrevention
	if spec.PublicAccessPrevention != nil {
		publicAccessPrevention = publicAccessPreventionMapping[*spec.PublicAccessPrevention]
	}

	attrs, err := bucket.Attrs(ctx)

	if errors.Is(err, storage.ErrBucketNotExist) {
		createAttrs := storage.BucketAttrs{
			Location:     spec.Location,
			StorageClass: storageClass,
		}
		appliedByCreate := vedrov1alpha1.BucketAppliedState{
			StorageClass: spec.StorageClass,
		}

		if spec.Versioning != nil {
			createAttrs.VersioningEnabled = spec.Versioning.Enabled
			appliedByCreate.Versioning = spec.Versioning
		}

		if spec.PublicAccessPrevention != nil {
			createAttrs.PublicAccessPrevention = publicAccessPrevention
			appliedByCreate.PublicAccessPrevention = spec.PublicAccessPrevention
		}

		if err := bucket.Create(ctx, b.projectId, &createAttrs); err != nil {
			return nil, fmt.Errorf("create bucket %q: %w", spec.Name, err)
		}

		return &cloud.BucketState{
			ExternalName: spec.Name,
			Location:     normalizedLocation,
			Applied:      &appliedByCreate,
		}, nil
	}

	if err != nil {
		return nil, fmt.Errorf("get bucket attrs %q: %w", spec.Name, err)
	}

	if attrs.Location != normalizedLocation {
		return nil, fmt.Errorf(
			"bucket %q already exists in location %q, desired location is %q",
			spec.Name,
			attrs.Location,
			normalizedLocation,
		)
	}

	updateAttrs := storage.BucketAttrsToUpdate{}
	appliedByUpdate := vedrov1alpha1.BucketAppliedState{}
	if status.Applied != nil {
		appliedByUpdate = *status.Applied
	}
	needsUpdate := false

	if spec.StorageClass != "" && attrs.StorageClass != storageClass {
		updateAttrs.StorageClass = storageClass
		appliedByUpdate.StorageClass = spec.StorageClass
		needsUpdate = true
	}

	if spec.Versioning != nil && attrs.VersioningEnabled != spec.Versioning.Enabled {
		updateAttrs.VersioningEnabled = spec.Versioning.Enabled
		appliedByUpdate.Versioning = spec.Versioning
		needsUpdate = true
	}

	if spec.PublicAccessPrevention != nil && attrs.PublicAccessPrevention != publicAccessPrevention {
		updateAttrs.PublicAccessPrevention = publicAccessPrevention
		appliedByUpdate.PublicAccessPrevention = spec.PublicAccessPrevention
		needsUpdate = true
	}

	if needsUpdate {
		attrs, err = bucket.Update(ctx, updateAttrs)
		if err != nil {
			return nil, fmt.Errorf("update bucket %q: %w", spec.Name, err)
		}
	}

	return &cloud.BucketState{
		ExternalName: spec.Name,
		Location:     attrs.Location,
		Applied:      &appliedByUpdate,
	}, nil
}

func (b *Bucket) DeleteBucket(ctx context.Context, status vedrov1alpha1.BucketStatus) error {
	return nil
}
