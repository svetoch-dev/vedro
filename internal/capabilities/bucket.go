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
