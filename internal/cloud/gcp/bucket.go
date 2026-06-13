package gcp

import (
	"context"
	"regexp"
	"strings"

	"cloud.google.com/go/storage"
	vedrov1alpha1 "github.com/svetoch-dev/vedro/api/v1alpha1"
	"github.com/svetoch-dev/vedro/internal/cloud"
	"github.com/svetoch-dev/vedro/internal/validation"
)

var (
	invalid func(string) validation.ValidationResult = validation.Invalid
	valid   func() validation.ValidationResult       = validation.Valid
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
	regionalPattern := regexp.MustCompile(`^[a-z]+-[a-z]+[0-9]+$`)
	if regionalPattern.MatchString(location) {
		return valid()
	}

	// Allow predefined dual-region IDs like NAM4, EUR4, ASIA1.
	dualRegionPattern := regexp.MustCompile(`^[A-Z]+[0-9]+$`)
	if dualRegionPattern.MatchString(normalized) {
		return valid()
	}

	return invalid("unsupported bucket location")
}

func validateBucketName(name string) validation.ValidationResult {
	bucketNamePattern := regexp.MustCompile(`^[a-z0-9][a-z0-9._-]{1,61}[a-z0-9]$`)
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
	client *storage.Client
}

func (p *Bucket) ValidateBucketSpec(spec vedrov1alpha1.BucketSpec) validation.ValidationResult {
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

func (p *Bucket) EnsureBucket(ctx context.Context, spec vedrov1alpha1.BucketSpec) (*cloud.BucketState, error) {
	return nil, nil
}

func (p *Bucket) DeleteBucket(ctx context.Context, status vedrov1alpha1.BucketStatus) error {
	return nil
}
