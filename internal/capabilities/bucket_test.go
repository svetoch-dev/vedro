package capabilities

import (
	// 	"reflect"

	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	// 	"cloud.google.com/go/storage"
	vedro "github.com/svetoch-dev/vedro/api/v1alpha1"
	"github.com/svetoch-dev/vedro/internal/cloud"
	"github.com/svetoch-dev/vedro/internal/helpers"
	// "github.com/svetoch-dev/vedro/internal/cloud"
	// "github.com/svetoch-dev/vedro/internal/helpers"
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
		"StorageClassInfrequent": vedro.UnsupportedFeature{
			Field:   "spec.StorageClass",
			Message: fmt.Sprintf("StorageClass %s is not supported by this provider", vedro.BucketStorageClassInfrequentAccess),
			Reason:  vedro.BucketUnsupportedStorageClass,
		},
		"StorageClassArchive": vedro.UnsupportedFeature{
			Field:   "spec.StorageClass",
			Message: fmt.Sprintf("StorageClass %s is not supported by this provider", vedro.BucketStorageClassArchive),
			Reason:  vedro.BucketUnsupportedStorageClass,
		},
	}
)

var _ = Describe("ValidateBucketCapabilities", func() {

	It("returns empty []UnsupportedFeature if spec is default", func() {
		caps := cloud.BucketCapabilities{}
		bucket := vedro.BucketSpec{}
		unsupported := ValidateBucketCapabilities(caps, bucket)
		Expect(unsupported).To(BeEmpty())
	})
	It("supported/set Labels,Versioning,PublicAccessPrevention", func() {
		caps := cloud.BucketCapabilities{
			Versioning:             true,
			PublicAccessPrevention: true,
			Labels:                 true,
		}
		bucket := vedro.BucketSpec{
			StorageClass:           "Standard",
			PublicAccessPrevention: helpers.Ptr(true),
			Versioning: &vedro.BucketVersioning{
				Enabled: true,
			},
			Labels: map[string]string{
				"some": "label",
			},
		}
		unsupported := ValidateBucketCapabilities(caps, bucket)
		Expect(unsupported).To(BeEmpty())
	})
	It("unsupported/set Labels,Versioning,PublicAccessPrevention", func() {
		caps := cloud.BucketCapabilities{
			Versioning:             false,
			PublicAccessPrevention: false,
			Labels:                 false,
		}
		bucket := vedro.BucketSpec{
			StorageClass:           "Standard",
			PublicAccessPrevention: helpers.Ptr(true),
			Versioning: &vedro.BucketVersioning{
				Enabled: true,
			},
			Labels: map[string]string{
				"some": "label",
			},
		}
		want := []vedro.UnsupportedFeature{
			unsupportedFeatures["Versioning"],
			unsupportedFeatures["Labels"],
			unsupportedFeatures["PublicAccessPrevention"],
		}
		unsupported := ValidateBucketCapabilities(caps, bucket)
		Expect(unsupported).NotTo(BeEmpty())
		Expect(unsupported).To(Equal(want))

	})
	It("supported/set Lifecycle with expiration rules", func() {
		caps := cloud.BucketCapabilities{
			Lifecycle: cloud.LifecycleCapabilities{
				RuleNames:      true,
				RuleExpiration: true,
			},
		}
		bucket := vedro.BucketSpec{
			Lifecycle: &vedro.BucketLifecycle{
				Rules: []vedro.BucketLifecycleRule{
					{
						Name:    helpers.Ptr("name"),
						Enabled: true,
						AgeDays: helpers.Ptr(int64(12)),
						Action:  vedro.BucketLifecycleActionDelete,
					},
				},
			},
		}
		unsupported := ValidateBucketCapabilities(caps, bucket)
		Expect(unsupported).To(BeEmpty())
	})
	It("unsupported/set Lifecycle with expiration rules", func() {
		bucket := vedro.BucketSpec{
			Lifecycle: &vedro.BucketLifecycle{
				Rules: []vedro.BucketLifecycleRule{
					{
						Name:    helpers.Ptr("name"),
						Enabled: true,
						AgeDays: helpers.Ptr(int64(12)),
						Action:  vedro.BucketLifecycleActionDelete,
					},
				},
			},
		}
		caps := cloud.BucketCapabilities{
			Lifecycle: cloud.LifecycleCapabilities{
				RuleExpiration: false,
			},
		}

		want := []vedro.UnsupportedFeature{
			unsupportedFeatures["Lifecycle"],
		}
		unsupported := ValidateBucketCapabilities(caps, bucket)
		Expect(unsupported).NotTo(BeEmpty())
		Expect(unsupported).To(Equal(want))
	})
	It("unsupported/set Lifecycle with named rules", func() {
		bucket := vedro.BucketSpec{
			Lifecycle: &vedro.BucketLifecycle{
				Rules: []vedro.BucketLifecycleRule{
					{
						Name:    helpers.Ptr("name"),
						Enabled: true,
						AgeDays: helpers.Ptr(int64(12)),
						Action:  vedro.BucketLifecycleActionDelete,
					},
				},
			},
		}
		caps := cloud.BucketCapabilities{
			Lifecycle: cloud.LifecycleCapabilities{
				RuleExpiration: true,
				RuleNames:      false,
			},
		}

		ufNamed := unsupportedFeatures["LifecycleNamed"]
		ufNamed.Field = fmt.Sprintf(ufNamed.Field, 0)

		want := []vedro.UnsupportedFeature{
			ufNamed,
		}
		unsupported := ValidateBucketCapabilities(caps, bucket)
		Expect(unsupported).NotTo(BeEmpty())
		Expect(unsupported).To(Equal(want))
	})

})
