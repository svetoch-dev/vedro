package validation

import (
	"regexp"
	"strings"

	vedrov1alpha1 "github.com/svetoch-dev/vedro/api/v1alpha1"
)

var (
	regionalPattern = regexp.MustCompile(`^[A-Z]+-[A-Z]+[0-9]+$`)
)

func ValidateBucketNameImmutability(bckt vedrov1alpha1.Bucket) ValidationResult {
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

func ValidateBucketLocation(location string, fn func(location string) ValidationResult) ValidationResult {
	if location == "" {
		return invalid("location is an empty string")
	}

	normalized := strings.ToUpper(location)

	// Allow normal regional names like europe-west1, us-central1.
	if !regionalPattern.MatchString(normalized) {
		return invalid("unsupported bucket location")
	}

	// per provider validation
	v := fn(location)

	if v.Valid {
		return v
	}

}
