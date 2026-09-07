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
var ycServiceAccountRefPattern = regexp.MustCompile(
	`^[a-z](?:[-a-z0-9]{1,61}[a-z0-9]):[a-z](?:[-a-z0-9]{1,61}[a-z0-9])$`,
)

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

	if policy == vedro.PrincipalManagementPolicyManaged &&
		!ycServiceAccountNamePattern.MatchString(principalName) {
		return validation.Invalid(
			"Managed serviceAccount name must be 3-63 characters, contain only lowercase letters, numbers, and dashes, start with a letter, and end with a letter or number. Example: some-sa",
		)
	}

	if policy == vedro.PrincipalManagementPolicyReference &&
		!ycServiceAccountRefPattern.MatchString(principalName) {
		return validation.Invalid(
			"referenced serviceAccount must match <folder-name>:<service-account-name>: Example prod:some-sa",
		)
	}

	return validation.Valid()
}

func validateYcUser(principal vedro.CloudPrincipal) validation.ValidationResult {
	kind := principal.Spec.Kind
	principalName := helpers.PrincipalNameFromCR(principal)

	if kind != vedro.PrincipalKindUser {
		return validation.Valid()
	}

	if !validation.EmailPattern.MatchString(principalName) {
		return validation.Invalid(
			fmt.Sprintf("Referenced %s names must be valid emails. Example: user@example.com", kind),
		)
	}
	return validation.Valid()
}

func (p *Principal) ValidatePrincipalSpec(principal vedro.CloudPrincipal) validation.ValidationResult {
	status := principal.Status
	principalName := helpers.PrincipalNameFromCR(principal)

	if principal.Spec.ManagementPolicy == vedro.PrincipalManagementPolicyManaged {
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
	}

	v := validateYcServiceAccount(principal)

	if !v.Valid {
		return v
	}

	v = validateYcUser(principal)

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
	if principalSetup.Policy == vedro.PrincipalManagementPolicyManaged &&
		errors.Is(err, cloud.ErrPrincipalNotFound) {
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
