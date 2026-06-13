package capabilities

import (
	"fmt"

	vedrov1alpha1 "github.com/svetoch-dev/vedro/api/v1alpha1"
	"github.com/svetoch-dev/vedro/internal/cloud"
)

func lifecycleHasExpiredRule(rules []vedrov1alpha1.BucketLifecycleRule) (bool, int) {
	for i, rule := range rules {
		if rule.Enabled && rule.AgeDays != nil && rule.Action == vedrov1alpha1.BucketLifecycleActionDelete {
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

	if spec.Lifecycle != nil {
		found, index := lifecycleHasExpiredRule(spec.Lifecycle.Rules)
		if found && !caps.LifecycleExpiration {
			unsupported = append(unsupported, vedrov1alpha1.UnsupportedFeature{
				Field:   fmt.Sprintf("spec.lifecycle.rules[%d].AgeDays", index),
				Message: "Object expiration is not supported by this provider",
				Reason:  vedrov1alpha1.BucketUnsupportedLifecycleExpiration,
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

	if spec.PublicAccess != nil && !caps.PublicAccess {
		unsupported = append(unsupported, vedrov1alpha1.UnsupportedFeature{
			Field:   "spec.PublicAccess",
			Message: "PublicAccess is not supported by this provider",
			Reason:  vedrov1alpha1.BucketUnsupportedPublicAccess,
		})
	}

	if spec.StorageClass == vedrov1alpha1.BucketStorageClassInfrequentAccess && !caps.StorageClassInfrequentAccess {
		unsupported = append(unsupported, vedrov1alpha1.UnsupportedFeature{
			Field:   "spec.StorageClass",
			Message: fmt.Sprintf("StorageClass %s is not supported by this provider", vedrov1alpha1.BucketStorageClassInfrequentAccess),
			Reason:  vedrov1alpha1.BucketUnsupportedStorageClass,
		})
	}

	if spec.StorageClass == vedrov1alpha1.BucketStorageClassArchive && !caps.StorageClassArchive {
		unsupported = append(unsupported, vedrov1alpha1.UnsupportedFeature{
			Field:   "spec.StorageClass",
			Message: fmt.Sprintf("StorageClass %s is not supported by this provider", vedrov1alpha1.BucketStorageClassArchive),
			Reason:  vedrov1alpha1.BucketUnsupportedStorageClass,
		})
	}

	return unsupported
}
