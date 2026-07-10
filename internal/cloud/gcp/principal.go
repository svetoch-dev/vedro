package gcp

import (
	"context"

	vedro "github.com/svetoch-dev/vedro/api/v1alpha1"
	"github.com/svetoch-dev/vedro/internal/cloud"
	"github.com/svetoch-dev/vedro/internal/validation"
)

var _ cloud.PrincipalProvider = (*Principal)(nil)

type Principal struct {
	api cloud.PrincipalAPI
}

func (p *Principal) ValidatePrincipalSpec(principal vedro.CloudPrincipal) validation.ValidationResult {
	return validation.Valid()
}

func (p *Principal) EnsurePrincipal(
	ctx context.Context,
	principal vedro.CloudPrincipal,
) (*cloud.PrincipalAttrs, error) {
	return nil, nil
}

func (p *Principal) DeletePrincipal(
	ctx context.Context,
	principal vedro.CloudPrincipal,
) error {
	return nil
}
