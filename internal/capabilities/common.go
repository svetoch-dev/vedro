package capabilities

import (
	"fmt"

	vedro "github.com/svetoch-dev/vedro/api/v1alpha1"
)

var (
	unsupportedFeatures = map[string]vedro.UnsupportedFeature{
		"Versioning": {
			Field:   "spec.Versioning",
			Message: "Versioning is not supported by this provider",
			Reason:  vedro.BucketUnsupportedVersioning,
		},
		"Lifecycle": {
			Field:   "spec.lifecycle",
			Message: "Lifecycle is not supported by this provider",
			Reason:  vedro.BucketUnsupportedLifecycle,
		},
		"LifecycleEpiration": {
			Field:   "spec.lifecycle.rules[%d].AgeDays",
			Message: "Object expiration is not supported by this provider",
			Reason:  vedro.BucketUnsupportedLifecycleExpiration,
		},
		"LifecycleNamed": {
			Field:   "spec.lifecycle.rules[%d].Name",
			Message: "Named lifecycle rules are not supported by this provider",
			Reason:  vedro.BucketUnsupportedLifecycleNamed,
		},
		"Labels": {
			Field:   "spec.Labels",
			Message: "Labels are not supported by this provider",
			Reason:  vedro.BucketUnsupportedLabels,
		},
		"PublicAccessPrevention": {
			Field:   "spec.PublicAccessPrevention",
			Message: "PublicAccessPrevention is not supported by this provider",
			Reason:  vedro.BucketUnsupportedPublicAccessPrevention,
		},
		"StorageClassWarm": {
			Field:   "spec.StorageClass",
			Message: fmt.Sprintf("StorageClass %s is not supported by this provider", vedro.BucketStorageClassWarm),
			Reason:  vedro.BucketUnsupportedStorageClass,
		},
		"StorageClassIce": {
			Field:   "spec.StorageClass",
			Message: fmt.Sprintf("StorageClass %s is not supported by this provider", vedro.BucketStorageClassIce),
			Reason:  vedro.BucketUnsupportedStorageClass,
		},
		"StorageClassCold": {
			Field:   "spec.StorageClass",
			Message: fmt.Sprintf("StorageClass %s is not supported by this provider", vedro.BucketStorageClassCold),
			Reason:  vedro.BucketUnsupportedStorageClass,
		},
		"BucketAdmin": {
			Field:   "access.level",
			Message: "BucketAdmin acces is unsupported by this provider",
			Reason:  vedro.BucketAccessUnsupportedBucketAdmin,
		},
		"ObjectAdmin": {
			Field:   "access.level",
			Message: "ObjectAdmin acces is unsupported by this provider",
			Reason:  vedro.BucketAccessUnsupportedObjectAdmin,
		},
		"ObjectWriter": {
			Field:   "access.level",
			Message: "ObjectWriter acces is unsupported by this provider",
			Reason:  vedro.BucketAccessUnsupportedObjectWriter,
		},
		"ObjectReader": {
			Field:   "access.level",
			Message: "ObjectReader acces is unsupported by this provider",
			Reason:  vedro.BucketAccessUnsupportedObjectReader,
		},
	}
)
