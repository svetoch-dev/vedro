package gcp

import (
	"reflect"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"cloud.google.com/go/storage"
	vedro "github.com/svetoch-dev/vedro/api/v1alpha1"
	"github.com/svetoch-dev/vedro/internal/cloud"
	"github.com/svetoch-dev/vedro/internal/helpers"
)

// gcsSetLabels reads the unexported setLabels map from a
// storage.BucketAttrsToUpdate using reflection so the tests can verify which
// labels are being added or modified.
func gcsSetLabels(update *storage.BucketAttrsToUpdate) map[string]string {
	v := reflect.ValueOf(update).Elem()
	field := v.FieldByName("setLabels")
	if !field.IsValid() || field.IsNil() {
		return nil
	}
	result := map[string]string{}
	for _, key := range field.MapKeys() {
		result[key.String()] = field.MapIndex(key).String()
	}
	return result
}

// gcsDeleteLabels reads the unexported deleteLabels map from a
// storage.BucketAttrsToUpdate using reflection so the tests can verify which
// labels are being removed.
func gcsDeleteLabels(update *storage.BucketAttrsToUpdate) map[string]bool {
	v := reflect.ValueOf(update).Elem()
	field := v.FieldByName("deleteLabels")
	if !field.IsValid() || field.IsNil() {
		return nil
	}
	result := map[string]bool{}
	for _, key := range field.MapKeys() {
		result[key.String()] = field.MapIndex(key).Bool()
	}
	return result
}

var _ = Describe("setGCSLabels", func() {
	var currentAttrs *storage.BucketAttrsToUpdate

	BeforeEach(func() {
		currentAttrs = &storage.BucketAttrsToUpdate{}
	})

	It("does not update labels if update attrs are nil", func() {
		desired := map[string]string{
			"other": "set",
		}
		current := map[string]string{
			"remove": "no",
		}
		currentAttrs = nil

		setGCSLabels(desired, current, currentAttrs)

		Expect(currentAttrs).To(BeNil())
	})
	It("does update labels if there are no current labels", func() {
		desired := map[string]string{
			"keep":  "yes",
			"new":   "value",
			"other": "set",
		}
		current := map[string]string{}

		setGCSLabels(desired, current, currentAttrs)

		Expect(currentAttrs).NotTo(BeNil())
		Expect(gcsSetLabels(currentAttrs)).To(Equal(map[string]string{
			"keep":  "yes",
			"new":   "value",
			"other": "set",
		}))
		Expect(gcsDeleteLabels(currentAttrs)).To(BeEmpty())

	})
	It("does not update labels if they match", func() {
		desired := map[string]string{
			"keep":  "yes",
			"new":   "value",
			"other": "set",
		}
		current := map[string]string{
			"keep":  "yes",
			"new":   "value",
			"other": "set",
		}

		setGCSLabels(desired, current, currentAttrs)

		Expect(currentAttrs).NotTo(BeNil())
		Expect(gcsSetLabels(currentAttrs)).To(BeEmpty())
		Expect(gcsDeleteLabels(currentAttrs)).To(BeEmpty())

	})
	It("update labels values if they dont match", func() {
		desired := map[string]string{
			"keep":  "yes",
			"new":   "value",
			"other": "set",
		}
		current := map[string]string{
			"keep":  "no",
			"new":   "",
			"other": "unset",
		}

		setGCSLabels(desired, current, currentAttrs)

		Expect(currentAttrs).NotTo(BeNil())
		Expect(gcsSetLabels(currentAttrs)).To(Equal(map[string]string{
			"keep":  "yes",
			"new":   "value",
			"other": "set",
		}))

		Expect(gcsDeleteLabels(currentAttrs)).To(BeEmpty())

	})
	It("delete current labels if they are not in desired", func() {
		desired := map[string]string{
			"keep": "yes",
			"new":  "value",
			"test": "123",
		}
		current := map[string]string{
			"keep":  "no",
			"new":   "",
			"other": "unset",
		}

		setGCSLabels(desired, current, currentAttrs)

		Expect(currentAttrs).NotTo(BeNil())
		Expect(gcsSetLabels(currentAttrs)).To(Equal(map[string]string{
			"keep": "yes",
			"new":  "value",
			"test": "123",
		}))

		Expect(gcsDeleteLabels(currentAttrs)).To(Equal(map[string]bool{
			"other": true,
		}))

	})
})

