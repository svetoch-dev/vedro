package yc

import (
	"context"
	"errors"
	"fmt"

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
	credentialsID := ""

	if principalAuth.Status.Applied != nil {
		credentialsID = principalAuth.Status.Applied.CredentialsId
	}

	authSetup := cloud.PrincipalAuthSetup{
		Method:           principalAuth.Spec.Method,
		ServiceAccountID: principal.Status.ExternalId,
		CredentialsID:    credentialsID,
	}

	if authSetup.CredentialsID == "" {
		result, err := o.api.CreatePrincipalAuth(ctx, authSetup)
		if err != nil {
			return nil, fmt.Errorf("create auth failed %w", err)
		}

		return result, nil

	}

	result, err := o.api.GetPrincipalAuth(ctx, authSetup)

	if errors.Is(err, cloud.ErrAuthNotFound) {
		result, err := o.api.CreatePrincipalAuth(ctx, authSetup)
		if err != nil {
			return nil, err
		}

		return result, nil

	}

	if err != nil {
		return nil, fmt.Errorf("create auth failed %w", err)
	}

	return result, nil
}

func (o *PrincipalAuth) DeleteAuthentication(
	ctx context.Context,
	principalAuth vedro.CloudPrincipalAuth,
) error {
	if principalAuth.Status.Applied == nil ||
		principalAuth.Status.Applied.CredentialsId == "" {
		return fmt.Errorf("No credentialsId in CloudPrincipalAuth.Status")
	}
	authSetup := cloud.PrincipalAuthSetup{
		Method:           principalAuth.Status.Applied.Method,
		ServiceAccountID: principalAuth.Status.Applied.PrincipalId,
		CredentialsID:    principalAuth.Status.Applied.CredentialsId,
	}
	return o.api.DeletePrincipalAuth(ctx, authSetup)
}
