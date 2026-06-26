package capabilities

import (
	// 	"reflect"

	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	// 	"cloud.google.com/go/storage"
	vedrov1alpha1 "github.com/svetoch-dev/vedro/api/v1alpha1"
	"github.com/svetoch-dev/vedro/internal/cloud"
	"github.com/svetoch-dev/vedro/internal/helpers"
	// "github.com/svetoch-dev/vedro/internal/cloud"
	// "github.com/svetoch-dev/vedro/internal/helpers"
)

var (
	unsupportedFeatures = map[string]vedrov1alpha1.UnsupportedFeature{
		"Versioning": vedrov1alpha1.UnsupportedFeature{
			Field:   "spec.Versioning",
			Message: "Versioning is not supported by this provider",
			Reason:  vedrov1alpha1.BucketUnsupportedVersioning,
		},
		"Lifecycle": vedrov1alpha1.UnsupportedFeature{
			Field:   "spec.lifecycle",
			Message: "Lifecycle is not supported by this provider",
			Reason:  vedrov1alpha1.BucketUnsupportedLifecycle,
		},
		"LifecycleEpiration": vedrov1alpha1.UnsupportedFeature{
			Field:   "spec.lifecycle.rules[%d].AgeDays",
			Message: "Object expiration is not supported by this provider",
			Reason:  vedrov1alpha1.BucketUnsupportedLifecycleExpiration,
		},
		"LifecycleNamed": vedrov1alpha1.UnsupportedFeature{
			Field:   "spec.lifecycle.rules[%d].Name",
			Message: "Named lifecycle rules are not supported by this provider",
			Reason:  vedrov1alpha1.BucketUnsupportedLifecycleNamed,
		},
		"Labels": vedrov1alpha1.UnsupportedFeature{
			Field:   "spec.Labels",
			Message: "Labels are not supported by this provider",
			Reason:  vedrov1alpha1.BucketUnsupportedLabels,
		},
		"PublicAccessPrevention": vedrov1alpha1.UnsupportedFeature{
			Field:   "spec.PublicAccessPrevention",
			Message: "PublicAccessPrevention is not supported by this provider",
			Reason:  vedrov1alpha1.BucketUnsupportedPublicAccessPrevention,
		},
		"StorageClassInfrequent": vedrov1alpha1.UnsupportedFeature{
			Field:   "spec.StorageClass",
			Message: fmt.Sprintf("StorageClass %s is not supported by this provider", vedrov1alpha1.BucketStorageClassInfrequentAccess),
			Reason:  vedrov1alpha1.BucketUnsupportedStorageClass,
		},
		"StorageClassArchive": vedrov1alpha1.UnsupportedFeature{
			Field:   "spec.StorageClass",
			Message: fmt.Sprintf("StorageClass %s is not supported by this provider", vedrov1alpha1.BucketStorageClassArchive),
			Reason:  vedrov1alpha1.BucketUnsupportedStorageClass,
		},
	}
)

var _ = Describe("ValidateBucketCapabilities", func() {

	It("returns empty []UnsupportedFeature if spec is default", func() {
		caps := cloud.BucketCapabilities{}
		bucket := vedrov1alpha1.BucketSpec{}
		unsupported := ValidateBucketCapabilities(caps, bucket)
		Expect(unsupported).To(BeEmpty())
	})
	It("supported/set Labels,Versioning,PublicAccessPrevention", func() {
		caps := cloud.BucketCapabilities{
			Versioning:             true,
			PublicAccessPrevention: true,
			Labels:                 true,
		}
		bucket := vedrov1alpha1.BucketSpec{
			StorageClass:           "Standard",
			PublicAccessPrevention: helpers.Ptr(true),
			Versioning: &vedrov1alpha1.BucketVersioning{
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
		bucket := vedrov1alpha1.BucketSpec{
			StorageClass:           "Standard",
			PublicAccessPrevention: helpers.Ptr(true),
			Versioning: &vedrov1alpha1.BucketVersioning{
				Enabled: true,
			},
			Labels: map[string]string{
				"some": "label",
			},
		}
		want := []vedrov1alpha1.UnsupportedFeature{
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
		bucket := vedrov1alpha1.BucketSpec{
			Lifecycle: &vedrov1alpha1.BucketLifecycle{
				Rules: []vedrov1alpha1.BucketLifecycleRule{
					{
						Name:    helpers.Ptr("name"),
						Enabled: true,
						AgeDays: helpers.Ptr(int64(12)),
						Action:  vedrov1alpha1.BucketLifecycleActionDelete,
					},
				},
			},
		}
		unsupported := ValidateBucketCapabilities(caps, bucket)
		Expect(unsupported).To(BeEmpty())
	})
	It("unsupported/set Lifecycle with expiration rules", func() {
		bucket := vedrov1alpha1.BucketSpec{
			Lifecycle: &vedrov1alpha1.BucketLifecycle{
				Rules: []vedrov1alpha1.BucketLifecycleRule{
					{
						Name:    helpers.Ptr("name"),
						Enabled: true,
						AgeDays: helpers.Ptr(int64(12)),
						Action:  vedrov1alpha1.BucketLifecycleActionDelete,
					},
				},
			},
		}
		caps := cloud.BucketCapabilities{
			Lifecycle: cloud.LifecycleCapabilities{
				RuleExpiration: false,
			},
		}

		want := []vedrov1alpha1.UnsupportedFeature{
			unsupportedFeatures["Lifecycle"],
		}
		unsupported := ValidateBucketCapabilities(caps, bucket)
		Expect(unsupported).NotTo(BeEmpty())
		Expect(unsupported).To(Equal(want))
	})
	It("unsupported/set Lifecycle with named rules", func() {
		bucket := vedrov1alpha1.BucketSpec{
			Lifecycle: &vedrov1alpha1.BucketLifecycle{
				Rules: []vedrov1alpha1.BucketLifecycleRule{
					{
						Name:    helpers.Ptr("name"),
						Enabled: true,
						AgeDays: helpers.Ptr(int64(12)),
						Action:  vedrov1alpha1.BucketLifecycleActionDelete,
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

		want := []vedrov1alpha1.UnsupportedFeature{
			ufNamed,
		}
		unsupported := ValidateBucketCapabilities(caps, bucket)
		Expect(unsupported).NotTo(BeEmpty())
		Expect(unsupported).To(Equal(want))
	})

})
