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
			vedro.UnsupportedFeature{
				Field:   "access.level",
				Message: "BucketAdmin acces is unsupported by this provider",
				Reason:  vedro.BucketAccessUnsupportedBucketAdmin,
			},
		)
	}

	if spec.Access.Level == vedro.ObjectAdmin && !caps.ObjectAdmin {
		unsupported = append(
			unsupported,
			vedro.UnsupportedFeature{
				Field:   "access.level",
				Message: "ObjectAdmin acces is unsupported by this provider",
				Reason:  vedro.BucketAccessUnsupportedObjectAdmin,
			},
		)
	}

	if spec.Access.Level == vedro.ObjectWriter && !caps.ObjectWriter {
		unsupported = append(
			unsupported,
			vedro.UnsupportedFeature{
				Field:   "access.level",
				Message: "ObjectWriter acces is unsupported by this provider",
				Reason:  vedro.BucketAccessUnsupportedObjectWriter,
			},
		)
	}
	if spec.Access.Level == vedro.ObjectReader && !caps.ObjectReader {
		unsupported = append(
			unsupported,
			vedro.UnsupportedFeature{
				Field:   "access.level",
				Message: "ObjectReader acces is unsupported by this provider",
				Reason:  vedro.BucketAccessUnsupportedObjectReader,
			},
		)
	}

	return unsupported
}
