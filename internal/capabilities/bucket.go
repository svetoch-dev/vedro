package capabilities

import (
	"fmt"

	vedro "github.com/svetoch-dev/vedro/api/v1alpha1"
	"github.com/svetoch-dev/vedro/internal/cloud"
)

var (
	unsupportedFeatures = map[string]vedro.UnsupportedFeature{
		"Versioning": vedro.UnsupportedFeature{
			Field:   "spec.Versioning",
			Message: "Versioning is not supported by this provider",
			Reason:  vedro.BucketUnsupportedVersioning,
		},
		"Lifecycle": vedro.UnsupportedFeature{
			Field:   "spec.lifecycle",
			Message: "Lifecycle is not supported by this provider",
			Reason:  vedro.BucketUnsupportedLifecycle,
		},
		"LifecycleEpiration": vedro.UnsupportedFeature{
			Field:   "spec.lifecycle.rules[%d].AgeDays",
			Message: "Object expiration is not supported by this provider",
			Reason:  vedro.BucketUnsupportedLifecycleExpiration,
		},
		"LifecycleNamed": vedro.UnsupportedFeature{
			Field:   "spec.lifecycle.rules[%d].Name",
			Message: "Named lifecycle rules are not supported by this provider",
			Reason:  vedro.BucketUnsupportedLifecycleNamed,
		},
		"Labels": vedro.UnsupportedFeature{
			Field:   "spec.Labels",
			Message: "Labels are not supported by this provider",
			Reason:  vedro.BucketUnsupportedLabels,
		},
		"PublicAccessPrevention": vedro.UnsupportedFeature{
			Field:   "spec.PublicAccessPrevention",
			Message: "PublicAccessPrevention is not supported by this provider",
			Reason:  vedro.BucketUnsupportedPublicAccessPrevention,
		},
		"StorageClassWarm": vedro.UnsupportedFeature{
			Field:   "spec.StorageClass",
			Message: fmt.Sprintf("StorageClass %s is not supported by this provider", vedro.BucketStorageClassWarm),
			Reason:  vedro.BucketUnsupportedStorageClass,
		},
		"StorageClassIce": vedro.UnsupportedFeature{
			Field:   "spec.StorageClass",
			Message: fmt.Sprintf("StorageClass %s is not supported by this provider", vedro.BucketStorageClassIce),
			Reason:  vedro.BucketUnsupportedStorageClass,
		},
		"StorageClassCold": vedro.UnsupportedFeature{
			Field:   "spec.StorageClass",
			Message: fmt.Sprintf("StorageClass %s is not supported by this provider", vedro.BucketStorageClassCold),
			Reason:  vedro.BucketUnsupportedStorageClass,
		},
	}
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
		unsupported = append(unsupported, unsupportedFeatures["Versioning"])
	}

	if spec.Lifecycle != nil && !caps.LifecycleSupported() {
		unsupported = append(unsupported, unsupportedFeatures["Lifecycle"])
	}

	if spec.Lifecycle != nil && caps.LifecycleSupported() {
		found, index := lifecycleHasExpiredRule(spec.Lifecycle.Rules)
		if found && !caps.Lifecycle.RuleExpiration {
			uf := unsupportedFeatures["LifecycleEpiration"]
			uf.Field = fmt.Sprintf(uf.Field, index)
			unsupported = append(unsupported, uf)
		}
		found, index = lifecycleHasNamedRule(spec.Lifecycle.Rules)
		if found && !caps.Lifecycle.RuleNames {
			uf := unsupportedFeatures["LifecycleNamed"]
			uf.Field = fmt.Sprintf(uf.Field, index)
			unsupported = append(unsupported, uf)
		}
	}

	if len(spec.Labels) > 0 && !caps.Labels {
		unsupported = append(unsupported, unsupportedFeatures["Labels"])
	}

	if spec.PublicAccessPrevention != nil && !caps.PublicAccessPrevention {
		unsupported = append(unsupported, unsupportedFeatures["PublicAccessPrevention"])
	}

	if spec.StorageClass == vedro.BucketStorageClassWarm && !caps.StorageClass.Warm {
		unsupported = append(unsupported, unsupportedFeatures["StorageClassWarm"])
	}

	if spec.StorageClass == vedro.BucketStorageClassIce && !caps.StorageClass.Ice {
		unsupported = append(unsupported, unsupportedFeatures["StorageClassIce"])
	}

	if spec.StorageClass == vedro.BucketStorageClassCold && !caps.StorageClass.Cold {
		unsupported = append(unsupported, unsupportedFeatures["StorageClassCold"])
	}

	return unsupported
}
