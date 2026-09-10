package yc

import (
	"context"

	vedro "github.com/svetoch-dev/vedro/api/v1alpha1"
	"github.com/svetoch-dev/vedro/internal/cloud"
)

type PrincipalAuth struct {
	api cloud.PrincipalAPI
}

func (o *PrincipalAuth) EnsureAuthentication(
	ctx context.Context,
	principalAuth vedro.CloudPrincipalAuth,
	principal vedro.CloudPrincipal,
) (*cloud.PrincipalAuthResult, error) {
	return nil, nil
}

func (o *PrincipalAuth) DeleteAuthentication(
	ctx context.Context,
	principalAuth vedro.CloudPrincipalAuth,
) error {
	return nil
}