var _ = Describe("toGCSLifeCycle", func() {
	It("returns an empty lifecycle when the input is nil", func() {
		lifecycle, err := toGCSLifeCycle(nil)

		Expect(err).NotTo(HaveOccurred())
		Expect(lifecycle).To(Equal(storage.Lifecycle{}))
		Expect(lifecycle.Rules).To(BeEmpty())
	})

	It("returns an empty lifecycle when there are no rules", func() {
		lifecycle, err := toGCSLifeCycle(&vedro.BucketLifecycle{})

		Expect(err).NotTo(HaveOccurred())
		Expect(lifecycle.Rules).To(BeEmpty())
	})

	It("skips disabled rules", func() {
		in := &vedro.BucketLifecycle{
			Rules: []vedro.BucketLifecycleRule{
				{
					Enabled: false,
					Action:  vedro.BucketLifecycleActionDelete,
					AgeDays: helpers.Ptr(int64(30)),
				},
			},
		}

		lifecycle, err := toGCSLifeCycle(in)

		Expect(err).NotTo(HaveOccurred())
		Expect(lifecycle.Rules).To(BeEmpty())
	})

	It("maps an enabled delete rule with an age condition", func() {
		in := &vedro.BucketLifecycle{
			Rules: []vedro.BucketLifecycleRule{
				{
					Enabled: true,
					Action:  vedro.BucketLifecycleActionDelete,
					AgeDays: helpers.Ptr(int64(30)),
				},
			},
		}

		lifecycle, err := toGCSLifeCycle(in)

		Expect(err).NotTo(HaveOccurred())
		Expect(lifecycle.Rules).To(HaveLen(1))
		Expect(lifecycle.Rules[0].Action).To(Equal(storage.LifecycleAction{
			Type: storage.DeleteAction,
		}))
		Expect(lifecycle.Rules[0].Condition.AgeInDays).To(Equal(int64(30)))
	})

	It("skips rules that have no age condition", func() {
		in := &vedro.BucketLifecycle{
			Rules: []vedro.BucketLifecycleRule{
				{
					Enabled: true,
					Action:  vedro.BucketLifecycleActionDelete,
					AgeDays: nil,
				},
			},
		}

		lifecycle, err := toGCSLifeCycle(in)

		Expect(err).NotTo(HaveOccurred())
		Expect(lifecycle.Rules).To(BeEmpty())
	})
	It("skips rules that have age condition 0", func() {
		in := &vedro.BucketLifecycle{
			Rules: []vedro.BucketLifecycleRule{
				{
					Enabled: true,
					Action:  vedro.BucketLifecycleActionDelete,
					AgeDays: helpers.Ptr(int64(0)),
				},
			},
		}

		lifecycle, err := toGCSLifeCycle(in)

		Expect(err).NotTo(HaveOccurred())
		Expect(lifecycle.Rules).To(BeEmpty())
	})

	It("appends multiple enabled rules preserving order", func() {
		in := &vedro.BucketLifecycle{
			Rules: []vedro.BucketLifecycleRule{
				{
					Enabled: true,
					Action:  vedro.BucketLifecycleActionDelete,
					AgeDays: helpers.Ptr(int64(10)),
				},
				{
					Enabled: false,
					Action:  vedro.BucketLifecycleActionDelete,
					AgeDays: helpers.Ptr(int64(20)),
				},
				{
					Enabled: true,
					Action:  vedro.BucketLifecycleActionDelete,
					AgeDays: helpers.Ptr(int64(30)),
				},
			},
		}

		lifecycle, err := toGCSLifeCycle(in)

		Expect(err).NotTo(HaveOccurred())
		Expect(lifecycle.Rules).To(HaveLen(2))
		Expect(lifecycle.Rules[0].Condition.AgeInDays).To(Equal(int64(10)))
		Expect(lifecycle.Rules[1].Condition.AgeInDays).To(Equal(int64(30)))
	})

	It("returns an error when an action does not map to a GCS lifecycle action", func() {
		in := &vedro.BucketLifecycle{
			Rules: []vedro.BucketLifecycleRule{
				{
					Enabled: true,
					Action:  vedro.BucketLifecycleAction("Unknown"),
					AgeDays: helpers.Ptr(int64(30)),
				},
			},
		}

		lifecycle, err := toGCSLifeCycle(in)

		Expect(err).To(MatchError(ContainSubstring("lifecycle.rules[0].action Unknown doesn't map to any GCS lifecycle action")))
		Expect(lifecycle).To(Equal(storage.Lifecycle{}))
	})

	It("reports the index of the offending rule", func() {
		in := &vedro.BucketLifecycle{
			Rules: []vedro.BucketLifecycleRule{
				{
					Enabled: true,
					Action:  vedro.BucketLifecycleActionDelete,
					AgeDays: helpers.Ptr(int64(10)),
				},
				{
					Enabled: true,
					Action:  vedro.BucketLifecycleAction("Unknown"),
					AgeDays: helpers.Ptr(int64(30)),
				},
			},
		}

		_, err := toGCSLifeCycle(in)

		Expect(err).To(MatchError(ContainSubstring("lifecycle.rules[1].action Unknown")))
	})
})

