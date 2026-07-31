package gcp

import (
	"github.com/svetoch-dev/vedro/internal/cloud"
	cloudtest "github.com/svetoch-dev/vedro/internal/cloud/test"
)

var _ = cloudtest.BucketAccessProviderTests(cloudtest.Config{
	Location: "us-central1",
	NewBucketAccess: func(api cloud.BucketAPI) cloud.BucketAccessProvider {
		return &BucketAccess{api: api}
	},
})
