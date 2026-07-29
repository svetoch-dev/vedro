package yc

import (
	vedro "github.com/svetoch-dev/vedro/api/v1alpha1"
	"github.com/svetoch-dev/vedro/internal/cloud"
	cloudtest "github.com/svetoch-dev/vedro/internal/cloud/test"
)

// Provider-agnostic EnsureBucket/DeleteBucket behaviour lives in the shared
// cloudtest package; only GCP specifics are configured here.
var _ = cloudtest.BucketProviderTests(cloudtest.Config{
	Location:                "ru-central1",
	NormalizedLocation:      "ru-central1",
	OtherLocation:           "kz1",
	OtherNormalizedLocation: "kz1",
	ProviderConfigType:      vedro.ProviderTypeYandexCloud,
	BucketCaps:              (&Provider{}).Capabilities().Bucket,
	NewBucket: func(api cloud.BucketAPI) cloud.BucketProvider {
		return &Bucket{api: api}
	},
})

var _ = cloudtest.BucketValidationTests(cloudtest.Config{
	Location:                "ru-central1",
	NormalizedLocation:      "ru-central1",
	OtherLocation:           "kz1",
	OtherNormalizedLocation: "kz1",
	ProviderConfigType:      vedro.ProviderTypeYandexCloud,
	NewBucket: func(api cloud.BucketAPI) cloud.BucketProvider {
		return &Bucket{api: api}
	},
})
