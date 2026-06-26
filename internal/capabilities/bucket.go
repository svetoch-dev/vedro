package capabilities

import (
	"fmt"

	vedro "github.com/svetoch-dev/vedro/api/v1alpha1"
	"github.com/svetoch-dev/vedro/internal/cloud"
)

func lifecycleHasExpiredRule(rules []vedro.BucketLifecycleRule) (bool, int) {
	for i, rule := range rules {
		if rule.AgeDays != nil {
			return true, i
		}
	}
	return false, 0
}

func lifecycleHasNamedRule(rules []vedro.BucketLifecycleRule) (bool, int) {
	for i, rule := range rules {
		if rule.Name != nil {
			return true, i
		}
	}
	return false, 0
}

func ValidateBucketCapabilities(
	caps cloud.BucketCapabilities,
	spec vedro.BucketSpec,
) []vedro.UnsupportedFeature {

	var unsupported []vedro.UnsupportedFeature

	if spec.Versioning != nil && !caps.Versioning {
		unsupported = append(unsupported, vedro.UnsupportedFeature{
			Field:   "spec.Versioning",
			Message: "Versioning is not supported by this provider",
			Reason:  vedro.BucketUnsupportedVersioning,
		})
	}

	if spec.Lifecycle != nil && !caps.LifecycleSupported() {
		unsupported = append(unsupported, vedro.UnsupportedFeature{
			Field:   "spec.lifecycle",
			Message: "Lifecycle is not supported by this provider",
			Reason:  vedro.BucketUnsupportedLifecycle,
		})
	}

	if spec.Lifecycle != nil && caps.LifecycleSupported() {
		found, index := lifecycleHasExpiredRule(spec.Lifecycle.Rules)
		if found && !caps.Lifecycle.RuleExpiration {
			unsupported = append(unsupported, vedro.UnsupportedFeature{
				Field:   fmt.Sprintf("spec.lifecycle.rules[%d].AgeDays", index),
				Message: "Object expiration is not supported by this provider",
				Reason:  vedro.BucketUnsupportedLifecycleExpiration,
			})
		}
		found, index = lifecycleHasNamedRule(spec.Lifecycle.Rules)
		if found && !caps.Lifecycle.RuleNames {
			unsupported = append(unsupported, vedro.UnsupportedFeature{
				Field:   fmt.Sprintf("spec.lifecycle.rules[%d].Name", index),
				Message: "Named lifecycle rules are not supported by this provider",
				Reason:  vedro.BucketUnsupportedLifecycleNamed,
			})
		}
	}

	if len(spec.Labels) > 0 && !caps.Labels {
		unsupported = append(unsupported, vedro.UnsupportedFeature{
			Field:   "spec.Labels",
			Message: "Labels are not supported by this provider",
			Reason:  vedro.BucketUnsupportedLabels,
		})
	}

	if spec.PublicAccessPrevention != nil && !caps.PublicAccessPrevention {
		unsupported = append(unsupported, vedro.UnsupportedFeature{
			Field:   "spec.PublicAccessPrevention",
			Message: "PublicAccessPrevention is not supported by this provider",
			Reason:  vedro.BucketUnsupportedPublicAccessPrevention,
		})
	}

	if spec.StorageClass == vedro.BucketStorageClassInfrequentAccess && !caps.StorageClass.InfrequentAccess {
		unsupported = append(unsupported, vedro.UnsupportedFeature{
			Field:   "spec.StorageClass",
			Message: fmt.Sprintf("StorageClass %s is not supported by this provider", vedro.BucketStorageClassInfrequentAccess),
			Reason:  vedro.BucketUnsupportedStorageClass,
		})
	}

	if spec.StorageClass == vedro.BucketStorageClassArchive && !caps.StorageClass.Archive {
		unsupported = append(unsupported, vedro.UnsupportedFeature{
			Field:   "spec.StorageClass",
			Message: fmt.Sprintf("StorageClass %s is not supported by this provider", vedro.BucketStorageClassArchive),
			Reason:  vedro.BucketUnsupportedStorageClass,
		})
	}

	return unsupported
}
