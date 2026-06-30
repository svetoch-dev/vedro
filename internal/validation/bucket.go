package validation

import (
	"regexp"
	"strings"

	vedro "github.com/svetoch-dev/vedro/api/v1alpha1"
)

var (
	// Matches regional names like europe-west1, us-central1, us-east-1, cn-hongkong
	regionalPattern   = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)+$`)
	bucketNamePattern = regexp.MustCompile(`^[a-z0-9][a-z0-9._-]{1,61}[a-z0-9]$`)
)

func ValidateCloudSpecificConfig(cfg *vedro.BucketCloudSpecificConfig, pType vedro.ProviderType, validateCloud func(cfg *vedro.BucketCloudSpecificConfig) *ValidationResult) ValidationResult {
	if cfg == nil {
		return Valid()
	}

	if validateCloud != nil {
		v := validateCloud(cfg)

		if v != nil {
			return *v
		}
	}

	switch pType {
	case vedro.ProviderTypeGCP:
		if cfg.Yc != nil {
			return Invalid("spec.cloudSpecificConfig.yc can only be used with provider type yc")
		}
		return Valid()

	case vedro.ProviderTypeYandexCloud:
		if cfg.Gcp != nil {
			return Invalid("spec.cloudSpecificConfig.gcp can only be used with provider type gcp")
		}
		return Valid()

	default:
		return Invalid("spec.cloudSpecificConfig contains provider-specific settings unsupported by provider type")
	}
}

func ValidateBucketNameImmutability(bckt vedro.Bucket) ValidationResult {
	spec := bckt.Spec
	status := bckt.Status

	if spec.Name != "" && status.ExternalName != "" && status.ExternalName != spec.Name {
		return Invalid("spec.name cannot be changed after bucket creation")
	}

	if spec.Name == "" && status.ExternalName != "" && status.ExternalName != bckt.Name {
		return Invalid("metadata.name cannot be used as the bucket name source if spec.Name was used and bucket is created")
	}

	return Valid()
}

func ValidateBucketLocation(location string, fn func(location string) *ValidationResult) ValidationResult {
	if location == "" {
		return Invalid("location is an empty string")
	}

	// per provider validation
	v := fn(location)

	if v != nil {
		return *v
	}

	if !regionalPattern.MatchString(location) {
		return Invalid("unsupported bucket location")
	}

	return Valid()
}

func ValidateBucketName(name string, fn func(name string) *ValidationResult) ValidationResult {
	if name == "" {
		return Invalid("name is an empty string")
	}

	v := fn(name)

	if v != nil {
		return *v
	}

	if !bucketNamePattern.MatchString(name) {
		return Invalid(
			"bucket name must be 3-63 characters, contain only lowercase letters, numbers, dots, underscores, and dashes, and start/end with a letter or number",
		)
	}

	if strings.Contains(name, "..") {
		return Invalid("bucket name must not contain consecutive dots")
	}

	if strings.Contains(name, ".-") || strings.Contains(name, "-.") {
		return Invalid("bucket name must not contain dots next to dashes")
	}

	return Valid()
}