var _ = Describe("fromGCSLifeCycle", func() {
	It("returns an empty lifecycle when there are no rules", func() {
		result := fromGCSLifeCycle(storage.Lifecycle{})

		Expect(result.Rules).To(BeEmpty())
	})

	It("maps a delete rule with a positive age condition", func() {
		in := storage.Lifecycle{
			Rules: []storage.LifecycleRule{
				{
					Action:    storage.LifecycleAction{Type: storage.DeleteAction},
					Condition: storage.LifecycleCondition{AgeInDays: 30},
				},
			},
		}

		result := fromGCSLifeCycle(in)

		Expect(result.Rules).To(HaveLen(1))
		Expect(result.Rules[0].Enabled).To(BeTrue())
		Expect(*result.Rules[0].AgeDays).To(Equal(int64(30)))
		Expect(result.Rules[0].Action).To(Equal(vedro.BucketLifecycleActionDelete))
	})

	It("skips rules with a zero age condition", func() {
		in := storage.Lifecycle{
			Rules: []storage.LifecycleRule{
				{
					Action:    storage.LifecycleAction{Type: storage.DeleteAction},
					Condition: storage.LifecycleCondition{AgeInDays: 0},
				},
			},
		}

		result := fromGCSLifeCycle(in)

		Expect(result.Rules).To(BeEmpty())
	})

	It("skips rules with a negative age condition", func() {
		in := storage.Lifecycle{
			Rules: []storage.LifecycleRule{
				{
					Action:    storage.LifecycleAction{Type: storage.DeleteAction},
					Condition: storage.LifecycleCondition{AgeInDays: -1},
				},
			},
		}

		result := fromGCSLifeCycle(in)

		Expect(result.Rules).To(BeEmpty())
	})

	It("skips rules with a non-delete action", func() {
		in := storage.Lifecycle{
			Rules: []storage.LifecycleRule{
				{
					Action:    storage.LifecycleAction{Type: "Unknown"},
					Condition: storage.LifecycleCondition{AgeInDays: 30},
				},
			},
		}

		result := fromGCSLifeCycle(in)

		Expect(result.Rules).To(BeEmpty())
	})

	It("preserves the order of the eligible rules", func() {
		in := storage.Lifecycle{
			Rules: []storage.LifecycleRule{
				{
					Action:    storage.LifecycleAction{Type: storage.DeleteAction},
					Condition: storage.LifecycleCondition{AgeInDays: 10},
				},
				{
					Action:    storage.LifecycleAction{Type: "Unknown"},
					Condition: storage.LifecycleCondition{AgeInDays: 20},
				},
				{
					Action:    storage.LifecycleAction{Type: storage.DeleteAction},
					Condition: storage.LifecycleCondition{AgeInDays: 0},
				},
				{
					Action:    storage.LifecycleAction{Type: storage.DeleteAction},
					Condition: storage.LifecycleCondition{AgeInDays: 30},
				},
			},
		}

		result := fromGCSLifeCycle(in)

		Expect(result.Rules).To(HaveLen(2))
		Expect(*result.Rules[0].AgeDays).To(Equal(int64(10)))
		Expect(result.Rules[0].Action).To(Equal(vedro.BucketLifecycleActionDelete))
		Expect(*result.Rules[1].AgeDays).To(Equal(int64(30)))
		Expect(result.Rules[1].Action).To(Equal(vedro.BucketLifecycleActionDelete))
	})
})

