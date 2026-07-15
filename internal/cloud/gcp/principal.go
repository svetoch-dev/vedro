package gcp

import (
	"context"
	"regexp"

	vedro "github.com/svetoch-dev/vedro/api/v1alpha1"
	"github.com/svetoch-dev/vedro/internal/cloud"
	"github.com/svetoch-dev/vedro/internal/helpers"
	"github.com/svetoch-dev/vedro/internal/validation"
)

var _ cloud.PrincipalProvider = (*Principal)(nil)

var gcpPrincipalNamePattern = regexp.MustCompile(`^[a-z][a-z0-9-]{4,28}[a-z0-9]$`)

type Principal struct {
	api cloud.PrincipalAPI
}

func (p *Principal) ValidatePrincipalSpec(principal vedro.CloudPrincipal) validation.ValidationResult {
	name := helpers.PrincipalNameFromCR(principal)

	v := validation.ValidateNameImmutability(
		principal.Spec.Name,
		principal.Status.ExternalName,
		principal.Name,
	)
	if !v.Valid {
		return v
	}

	if !gcpPrincipalNamePattern.MatchString(name) {
		return validation.Invalid(
			"principal name must be 6-30 characters, contain only lowercase letters, numbers, and dashes, start with a letter, and end with a letter or number",
		)
	}

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
