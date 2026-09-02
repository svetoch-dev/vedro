package gcp

import (
	"context"
	"errors"
	"fmt"
	"regexp"

	vedro "github.com/svetoch-dev/vedro/api/v1alpha1"
	"github.com/svetoch-dev/vedro/internal/cloud"
	"github.com/svetoch-dev/vedro/internal/helpers"
	"github.com/svetoch-dev/vedro/internal/validation"
)

var gcpServiceAccountNamePattern = regexp.MustCompile(`^[a-z][a-z0-9-]{4,28}[a-z0-9]$`)
var emailPattern = regexp.MustCompile(`^[^@\s]+@[^@\s]+\.[^@\s]+$`)

type Principal struct {
	api cloud.PrincipalAPI
}

func (p *Principal) ValidatePrincipalSpec(principal vedro.CloudPrincipal) validation.ValidationResult {
	status := principal.Status
	kind := principal.Spec.Kind

	name := helpers.PrincipalNameFromCR(principal)

	v := validation.ValidateNameImmutability(
		name,
		status.ExternalName,
		// We use name as obj name
		// because Spec.Reference.Name and Spec.Managed.Name
		// are mandatory
		name,
	)
	if !v.Valid {
		return v
	}

	if kind == vedro.PrincipalKindServiceAccount && !gcpServiceAccountNamePattern.MatchString(name) {
		return validation.Invalid(
			"service account name must be 6-30 characters, contain only lowercase letters, numbers, and dashes, start with a letter, and end with a letter or number",
		)
	}

	if (kind == vedro.PrincipalKindUser || kind == vedro.PrincipalKindGroup) &&
		!emailPattern.MatchString(name) {
		return validation.Invalid(
			fmt.Sprintf("%s names must be valid email addresses without the GCP IAM prefix (for example, user@example.com)", kind),
		)
	}

	return validation.Valid()
}

func (p *Principal) EnsurePrincipal(
	ctx context.Context,
	principal vedro.CloudPrincipal,
) (*cloud.PrincipalAttrs, error) {
	principalName := helpers.PrincipalNameFromCR(principal)
	principalSetup := cloud.PrincipalSetup{
		Kind:   principal.Spec.Kind,
		Name:   principalName,
		Policy: principal.Spec.ManagementPolicy,
	}

	attrs, err := p.api.GetPrincipal(ctx, principalSetup)
	if errors.Is(err, cloud.ErrPrincipalNotFound) {
		attrs, err := p.api.CreatePrincipal(ctx, principalSetup)
		if err != nil {
			return nil, fmt.Errorf("create principal %q: %w", principalName, err)
		}
		return attrs, nil
	}

	if err != nil {
		return nil, fmt.Errorf("get principal attrs %q: %w", principalName, err)
	}

	return attrs, nil
}

func (p *Principal) DeletePrincipal(
	ctx context.Context,
	principal vedro.CloudPrincipal,
) error {
	principalSetup := cloud.PrincipalSetup{
		Kind:   principal.Spec.Kind,
		Name:   helpers.PrincipalNameForDelete(principal),
		Policy: principal.Spec.ManagementPolicy,
	}
	return p.api.DeletePrincipal(ctx, principalSetup)
}
