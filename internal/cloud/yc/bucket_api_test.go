package yc

import (
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	vedro "github.com/svetoch-dev/vedro/api/v1alpha1"
	"github.com/svetoch-dev/vedro/internal/cloud"
	"github.com/svetoch-dev/vedro/internal/helpers"
	storageapi "github.com/yandex-cloud/go-genproto/yandex/cloud/storage/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/fieldmaskpb"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

var _ = Describe("isNotFound", func() {
	DescribeTable("classifies YC API errors",
		func(err error, want bool) {
			Expect(isNotFound(err)).To(Equal(want))
		},
		Entry(
			"not found status",
			status.Error(codes.NotFound, "bucket not found"),
			true,
		),
		Entry(
			"wrapped not found status",
			fmt.Errorf("delete bucket: %w", status.Error(codes.NotFound, "bucket not found")),
			true,
		),
		Entry(
			"permission denied status",
			status.Error(codes.PermissionDenied, "permission denied"),
			false,
		),
		Entry(
			"plain error",
			fmt.Errorf("network error"),
			false,
		),
	)
})

var _ = Describe("fromYcVersioning", func() {
	DescribeTable("converts YC versioning",
		func(versioning storageapi.Versioning, want *vedro.BucketVersioning) {
			Expect(fromYcVersioning(versioning)).To(Equal(want))
		},
		Entry(
			"enabled",
			storageapi.Versioning_VERSIONING_ENABLED,
			&vedro.BucketVersioning{
				Enabled: true,
			},
		),
		Entry(
			"suspended",
			storageapi.Versioning_VERSIONING_SUSPENDED,
			&vedro.BucketVersioning{
				Enabled: false,
			},
		),
		Entry(
			"disabled",
			storageapi.Versioning_VERSIONING_DISABLED,
			nil,
		),
		Entry(
			"unspecified",
			storageapi.Versioning_VERSIONING_UNSPECIFIED,
			nil,
		),
	)
})

var _ = Describe("toYcVersioning", func() {
	DescribeTable("to YC versioning",
		func(versioning *vedro.BucketVersioning, want storageapi.Versioning) {
			Expect(toYcVersioning(versioning)).To(Equal(want))
		},
		Entry(
			"enabled",
			&vedro.BucketVersioning{
				Enabled: true,
			},
			storageapi.Versioning_VERSIONING_ENABLED,
		),
		Entry(
			"disabled",
			&vedro.BucketVersioning{
				Enabled: false,
			},
			storageapi.Versioning_VERSIONING_SUSPENDED,
		),
		Entry(
			"not set",
			nil,
			storageapi.Versioning_VERSIONING_DISABLED,
		),
	)
})

var _ = Describe("fromYcLifecycle", func() {
	DescribeTable("converts YC lifecycle rules",
		func(rules []*storageapi.LifecycleRule, want *vedro.BucketLifecycle) {
			Expect(fromYcLifecycle(rules)).To(Equal(want))
		},
		Entry(
			"nil",
			nil,
			nil,
		),
		Entry(
			"empty",
			[]*storageapi.LifecycleRule{},
			nil,
		),
		Entry(
			"skips nil and unsupported rules",
			[]*storageapi.LifecycleRule{
				nil,
				{
					Id:      wrapperspb.String("no-expiration"),
					Enabled: true,
				},
				{
					Id:         wrapperspb.String("no-days"),
					Enabled:    true,
					Expiration: &storageapi.LifecycleRule_Expiration{},
				},
			},
			nil,
		),
		Entry(
			"expiration days with name",
			[]*storageapi.LifecycleRule{
				{
					Id:      wrapperspb.String("delete-old-objects"),
					Enabled: true,
					Expiration: &storageapi.LifecycleRule_Expiration{
						Days: wrapperspb.Int64(30),
					},
				},
			},
			&vedro.BucketLifecycle{
				Rules: []vedro.BucketLifecycleRule{
					{
						Name:    helpers.Ptr("delete-old-objects"),
						Enabled: true,
						AgeDays: helpers.Ptr(int64(30)),
						Action:  vedro.BucketLifecycleActionDelete,
					},
				},
			},
		),
		Entry(
			"expiration days without name",
			[]*storageapi.LifecycleRule{
				{
					Enabled: false,
					Expiration: &storageapi.LifecycleRule_Expiration{
						Days: wrapperspb.Int64(7),
					},
				},
			},
			&vedro.BucketLifecycle{
				Rules: []vedro.BucketLifecycleRule{
					{
						Enabled: false,
						AgeDays: helpers.Ptr(int64(7)),
						Action:  vedro.BucketLifecycleActionDelete,
					},
				},
			},
		),
	)
})

var _ = Describe("toYcLifecycle", func() {
	DescribeTable("converts to YC lifecycle rules",
		func(rules *vedro.BucketLifecycle, want []*storageapi.LifecycleRule) {
			Expect(toYcLifecycle(rules)).To(Equal(want))
		},
		Entry(
			"nil",
			nil,
			[]*storageapi.LifecycleRule{},
		),
		Entry(
			"empty rules",
			&vedro.BucketLifecycle{},
			[]*storageapi.LifecycleRule{},
		),
		Entry(
			"lifecycle AgeDays with name and action delete",
			&vedro.BucketLifecycle{
				Rules: []vedro.BucketLifecycleRule{
					{
						Name:    helpers.Ptr("delete-old-objects"),
						Enabled: true,
						AgeDays: helpers.Ptr(int64(30)),
						Action:  vedro.BucketLifecycleActionDelete,
					},
				},
			},
			[]*storageapi.LifecycleRule{
				{
					Id:      wrapperspb.String("delete-old-objects"),
					Enabled: true,
					Expiration: &storageapi.LifecycleRule_Expiration{
						Days: wrapperspb.Int64(30),
					},
				},
			},
		),
		Entry(
			"multiple lifecycle AgeDays with action delete",
			&vedro.BucketLifecycle{
				Rules: []vedro.BucketLifecycleRule{
					{
						Name:    helpers.Ptr("delete-old-objects-1"),
						Enabled: false,
						AgeDays: helpers.Ptr(int64(30)),
						Action:  vedro.BucketLifecycleActionDelete,
					},
					{
						Name:    helpers.Ptr("delete-old-objects-2"),
						Enabled: true,
						AgeDays: helpers.Ptr(int64(10)),
						Action:  vedro.BucketLifecycleActionDelete,
					},
				},
			},
			[]*storageapi.LifecycleRule{
				{
					Id:      wrapperspb.String("delete-old-objects-1"),
					Enabled: false,
					Expiration: &storageapi.LifecycleRule_Expiration{
						Days: wrapperspb.Int64(30),
					},
				},
				{
					Id:      wrapperspb.String("delete-old-objects-2"),
					Enabled: true,
					Expiration: &storageapi.LifecycleRule_Expiration{
						Days: wrapperspb.Int64(10),
					},
				},
			},
		),
		Entry(
			"lifecycle AgeDays without name",
			&vedro.BucketLifecycle{
				Rules: []vedro.BucketLifecycleRule{
					{
						Enabled: true,
						AgeDays: helpers.Ptr(int64(7)),
						Action:  vedro.BucketLifecycleActionDelete,
					},
				},
			},
			[]*storageapi.LifecycleRule{
				{
					Enabled: true,
					Expiration: &storageapi.LifecycleRule_Expiration{
						Days: wrapperspb.Int64(7),
					},
				},
			},
		),
	)
})

var _ = Describe("fromYcTags", func() {
	DescribeTable("converts YC tags",
		func(tags []*storageapi.Tag, want map[string]string) {
			Expect(fromYcTags(tags)).To(Equal(want))
		},
		Entry(
			"nil",
			nil,
			map[string]string{},
		),
		Entry(
			"empty",
			[]*storageapi.Tag{},
			map[string]string{},
		),

		Entry(
			"multiple tags",
			[]*storageapi.Tag{
				{
					Key:   "env",
					Value: "prod",
				},
				{
					Key:   "team",
					Value: "storage",
				},
			},
			map[string]string{
				"env":  "prod",
				"team": "storage",
			},
		),
	)
})

var _ = Describe("toYcTags", func() {
	DescribeTable("converts to YC tags",
		func(tags map[string]string, want []*storageapi.Tag) {
			Expect(toYcTags(tags)).To(Equal(want))
		},
		Entry(
			"nil",
			nil,
			[]*storageapi.Tag{},
		),
		Entry(
			"empty",
			map[string]string{},
			[]*storageapi.Tag{},
		),

		Entry(
			"multiple tags",
			map[string]string{
				"env":  "prod",
				"team": "storage",
			},

			[]*storageapi.Tag{
				{
					Key:   "env",
					Value: "prod",
				},
				{
					Key:   "team",
					Value: "storage",
				},
			},
		),
	)
})

var _ = Describe("fromYcBucket", func() {
	It("returns an error for a nil bucket", func() {
		result, err := fromYcBucket(nil, "ru-central1")

		Expect(result).To(BeNil())
		Expect(err).To(MatchError("yc storageapi.Bucket is nil"))
	})

	It("returns an error for an unmapped storage class", func() {
		result, err := fromYcBucket(&storageapi.Bucket{
			Name:                "example",
			DefaultStorageClass: "UNKNOWN",
		}, "ru-central1")

		Expect(result).To(BeNil())
		Expect(err).To(MatchError("yc StorageClass UNKNOWN doesnt map to any bucket StorageClass"))
	})

	DescribeTable("maps bucket identity and storage class",
		func(ycStorageClass string, wantStorageClass vedro.BucketStorageClass) {
			result, err := fromYcBucket(&storageapi.Bucket{
				Name:                "example",
				DefaultStorageClass: ycStorageClass,
			}, "ru-central1")

			Expect(err).NotTo(HaveOccurred())
			Expect(result).To(Equal(&cloud.BucketAttrs{
				Name:     "example",
				Location: "ru-central1",
				Properties: &vedro.BucketProperties{
					StorageClass: wantStorageClass,
					Labels:       map[string]string{},
				},
			}))
		},
		Entry("standard", "STANDARD", vedro.BucketStorageClassStandard),
		Entry("nearline", "NEARLINE", vedro.BucketStorageClassCold),
		Entry("cold", "COLD", vedro.BucketStorageClassCold),
		Entry("standard ia", "STANDARD_IA", vedro.BucketStorageClassCold),
		Entry("ice", "ICE", vedro.BucketStorageClassIce),
		Entry("glacier", "GLACIER", vedro.BucketStorageClassIce),
	)
	It("maps the YC resource ID separately from the bucket name", func() {
		result, err := fromYcBucket(&storageapi.Bucket{
			Name:                "my-bucket",
			ResourceId:          "e3r-resource-id",
			DefaultStorageClass: "STANDARD",
		}, "ru-central1")

		Expect(err).NotTo(HaveOccurred())
		Expect(result.Name).To(Equal("my-bucket"))
		Expect(result.Id).To(Equal("e3r-resource-id"))
	})

})

var _ = Describe("toCreateBucketRequest", func() {
	It("returns an error for an unmapped storage class", func() {
		result, err := toCreateBucketRequest(cloud.BucketAttrs{
			Name: "example",
			Properties: &vedro.BucketProperties{
				StorageClass: vedro.BucketStorageClass("Unknown"),
			},
		}, "folder-id")

		Expect(result).To(BeNil())
		Expect(err).To(MatchError(`bucket storage class "Unknown" does not map to YC`))
	})

	DescribeTable("builds the create request with mapped storage class",
		func(storageClass vedro.BucketStorageClass, wantStorageClass string) {
			result, err := toCreateBucketRequest(cloud.BucketAttrs{
				Name: "example",
				Properties: &vedro.BucketProperties{
					StorageClass: storageClass,
				},
			}, "folder-id")

			Expect(err).NotTo(HaveOccurred())
			Expect(result).To(Equal(&storageapi.CreateBucketRequest{
				Name:                "example",
				FolderId:            "folder-id",
				Tags:                []*storageapi.Tag{},
				Versioning:          storageapi.Versioning_VERSIONING_DISABLED,
				LifecycleRules:      []*storageapi.LifecycleRule{},
				DefaultStorageClass: wantStorageClass,
			}))
		},
		Entry("standard", vedro.BucketStorageClassStandard, "STANDARD"),
		Entry("cold", vedro.BucketStorageClassCold, "COLD"),
		Entry("ice", vedro.BucketStorageClassIce, "ICE"),
	)
})

var _ = Describe("patchYcBucketAttrs", func() {
	It("builds an empty update for an empty patch", func() {
		result, err := patchYcBucketAttrs(cloud.BucketPatch{}, "example")

		Expect(err).NotTo(HaveOccurred())
		Expect(result).To(Equal(&storageapi.UpdateBucketRequest{
			Name:       "example",
			UpdateMask: &fieldmaskpb.FieldMask{},
		}))
	})

	It("returns an error for an unmapped storage class", func() {
		result, err := patchYcBucketAttrs(cloud.BucketPatch{
			StorageClass: helpers.PatchTo(vedro.BucketStorageClass("Unknown")),
		}, "example")

		Expect(result).To(BeNil())
		Expect(err).To(MatchError(`bucket storage class "Unknown" does not map to YC`))
	})

	DescribeTable("patches storage class",
		func(storageClass vedro.BucketStorageClass, wantStorageClass string) {
			result, err := patchYcBucketAttrs(cloud.BucketPatch{
				StorageClass: helpers.PatchTo(storageClass),
			}, "example")

			Expect(err).NotTo(HaveOccurred())
			Expect(result).To(Equal(&storageapi.UpdateBucketRequest{
				Name:                "example",
				DefaultStorageClass: wantStorageClass,
				UpdateMask: &fieldmaskpb.FieldMask{
					Paths: []string{"default_storage_class"},
				},
			}))
		},
		Entry("standard", vedro.BucketStorageClassStandard, "STANDARD"),
		Entry("cold", vedro.BucketStorageClassCold, "COLD"),
		Entry("ice", vedro.BucketStorageClassIce, "ICE"),
	)

	It("patches labels", func() {
		result, err := patchYcBucketAttrs(cloud.BucketPatch{
			Labels: helpers.PatchTo(map[string]string{
				"env": "prod",
			}),
		}, "example")

		Expect(err).NotTo(HaveOccurred())
		Expect(result).To(Equal(&storageapi.UpdateBucketRequest{
			Name: "example",
			Tags: []*storageapi.Tag{
				{
					Key:   "env",
					Value: "prod",
				},
			},
			UpdateMask: &fieldmaskpb.FieldMask{
				Paths: []string{"tags"},
			},
		}))
	})

	DescribeTable("patches versioning",
		func(versioning *vedro.BucketVersioning, want storageapi.Versioning) {
			result, err := patchYcBucketAttrs(cloud.BucketPatch{
				Versioning: cloud.Change[*vedro.BucketVersioning]{
					Set:   true,
					Value: versioning,
				},
			}, "example")

			Expect(err).NotTo(HaveOccurred())
			Expect(result).To(Equal(&storageapi.UpdateBucketRequest{
				Name:       "example",
				Versioning: want,
				UpdateMask: &fieldmaskpb.FieldMask{
					Paths: []string{"versioning"},
				},
			}))
		},
		Entry(
			"enabled",
			&vedro.BucketVersioning{Enabled: true},
			storageapi.Versioning_VERSIONING_ENABLED,
		),
		Entry(
			"suspended",
			&vedro.BucketVersioning{Enabled: false},
			storageapi.Versioning_VERSIONING_SUSPENDED,
		),
		Entry(
			"nil is suspended because YC cannot update back to disabled",
			nil,
			storageapi.Versioning_VERSIONING_SUSPENDED,
		),
	)

	It("patches lifecycle rules", func() {
		result, err := patchYcBucketAttrs(cloud.BucketPatch{
			Lifecycle: helpers.PatchTo(&vedro.BucketLifecycle{
				Rules: []vedro.BucketLifecycleRule{
					{
						Name:    helpers.Ptr("delete-old-objects"),
						Enabled: true,
						AgeDays: helpers.Ptr(int64(30)),
						Action:  vedro.BucketLifecycleActionDelete,
					},
				},
			}),
		}, "example")

		Expect(err).NotTo(HaveOccurred())
		Expect(result).To(Equal(&storageapi.UpdateBucketRequest{
			Name: "example",
			LifecycleRules: []*storageapi.LifecycleRule{
				{
					Id:      wrapperspb.String("delete-old-objects"),
					Enabled: true,
					Expiration: &storageapi.LifecycleRule_Expiration{
						Days: wrapperspb.Int64(30),
					},
				},
			},
			UpdateMask: &fieldmaskpb.FieldMask{
				Paths: []string{"lifecycle_rules"},
			},
		}))
	})

	It("builds a combined update mask in field evaluation order", func() {
		result, err := patchYcBucketAttrs(cloud.BucketPatch{
			StorageClass: helpers.PatchTo(vedro.BucketStorageClassIce),
			Labels: helpers.PatchTo(map[string]string{
				"env": "prod",
			}),
			Versioning: helpers.PatchTo(&vedro.BucketVersioning{
				Enabled: true,
			}),
			Lifecycle: helpers.PatchTo(&vedro.BucketLifecycle{}),
		}, "example")

		Expect(err).NotTo(HaveOccurred())
		Expect(result.UpdateMask.Paths).To(Equal([]string{
			"default_storage_class",
			"tags",
			"versioning",
			"lifecycle_rules",
		}))
		Expect(result.DefaultStorageClass).To(Equal("ICE"))
		Expect(result.Versioning).To(Equal(storageapi.Versioning_VERSIONING_ENABLED))
		Expect(result.Tags).To(Equal([]*storageapi.Tag{
			{
				Key:   "env",
				Value: "prod",
			},
		}))
		Expect(result.LifecycleRules).To(Equal([]*storageapi.LifecycleRule{}))
	})
})
