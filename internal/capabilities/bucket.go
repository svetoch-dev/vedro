package capabilities

import (
	"fmt"

	vedrov1alpha1 "github.com/svetoch-dev/vedro/api/v1alpha1"
	"github.com/svetoch-dev/vedro/internal/cloud"
)

func lifecycleHasExpiredRule(rules []vedrov1alpha1.BucketLifecycleRule) (bool, int) {
	for i, rule := range rules {
		if rule.AgeDays != nil {
			return true, i
		}
	}
	return false, 0
}

func lifecycleHasNamedRule(rules []vedrov1alpha1.BucketLifecycleRule) (bool, int) {
	for i, rule := range rules {
		if rule.Name != nil {
			return true, i
		}
	}
	return false, 0
}

func ValidateBucketCapabilities(
	caps cloud.BucketCapabilities,
	spec vedrov1alpha1.BucketSpec,
) []vedrov1alpha1.UnsupportedFeature {

	var unsupported []vedrov1alpha1.UnsupportedFeature

	if spec.Versioning != nil && !caps.Versioning {
		unsupported = append(unsupported, vedrov1alpha1.UnsupportedFeature{
			Field:   "spec.Versioning",
			Message: "Versioning is not supported by this provider",
			Reason:  vedrov1alpha1.BucketUnsupportedVersioning,
		})
	}

	if spec.Lifecycle != nil && !caps.Lifecycle.Supported {
		unsupported = append(unsupported, vedrov1alpha1.UnsupportedFeature{
			Field:   "spec.lifecycle",
			Message: "Lifecycle is not supported by this provider",
			Reason:  vedrov1alpha1.BucketUnsupportedLifecycle,
		})
	}

	if spec.Lifecycle != nil && caps.Lifecycle.Supported {
		found, index := lifecycleHasExpiredRule(spec.Lifecycle.Rules)
		if found && !caps.Lifecycle.RuleExpiration {
			unsupported = append(unsupported, vedrov1alpha1.UnsupportedFeature{
				Field:   fmt.Sprintf("spec.lifecycle.rules[%d].AgeDays", index),
				Message: "Object expiration is not supported by this provider",
				Reason:  vedrov1alpha1.BucketUnsupportedLifecycleExpiration,
			})
		}
		found, index = lifecycleHasNamedRule(spec.Lifecycle.Rules)
		if found && !caps.Lifecycle.RuleNames {
			unsupported = append(unsupported, vedrov1alpha1.UnsupportedFeature{
				Field:   fmt.Sprintf("spec.lifecycle.rules[%d].Name", index),
				Message: "Named lifecycle rules are not supported by this provider",
				Reason:  vedrov1alpha1.BucketUnsupportedLifecycleNamed,
			})
		}
	}

	if len(spec.Labels) > 0 && !caps.Labels {
		unsupported = append(unsupported, vedrov1alpha1.UnsupportedFeature{
			Field:   "spec.Labels",
			Message: "Labels are not supported by this provider",
			Reason:  vedrov1alpha1.BucketUnsupportedLabels,
		})
	}

	if spec.PublicAccessPrevention != nil && !caps.PublicAccessPrevention {
		unsupported = append(unsupported, vedrov1alpha1.UnsupportedFeature{
			Field:   "spec.PublicAccessPrevention",
			Message: "PublicAccessPrevention is not supported by this provider",
			Reason:  vedrov1alpha1.BucketUnsupportedPublicAccessPrevention,
		})
	}

	if spec.StorageClass == vedrov1alpha1.BucketStorageClassInfrequentAccess && !caps.StorageClass.InfrequentAccess {
		unsupported = append(unsupported, vedrov1alpha1.UnsupportedFeature{
			Field:   "spec.StorageClass",
			Message: fmt.Sprintf("StorageClass %s is not supported by this provider", vedrov1alpha1.BucketStorageClassInfrequentAccess),
			Reason:  vedrov1alpha1.BucketUnsupportedStorageClass,
		})
	}

	if spec.StorageClass == vedrov1alpha1.BucketStorageClassArchive && !caps.StorageClass.Archive {
		unsupported = append(unsupported, vedrov1alpha1.UnsupportedFeature{
			Field:   "spec.StorageClass",
			Message: fmt.Sprintf("StorageClass %s is not supported by this provider", vedrov1alpha1.BucketStorageClassArchive),
			Reason:  vedrov1alpha1.BucketUnsupportedStorageClass,
		})
	}

	return unsupported
}