var _ = Describe("fromGCSBucketAttrs", func() {
	It("maps all fields when the bucket is fully populated", func() {
		in := storage.BucketAttrs{
			Name:                   "my-bucket",
			Location:               "EUROPE-WEST1",
			StorageClass:           "NEARLINE",
			PublicAccessPrevention: storage.PublicAccessPreventionEnforced,
			VersioningEnabled:      true,
			Labels:                 map[string]string{"env": "prod", "team": "data"},
			SoftDeletePolicy: &storage.SoftDeletePolicy{
				RetentionDuration: 0,
			},
			Lifecycle: storage.Lifecycle{
				Rules: []storage.LifecycleRule{
					{
						Action:    storage.LifecycleAction{Type: storage.DeleteAction},
						Condition: storage.LifecycleCondition{AgeInDays: 30},
					},
				},
			},
		}
		wantCloudSpecific := vedro.BucketCloudSpecificConfig{
			Gcp: &vedro.BucketGcpConfig{
				SoftDeletePolicy: &vedro.SoftDeletePolicy{
					RetentionDuration: v1.Duration{
						Duration: 0,
					},
				},
			},
		}
		result, err := fromGCSBucketAttrs(in)

		Expect(err).NotTo(HaveOccurred())
		Expect(result).NotTo(BeNil())
		Expect(result.Name).To(Equal("my-bucket"))
		Expect(result.Location).To(Equal("EUROPE-WEST1"))
		Expect(result.Properties).NotTo(BeNil())

		Expect(result.Properties.StorageClass).To(Equal(vedro.BucketStorageClassWarm))
		Expect(result.Properties.PublicAccessPrevention).To(Equal(helpers.Ptr(true)))
		Expect(result.Properties.Versioning).To(Equal(&vedro.BucketVersioning{Enabled: true}))
		Expect(result.Properties.Labels).To(Equal(map[string]string{"env": "prod", "team": "data"}))

		Expect(result.Properties.Lifecycle).NotTo(BeNil())
		Expect(result.Properties.Lifecycle.Rules).To(HaveLen(1))
		Expect(*result.Properties.Lifecycle.Rules[0].AgeDays).To(Equal(int64(30)))
		Expect(result.Properties.Lifecycle.Rules[0].Action).To(Equal(vedro.BucketLifecycleActionDelete))
		Expect(result.Properties.Lifecycle.Rules[0].Enabled).To(BeTrue())

		Expect(result.Properties.CloudSpecificConfig).NotTo(BeNil())
		Expect(*result.Properties.CloudSpecificConfig).To(Equal(wantCloudSpecific))
	})

	It("maps an inherited public access prevention to false", func() {
		in := storage.BucketAttrs{
			Name:                   "my-bucket",
			Location:               "EUROPE-WEST1",
			StorageClass:           "STANDARD",
			PublicAccessPrevention: storage.PublicAccessPreventionInherited,
			VersioningEnabled:      false,
		}

		result, err := fromGCSBucketAttrs(in)

		Expect(err).NotTo(HaveOccurred())
		Expect(*result.Properties.PublicAccessPrevention).To(BeFalse())
		Expect(*result.Properties.Versioning).To(Equal(vedro.BucketVersioning{Enabled: false}))
		Expect(result.Properties.StorageClass).To(Equal(vedro.BucketStorageClassStandard))
	})
	It("maps attributes to CloudSpecificConfig.gcp if they are set", func() {
		in := storage.BucketAttrs{
			Name:         "my-bucket",
			Location:     "EUROPE-WEST1",
			StorageClass: "STANDARD",
			SoftDeletePolicy: &storage.SoftDeletePolicy{
				RetentionDuration: 0,
			},
		}

		want := vedro.BucketCloudSpecificConfig{
			Gcp: &vedro.BucketGcpConfig{
				SoftDeletePolicy: &vedro.SoftDeletePolicy{
					RetentionDuration: v1.Duration{
						Duration: 0,
					},
				},
			},
		}

		result, err := fromGCSBucketAttrs(in)

		Expect(err).NotTo(HaveOccurred())
		Expect(result.Properties.CloudSpecificConfig).NotTo(BeNil())
		Expect(*result.Properties.CloudSpecificConfig).To(Equal(want))
		Expect(result.Properties.StorageClass).To(Equal(vedro.BucketStorageClassStandard))
	})
	It("does not map attributes to CloudSpecificConfig.gcp if they do not exist", func() {
		in := storage.BucketAttrs{
			Name:             "my-bucket",
			Location:         "EUROPE-WEST1",
			StorageClass:     "STANDARD",
			SoftDeletePolicy: nil,
		}

		result, err := fromGCSBucketAttrs(in)

		Expect(err).NotTo(HaveOccurred())
		Expect(result.Properties.CloudSpecificConfig).To(BeNil())
	})

	It("maps an unknown public access prevention to nil", func() {
		in := storage.BucketAttrs{
			Name:                   "my-bucket",
			Location:               "EUROPE-WEST1",
			StorageClass:           "STANDARD",
			PublicAccessPrevention: storage.PublicAccessPreventionUnknown,
		}

		result, err := fromGCSBucketAttrs(in)

		Expect(err).NotTo(HaveOccurred())
		Expect(result.Properties.PublicAccessPrevention).To(BeNil())
	})

	It("maps COLDLINE storage class to Cold", func() {
		in := storage.BucketAttrs{
			Name:         "my-bucket",
			Location:     "EUROPE-WEST1",
			StorageClass: "COLDLINE",
		}

		result, err := fromGCSBucketAttrs(in)

		Expect(err).NotTo(HaveOccurred())
		Expect(result.Properties.StorageClass).To(Equal(vedro.BucketStorageClassCold))
	})

	It("maps ARCHIVE storage class to Ice", func() {
		in := storage.BucketAttrs{
			Name:         "my-bucket",
			Location:     "EUROPE-WEST1",
			StorageClass: "ARCHIVE",
		}

		result, err := fromGCSBucketAttrs(in)

		Expect(err).NotTo(HaveOccurred())
		Expect(result.Properties.StorageClass).To(Equal(vedro.BucketStorageClassIce))
	})

	It("returns an empty (non-nil) lifecycle when the GCS lifecycle has no rules", func() {
		in := storage.BucketAttrs{
			Name:                   "my-bucket",
			Location:               "EUROPE-WEST1",
			StorageClass:           "STANDARD",
			PublicAccessPrevention: storage.PublicAccessPreventionInherited,
		}

		result, err := fromGCSBucketAttrs(in)

		Expect(err).NotTo(HaveOccurred())
		Expect(result.Properties.Lifecycle).NotTo(BeNil())
		Expect(result.Properties.Lifecycle.Rules).To(BeEmpty())
	})

	It("returns an error when the public access prevention does not map", func() {
		in := storage.BucketAttrs{
			Name:                   "my-bucket",
			Location:               "EUROPE-WEST1",
			StorageClass:           "STANDARD",
			PublicAccessPrevention: storage.PublicAccessPrevention(99),
		}

		result, err := fromGCSBucketAttrs(in)

		Expect(err).To(MatchError(ContainSubstring("gcs PublicAccessPrevention")))
		Expect(err).To(MatchError(ContainSubstring("doesnt map to any bucket StorageClass")))
		Expect(result).To(BeNil())
	})

	It("returns an error when the storage class does not map", func() {
		in := storage.BucketAttrs{
			Name:                   "my-bucket",
			Location:               "EUROPE-WEST1",
			StorageClass:           "UNKNOWN",
			PublicAccessPrevention: storage.PublicAccessPreventionInherited,
		}

		result, err := fromGCSBucketAttrs(in)

		Expect(err).To(MatchError(ContainSubstring("gcs StorageClass UNKNOWN doesnt map to any bucket StorageClass")))
		Expect(result).To(BeNil())
	})
})

