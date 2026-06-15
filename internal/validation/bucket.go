package validation

import (
	vedrov1alpha1 "github.com/svetoch-dev/vedro/api/v1alpha1"
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
