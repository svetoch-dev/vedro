package yc

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

var ycServiceAccountNamePattern = regexp.MustCompile(`^[a-z][a-z0-9-]{1,61}[a-z0-9]$`)
var ycPrincipalIDPattern = regexp.MustCompile(`^aje[a-z0-9]+$`)

type Principal struct {
	api cloud.PrincipalAPI
}

func validateYcServiceAccount(principal vedro.CloudPrincipal) validation.ValidationResult {
	kind := principal.Spec.Kind
	policy := principal.Spec.ManagementPolicy

	principalName := helpers.PrincipalNameFromCR(principal)

	if kind != vedro.PrincipalKindServiceAccount {
		return validation.Valid()
	}

	if policy == vedro.PrincipalManagementPolicyManaged && !ycServiceAccountNamePattern.MatchString(principalName) {
		return validation.Invalid(
			"Managed serviceAccount name must be 3-63 characters, contain only lowercase letters, numbers, and dashes, start with a letter, and end with a letter or number. Example: some-sa",
		)
	}

	if policy == vedro.PrincipalManagementPolicyReference {
		if !ycPrincipalIDPattern.MatchString(principalName) {
			return validation.Invalid(
				"Referenced serviceAccount names must be valid yc ids. Example: aje9sb6ffd2u12345678",
			)
		}
	}

	return validation.Valid()
}

func validateYcUserAndGroup(principal vedro.CloudPrincipal) validation.ValidationResult {
	kind := principal.Spec.Kind
	principalName := helpers.PrincipalNameFromCR(principal)

	if kind != vedro.PrincipalKindGroup && kind != vedro.PrincipalKindUser {
		return validation.Valid()
	}

	if kind == vedro.PrincipalKindGroup {
		return validation.Invalid(
			"Group kind is not yet supported for yc by this operator",
		)
	}

	if !ycPrincipalIDPattern.MatchString(principalName) {
		return validation.Invalid(
			fmt.Sprintf("Referenced %s names must be valid yc ids. Example: aje9sb6ffd2u12345678", kind),
		)
	}
	return validation.Valid()
}

func validateYcManagementPolicy(principal vedro.CloudPrincipal) validation.ValidationResult {
	kind := principal.Spec.Kind
	policy := principal.Spec.ManagementPolicy

	if policy == vedro.PrincipalManagementPolicyManaged &&
		kind != vedro.PrincipalKindServiceAccount {
		return validation.Invalid(fmt.Sprintf("%s cant be managed", kind))
	}

	return validation.Valid()
}

func (p *Principal) ValidatePrincipalSpec(principal vedro.CloudPrincipal) validation.ValidationResult {
	status := principal.Status
	principalName := helpers.PrincipalNameFromCR(principal)

	v := validation.ValidateNameImmutability(
		principalName,
		status.ExternalName,
		// We use principalName as obj name
		// because Spec.Reference.Name and Spec.Managed.Name
		// are mandatory
		principalName,
	)
	if !v.Valid {
		return v
	}

	v = validateYcManagementPolicy(principal)

	if !v.Valid {
		return v
	}

	v = validateYcServiceAccount(principal)

	if !v.Valid {
		return v
	}

	v = validateYcUserAndGroup(principal)

	if !v.Valid {
		return v
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