var _ = Describe("toGCSBucketAttrs", func() {
	It("maps only name and location when properties is nil", func() {
		in := cloud.BucketAttrs{
			Name:       "my-bucket",
			Location:   "EUROPE-WEST1",
			Properties: nil,
		}

		result, err := toGCSBucketAttrs(in)

		Expect(err).NotTo(HaveOccurred())
		Expect(result).NotTo(BeNil())
		Expect(result.Name).To(Equal("my-bucket"))
		Expect(result.Location).To(Equal("EUROPE-WEST1"))
		Expect(result.StorageClass).To(BeEmpty())
		Expect(result.PublicAccessPrevention).To(BeZero())
		Expect(result.VersioningEnabled).To(BeFalse())
		Expect(result.Labels).To(BeNil())
		Expect(result.Lifecycle.Rules).To(BeEmpty())
	})

	It("maps all fields when the bucket is fully populated", func() {
		in := cloud.BucketAttrs{
			Name:     "my-bucket",
			Location: "EUROPE-WEST1",
			Properties: &vedro.BucketProperties{
				StorageClass:           vedro.BucketStorageClassWarm,
				PublicAccessPrevention: helpers.Ptr(true),
				Versioning:             &vedro.BucketVersioning{Enabled: true},
				Labels:                 map[string]string{"env": "prod", "team": "data"},
				Lifecycle: &vedro.BucketLifecycle{
					Rules: []vedro.BucketLifecycleRule{
						{
							Enabled: true,
							Action:  vedro.BucketLifecycleActionDelete,
							AgeDays: helpers.Ptr(int64(30)),
						},
					},
				},
				CloudSpecificConfig: &vedro.BucketCloudSpecificConfig{
					Gcp: &vedro.BucketGcpConfig{
						SoftDeletePolicy: &vedro.SoftDeletePolicy{
							RetentionDuration: v1.Duration{
								Duration: 7 * time.Hour,
							},
						},
					},
				},
			},
		}

		result, err := toGCSBucketAttrs(in)

		Expect(err).NotTo(HaveOccurred())
		Expect(result).NotTo(BeNil())
		Expect(result.Name).To(Equal("my-bucket"))
		Expect(result.Location).To(Equal("EUROPE-WEST1"))
		Expect(result.StorageClass).To(Equal("NEARLINE"))
		Expect(result.PublicAccessPrevention).To(Equal(storage.PublicAccessPreventionEnforced))
		Expect(result.VersioningEnabled).To(BeTrue())
		Expect(result.Labels).To(Equal(map[string]string{"env": "prod", "team": "data"}))

		Expect(result.Lifecycle.Rules).To(HaveLen(1))
		Expect(result.Lifecycle.Rules[0].Action).To(Equal(storage.LifecycleAction{Type: storage.DeleteAction}))
		Expect(result.Lifecycle.Rules[0].Condition.AgeInDays).To(Equal(int64(30)))

		Expect(result.SoftDeletePolicy).NotTo(BeNil())
		Expect(result.SoftDeletePolicy.RetentionDuration).To(Equal(7 * time.Hour))
	})
	It("maps cloudSpecifiConfigs if they are set", func() {
		in := cloud.BucketAttrs{
			Name:     "my-bucket",
			Location: "EUROPE-WEST1",
			Properties: &vedro.BucketProperties{
				StorageClass: vedro.BucketStorageClassWarm,
				CloudSpecificConfig: &vedro.BucketCloudSpecificConfig{
					Gcp: &vedro.BucketGcpConfig{
						SoftDeletePolicy: &vedro.SoftDeletePolicy{
							RetentionDuration: v1.Duration{
								Duration: 7 * time.Hour,
							},
						},
					},
				},
			},
		}

		result, err := toGCSBucketAttrs(in)

		Expect(err).NotTo(HaveOccurred())
		Expect(result).NotTo(BeNil())

		Expect(result.SoftDeletePolicy).NotTo(BeNil())
		Expect(result.SoftDeletePolicy.RetentionDuration).To(Equal(7 * time.Hour))
	})
	It("does not map cloudSpecifiConfigs if they are not set", func() {
		in := cloud.BucketAttrs{
			Name:     "my-bucket",
			Location: "EUROPE-WEST1",
			Properties: &vedro.BucketProperties{
				StorageClass: vedro.BucketStorageClassWarm,
				CloudSpecificConfig: &vedro.BucketCloudSpecificConfig{
					Yc: &vedro.BucketYcConfig{},
				},
			},
		}

		result, err := toGCSBucketAttrs(in)

		Expect(err).NotTo(HaveOccurred())
		Expect(result).NotTo(BeNil())

		Expect(result.SoftDeletePolicy).To(BeNil())
	})
	It("maps the standard storage class to STANDARD", func() {
		in := cloud.BucketAttrs{
			Name:     "my-bucket",
			Location: "EUROPE-WEST1",
			Properties: &vedro.BucketProperties{
				StorageClass: vedro.BucketStorageClassStandard,
			},
		}

		result, err := toGCSBucketAttrs(in)

		Expect(err).NotTo(HaveOccurred())
		Expect(result.StorageClass).To(Equal("STANDARD"))
	})

	It("maps the ice storage class to ARCHIVE", func() {
		in := cloud.BucketAttrs{
			Name:     "my-bucket",
			Location: "EUROPE-WEST1",
			Properties: &vedro.BucketProperties{
				StorageClass: vedro.BucketStorageClassIce,
			},
		}

		result, err := toGCSBucketAttrs(in)

		Expect(err).NotTo(HaveOccurred())
		Expect(result.StorageClass).To(Equal("ARCHIVE"))
	})

	It("maps the cold storage class to COLDLINE", func() {
		in := cloud.BucketAttrs{
			Name:     "my-bucket",
			Location: "EUROPE-WEST1",
			Properties: &vedro.BucketProperties{
				StorageClass: vedro.BucketStorageClassCold,
			},
		}

		result, err := toGCSBucketAttrs(in)

		Expect(err).NotTo(HaveOccurred())
		Expect(result.StorageClass).To(Equal("COLDLINE"))
	})

	It("maps a nil public access prevention to inherited", func() {
		in := cloud.BucketAttrs{
			Name:     "my-bucket",
			Location: "EUROPE-WEST1",
			Properties: &vedro.BucketProperties{
				StorageClass:           vedro.BucketStorageClassStandard,
				PublicAccessPrevention: nil,
			},
		}

		result, err := toGCSBucketAttrs(in)

		Expect(err).NotTo(HaveOccurred())
		Expect(result.PublicAccessPrevention).To(Equal(storage.PublicAccessPreventionInherited))
	})

	It("maps a false public access prevention to inherited", func() {
		in := cloud.BucketAttrs{
			Name:     "my-bucket",
			Location: "EUROPE-WEST1",
			Properties: &vedro.BucketProperties{
				StorageClass:           vedro.BucketStorageClassStandard,
				PublicAccessPrevention: helpers.Ptr(false),
			},
		}

		result, err := toGCSBucketAttrs(in)

		Expect(err).NotTo(HaveOccurred())
		Expect(result.PublicAccessPrevention).To(Equal(storage.PublicAccessPreventionInherited))
	})

	It("maps a nil versioning to disabled", func() {
		in := cloud.BucketAttrs{
			Name:     "my-bucket",
			Location: "EUROPE-WEST1",
			Properties: &vedro.BucketProperties{
				StorageClass: vedro.BucketStorageClassStandard,
				Versioning:   nil,
			},
		}

		result, err := toGCSBucketAttrs(in)

		Expect(err).NotTo(HaveOccurred())
		Expect(result.VersioningEnabled).To(BeFalse())
	})

	It("maps a disabled versioning to disabled", func() {
		in := cloud.BucketAttrs{
			Name:     "my-bucket",
			Location: "EUROPE-WEST1",
			Properties: &vedro.BucketProperties{
				StorageClass: vedro.BucketStorageClassStandard,
				Versioning:   &vedro.BucketVersioning{Enabled: false},
			},
		}

		result, err := toGCSBucketAttrs(in)

		Expect(err).NotTo(HaveOccurred())
		Expect(result.VersioningEnabled).To(BeFalse())
	})

	It("passes nil labels through unchanged", func() {
		in := cloud.BucketAttrs{
			Name:     "my-bucket",
			Location: "EUROPE-WEST1",
			Properties: &vedro.BucketProperties{
				StorageClass: vedro.BucketStorageClassStandard,
				Labels:       nil,
			},
		}

		result, err := toGCSBucketAttrs(in)

		Expect(err).NotTo(HaveOccurred())
		Expect(result.Labels).To(BeNil())
	})

	It("maps a nil lifecycle to an empty GCS lifecycle", func() {
		in := cloud.BucketAttrs{
			Name:     "my-bucket",
			Location: "EUROPE-WEST1",
			Properties: &vedro.BucketProperties{
				StorageClass: vedro.BucketStorageClassStandard,
				Lifecycle:    nil,
			},
		}

		result, err := toGCSBucketAttrs(in)

		Expect(err).NotTo(HaveOccurred())
		Expect(result.Lifecycle.Rules).To(BeEmpty())
	})

	It("returns an error when the storage class does not map", func() {
		in := cloud.BucketAttrs{
			Name:     "my-bucket",
			Location: "EUROPE-WEST1",
			Properties: &vedro.BucketProperties{
				StorageClass: vedro.BucketStorageClass("Unknown"),
			},
		}

		result, err := toGCSBucketAttrs(in)

		Expect(err).To(MatchError(ContainSubstring("gcs StorageClass Unknown doesnt map to any bucket StorageClass")))
		Expect(result).To(BeNil())
	})
})

