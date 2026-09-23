package gcp

import (
	"context"
	"errors"
	"fmt"
	"time"

	vedro "github.com/svetoch-dev/vedro/api/v1alpha1"
	"github.com/svetoch-dev/vedro/internal/cloud"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

const gcpKeyPropagationGracePeriod = time.Second * 60

type PrincipalAuth struct {
	api cloud.PrincipalAPI
}

func (o *PrincipalAuth) EnsureAuthentication(
	ctx context.Context,
	principalAuth vedro.CloudPrincipalAuth,
	principal vedro.CloudPrincipal,
) (*cloud.PrincipalAuthResult, error) {
	spec := principalAuth.Spec
	status := principalAuth.Status

	logger := log.FromContext(ctx)
	credentialsID := ""

	if status.Applied != nil {
		credentialsID = status.Applied.CredentialsId
	}

	authSetup := cloud.PrincipalAuthSetup{
		Method:           spec.Method,
		ServiceAccountID: principal.Status.ExternalId,
		CredentialsID:    credentialsID,
	}

	if spec.Method == vedro.AuthMethodWorkloadIdentity {
		authSetup.K8sServiceAccount = &vedro.NamespacedName{
			Name:      spec.WorkloadIdentity.ServiceAccountRef.Name,
			Namespace: principalAuth.Namespace,
		}
	}

	if authSetup.CredentialsID == "" {
		logger.Info("initial credentials creation")
		result, err := o.api.CreatePrincipalAuth(ctx, authSetup)
		if err != nil {
			return nil, fmt.Errorf("create auth failed %w", err)
		}

		return result, nil

	}

	result, err := o.api.GetPrincipalAuth(ctx, authSetup)

	// We need this check for StaticCreds because gcp
	// uses eventually consitant mechanisms
	// to store key state. So any operation with
	// the key performed immediately after creation
	// can return errors. To mitigate this we just
	// check against a GracePeriod const
	withinGracePeriod := spec.Method == vedro.AuthMethodStaticCredentials &&
		time.Since(status.Applied.CreatedAt.Time) < gcpKeyPropagationGracePeriod

	if errors.Is(err, cloud.ErrAuthNotFound) {
		if withinGracePeriod {
			return &cloud.PrincipalAuthResult{
				Method:        authSetup.Method,
				CredentialsID: authSetup.CredentialsID,
			}, nil
		}
		logger.Info("credentials not found")
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
		return fmt.Errorf("no credentialsId in CloudPrincipalAuth.Status")
	}
	authSetup := cloud.PrincipalAuthSetup{
		Method:           principalAuth.Status.Applied.Method,
		ServiceAccountID: principalAuth.Status.Applied.PrincipalId,
		CredentialsID:    principalAuth.Status.Applied.CredentialsId,
	}
	return o.api.DeletePrincipalAuth(ctx, authSetup)
}
