package capabilities

import (
	vedro "github.com/svetoch-dev/vedro/api/v1alpha1"
	"github.com/svetoch-dev/vedro/internal/cloud"
)

func ValidateBucketAccessCapabilities(
	caps cloud.BucketAccessCapabilities,
	spec vedro.BucketAccessSpec,
) []vedro.UnsupportedFeature {
	var unsupported []vedro.UnsupportedFeature

	if spec.Access.Level == vedro.BucketAdmin && !caps.BucketAdmin {
		unsupported = append(
			unsupported,
			unsupportedFeatures["BucketAdmin"],
		)
	}

	if spec.Access.Level == vedro.ObjectAdmin && !caps.ObjectAdmin {
		unsupported = append(
			unsupported,
			unsupportedFeatures["ObjectAdmin"],
		)
	}

	if spec.Access.Level == vedro.ObjectWriter && !caps.ObjectWriter {
		unsupported = append(
			unsupported,
			unsupportedFeatures["ObjectWriter"],
		)
	}
	if spec.Access.Level == vedro.ObjectReader && !caps.ObjectReader {
		unsupported = append(
			unsupported,
			unsupportedFeatures["ObjectReader"],
		)
	}

	return unsupported
}