var _ = Describe("patchGCSBucketAttrs", func() {
	var currentAttrs *cloud.BucketAttrs

	BeforeEach(func() {
		currentAttrs = &cloud.BucketAttrs{
			Properties: &vedro.BucketProperties{
				Labels: map[string]string{"keep": "yes", "remove": "no"},
			},
		}
	})

	It("returns an empty update when the patch has no changes", func() {
		update, err := patchGCSBucketAttrs(cloud.BucketPatch{}, currentAttrs)

		Expect(err).NotTo(HaveOccurred())
		Expect(update).NotTo(BeNil())
		Expect(update.StorageClass).To(BeEmpty())
		Expect(update.PublicAccessPrevention).To(BeZero())
		Expect(update.VersioningEnabled).To(BeNil())
		Expect(update.Lifecycle).To(BeNil())
		Expect(gcsSetLabels(update)).To(BeEmpty())
		Expect(gcsDeleteLabels(update)).To(BeEmpty())
	})

	It("sets the storage class when the patch storage class is set", func() {
		patch := cloud.BucketPatch{
			StorageClass: helpers.PatchTo(vedro.BucketStorageClassIce),
		}

		update, err := patchGCSBucketAttrs(patch, currentAttrs)

		Expect(err).NotTo(HaveOccurred())
		Expect(update.StorageClass).To(Equal("ARCHIVE"))
	})

	It("sets the cold storage class when the patch storage class is set", func() {
		patch := cloud.BucketPatch{
			StorageClass: helpers.PatchTo(vedro.BucketStorageClassCold),
		}

		update, err := patchGCSBucketAttrs(patch, currentAttrs)

		Expect(err).NotTo(HaveOccurred())
		Expect(update.StorageClass).To(Equal("COLDLINE"))
	})

	It("returns an error when the patch storage class does not map", func() {
		patch := cloud.BucketPatch{
			StorageClass: helpers.PatchTo(vedro.BucketStorageClass("Unknown")),
		}

		update, err := patchGCSBucketAttrs(patch, currentAttrs)

		Expect(err).To(MatchError(ContainSubstring(`bucket storage class "Unknown" does not map to GCS`)))
		Expect(update).To(BeNil())
	})

	It("enables versioning when the patch versioning is set to enabled", func() {
		patch := cloud.BucketPatch{
			Versioning: helpers.PatchTo(&vedro.BucketVersioning{Enabled: true}),
		}

		update, err := patchGCSBucketAttrs(patch, currentAttrs)

		Expect(err).NotTo(HaveOccurred())
		Expect(update.VersioningEnabled).To(BeTrue())
	})

	It("disables versioning when the patch versioning is set to disabled", func() {
		patch := cloud.BucketPatch{
			Versioning: helpers.PatchTo(&vedro.BucketVersioning{Enabled: false}),
		}

		update, err := patchGCSBucketAttrs(patch, currentAttrs)

		Expect(err).NotTo(HaveOccurred())
		Expect(update.VersioningEnabled).To(BeFalse())
	})

	It("disables versioning when the patch versioning is nil", func() {
		patch := cloud.BucketPatch{
			Versioning: helpers.PatchTo[*vedro.BucketVersioning](nil),
		}

		update, err := patchGCSBucketAttrs(patch, currentAttrs)

		Expect(err).NotTo(HaveOccurred())
		Expect(update.VersioningEnabled).To(BeFalse())
	})

	It("sets enforced public access prevention when the patch PAP is true", func() {
		patch := cloud.BucketPatch{
			PublicAccessPrevention: helpers.PatchTo(helpers.Ptr(true)),
		}

		update, err := patchGCSBucketAttrs(patch, currentAttrs)

		Expect(err).NotTo(HaveOccurred())
		Expect(update.PublicAccessPrevention).To(Equal(storage.PublicAccessPreventionEnforced))
	})

	It("sets inherited public access prevention when the patch PAP is false", func() {
		patch := cloud.BucketPatch{
			PublicAccessPrevention: helpers.PatchTo(helpers.Ptr(false)),
		}

		update, err := patchGCSBucketAttrs(patch, currentAttrs)

		Expect(err).NotTo(HaveOccurred())
		Expect(update.PublicAccessPrevention).To(Equal(storage.PublicAccessPreventionInherited))
	})

	It("sets inherited public access prevention when the patch PAP is nil", func() {
		patch := cloud.BucketPatch{
			PublicAccessPrevention: helpers.PatchTo[*bool](nil),
		}

		update, err := patchGCSBucketAttrs(patch, currentAttrs)

		Expect(err).NotTo(HaveOccurred())
		Expect(update.PublicAccessPrevention).To(Equal(storage.PublicAccessPreventionInherited))
	})

	It("sets the lifecycle when the patch lifecycle is set", func() {
		patch := cloud.BucketPatch{
			Lifecycle: helpers.PatchTo(&vedro.BucketLifecycle{
				Rules: []vedro.BucketLifecycleRule{
					{
						Enabled: true,
						Action:  vedro.BucketLifecycleActionDelete,
						AgeDays: helpers.Ptr(int64(30)),
					},
				},
			}),
		}

		update, err := patchGCSBucketAttrs(patch, currentAttrs)

		Expect(err).NotTo(HaveOccurred())
		Expect(update.Lifecycle).NotTo(BeNil())
		Expect(update.Lifecycle.Rules).To(HaveLen(1))
		Expect(update.Lifecycle.Rules[0].Action).To(Equal(storage.LifecycleAction{Type: storage.DeleteAction}))
		Expect(update.Lifecycle.Rules[0].Condition.AgeInDays).To(Equal(int64(30)))
	})

	It("adds new labels, updates existing ones and removes labels absent from the patch", func() {
		patch := cloud.BucketPatch{
			Labels: helpers.PatchTo(map[string]string{
				"keep":  "yes",
				"new":   "value",
				"other": "set",
			}),
		}

		update, err := patchGCSBucketAttrs(patch, currentAttrs)

		Expect(err).NotTo(HaveOccurred())
		Expect(gcsSetLabels(update)).To(Equal(map[string]string{
			"keep":  "yes",
			"new":   "value",
			"other": "set",
		}))
		Expect(gcsDeleteLabels(update)).To(Equal(map[string]bool{
			"remove": true,
		}))
	})

	It("does not touch labels when the patch labels are not set", func() {
		patch := cloud.BucketPatch{
			StorageClass: helpers.PatchTo(vedro.BucketStorageClassStandard),
		}

		update, err := patchGCSBucketAttrs(patch, currentAttrs)

		Expect(err).NotTo(HaveOccurred())
		Expect(gcsSetLabels(update)).To(BeEmpty())
		Expect(gcsDeleteLabels(update)).To(BeEmpty())
	})
	It("patches cloudSpecificConfigs", func() {
		patch := cloud.BucketPatch{
			CloudSpecificConfig: helpers.PatchTo(&vedro.BucketCloudSpecificConfig{
				Gcp: &vedro.BucketGcpConfig{
					SoftDeletePolicy: &vedro.SoftDeletePolicy{
						RetentionDuration: v1.Duration{
							Duration: 8 * time.Hour,
						},
					},
				},
			}),
		}

		update, err := patchGCSBucketAttrs(patch, currentAttrs)

		Expect(err).NotTo(HaveOccurred())
		Expect(update.SoftDeletePolicy).NotTo(BeNil())
		Expect(update.SoftDeletePolicy.RetentionDuration).To(Equal(8 * time.Hour))
	})

	It("maps all set fields at once", func() {
		patch := cloud.BucketPatch{
			StorageClass:           helpers.PatchTo(vedro.BucketStorageClassIce),
			Versioning:             helpers.PatchTo(&vedro.BucketVersioning{Enabled: true}),
			PublicAccessPrevention: helpers.PatchTo(helpers.Ptr(true)),
			Lifecycle: helpers.PatchTo(&vedro.BucketLifecycle{
				Rules: []vedro.BucketLifecycleRule{
					{
						Enabled: true,
						Action:  vedro.BucketLifecycleActionDelete,
						AgeDays: helpers.Ptr(int64(10)),
					},
				},
			}),
			Labels: helpers.PatchTo(map[string]string{"env": "prod"}),
			CloudSpecificConfig: helpers.PatchTo(&vedro.BucketCloudSpecificConfig{
				Gcp: &vedro.BucketGcpConfig{
					SoftDeletePolicy: &vedro.SoftDeletePolicy{
						RetentionDuration: v1.Duration{
							Duration: 8 * time.Hour,
						},
					},
				},
			}),
		}

		update, err := patchGCSBucketAttrs(patch, currentAttrs)

		Expect(err).NotTo(HaveOccurred())
		Expect(update.StorageClass).To(Equal("ARCHIVE"))
		Expect(update.VersioningEnabled).To(BeTrue())
		Expect(update.PublicAccessPrevention).To(Equal(storage.PublicAccessPreventionEnforced))
		Expect(update.Lifecycle).NotTo(BeNil())
		Expect(update.Lifecycle.Rules).To(HaveLen(1))
		Expect(update.Lifecycle.Rules[0].Condition.AgeInDays).To(Equal(int64(10)))
		Expect(gcsSetLabels(update)).To(Equal(map[string]string{"env": "prod"}))
		Expect(gcsDeleteLabels(update)).To(Equal(map[string]bool{"keep": true, "remove": true}))

		Expect(update.SoftDeletePolicy).NotTo(BeNil())
		Expect(update.SoftDeletePolicy.RetentionDuration).To(Equal(8 * time.Hour))

	})
})
