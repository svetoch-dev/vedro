package gcp

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"regexp"
	"strings"

	"cloud.google.com/go/storage"
	vedrov1alpha1 "github.com/svetoch-dev/vedro/api/v1alpha1"
	"github.com/svetoch-dev/vedro/internal/cloud"
	"github.com/svetoch-dev/vedro/internal/helpers"
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

// Abstraction for tests
type storageClient interface {
	Bucket(name string) bucketHandle
}

type bucketHandle interface {
	Attrs(ctx context.Context) (*storage.BucketAttrs, error)
	Create(ctx context.Context, projectID string, attrs *storage.BucketAttrs) error
	Update(ctx context.Context, uattrs storage.BucketAttrsToUpdate) (*storage.BucketAttrs, error)
}

type storageClientAdapter struct {
	client *storage.Client
}

func (a *storageClientAdapter) Bucket(name string) bucketHandle {
	return a.client.Bucket(name)
}

type Bucket struct {
	client    storageClient
	projectId string
}

func (b *Bucket) ValidateBucketSpec(bckt vedrov1alpha1.Bucket) validation.ValidationResult {
	spec := bckt.Spec

	v := validation.ValidateBucketNameImmutability(bckt)

	if !v.Valid {
		return v
	}

	v = validateBucketLocation(spec.Location)

	if !v.Valid {
		return v
	}

	bucketName := helpers.BucketNameFromCR(bckt)
	v = validateBucketName(bucketName)
	if !v.Valid {
		return v
	}

	return valid()
}

func (b *Bucket) EnsureBucket(ctx context.Context, bckt vedrov1alpha1.Bucket) (*cloud.BucketState, error) {
	spec := bckt.Spec
	status := bckt.Status

	bucketName := helpers.BucketNameFromCR(bckt)
	normalizedLocation := strings.ToUpper(spec.Location)

	bucket := b.client.Bucket(bucketName)

	storageClass, ok := storageClassMapping[spec.StorageClass]
	if !ok {
		return nil, fmt.Errorf("spec.StorageClass %s doesnt map to any bucket StorageClass", spec.StorageClass)
	}

	var publicAccessPrevention storage.PublicAccessPrevention
	if spec.PublicAccessPrevention != nil {
		publicAccessPrevention = publicAccessPreventionMapping[*spec.PublicAccessPrevention]
	} else {
		publicAccessPrevention = publicAccessPreventionMapping[false]
	}

	var versioning bool
	if spec.Versioning != nil {
		versioning = spec.Versioning.Enabled
	} else {
		versioning = false
	}

	applied := vedrov1alpha1.BucketAppliedState{}
	if status.Applied != nil {
		applied = *status.Applied
	}

	attrs, err := bucket.Attrs(ctx)

	if errors.Is(err, storage.ErrBucketNotExist) {
		createAttrs := storage.BucketAttrs{
			Location:     spec.Location,
			StorageClass: storageClass,
		}

		applied.StorageClass = spec.StorageClass

		createAttrs.Labels = spec.Labels
		applied.Labels = spec.Labels

		createAttrs.VersioningEnabled = versioning
		applied.Versioning = spec.Versioning

		createAttrs.PublicAccessPrevention = publicAccessPrevention
		applied.PublicAccessPrevention = spec.PublicAccessPrevention

		if err := bucket.Create(ctx, b.projectId, &createAttrs); err != nil {
			return nil, fmt.Errorf("create bucket %q: %w", bucketName, err)
		}

		return &cloud.BucketState{
			ExternalName: bucketName,
			Location:     normalizedLocation,
			Applied:      &applied,
		}, nil
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

	updateAttrs := storage.BucketAttrsToUpdate{}
	needsUpdate := false

	if !maps.Equal(applied.Labels, spec.Labels) {
		setGCSLabels(spec.Labels, applied.Labels, &updateAttrs)
		applied.Labels = spec.Labels
		needsUpdate = true
	}

	if attrs.StorageClass != storageClass {
		updateAttrs.StorageClass = storageClass
		applied.StorageClass = spec.StorageClass
		needsUpdate = true
	}

	if attrs.VersioningEnabled != versioning {
		updateAttrs.VersioningEnabled = versioning
		applied.Versioning = spec.Versioning
		needsUpdate = true
	}

	if attrs.PublicAccessPrevention != publicAccessPrevention {
		updateAttrs.PublicAccessPrevention = publicAccessPrevention
		applied.PublicAccessPrevention = spec.PublicAccessPrevention
		needsUpdate = true
	}

	if needsUpdate {
		attrs, err = bucket.Update(ctx, updateAttrs)
		if err != nil {
			return nil, fmt.Errorf("update bucket %q: %w", bucketName, err)
		}
	}

	return &cloud.BucketState{
		ExternalName: bucketName,
		Location:     attrs.Location,
		Applied:      &applied,
	}, nil
}

func (b *Bucket) DeleteBucket(ctx context.Context, status vedrov1alpha1.BucketStatus) error {
	return nil
}
