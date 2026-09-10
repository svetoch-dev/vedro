package controller

import (
	"context"

	vedro "github.com/svetoch-dev/vedro/api/v1alpha1"
	"github.com/svetoch-dev/vedro/internal/cloud"
)

type fakePrincipalAuthProvider struct {
}

func (f *fakePrincipalAuthProvider) EnsureAuthentication(
	ctx context.Context,
	principalAuth vedro.CloudPrincipalAuth,
	principal vedro.CloudPrincipal,
) (*cloud.PrincipalAuthResult, error) {
	return nil, nil
}

func (f *fakePrincipalAuthProvider) DeleteAuthentication(
	ctx context.Context,
	principalAuth vedro.CloudPrincipalAuth,
) error {
	return nil
}
