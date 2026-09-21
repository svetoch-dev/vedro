package gcp

import (
	"github.com/svetoch-dev/vedro/internal/cloud"
	cloudtest "github.com/svetoch-dev/vedro/internal/cloud/test"
)

var _ = cloudtest.PrincipalAuthProviderTests(cloudtest.Config{
	NewPrincipalAuth: func(api cloud.PrincipalAPI) cloud.PrincipalAuthProvider {
		return &PrincipalAuth{api: api}
	},
	SupportsWorkloadIdentity: true,
})
