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

func validateGcpServiceAccount(principal vedro.CloudPrincipal) validation.ValidationResult {
	kind := principal.Spec.Kind
	policy := principal.Spec.ManagementPolicy

	principalName := helpers.PrincipalNameFromCR(principal)

	if kind != vedro.PrincipalKindServiceAccount {
		return validation.Valid()
	}

	if policy == vedro.PrincipalManagementPolicyManaged && !gcpServiceAccountNamePattern.MatchString(principalName) {
		return validation.Invalid(
			"Managed serviceAccount name must be 6-30 characters, contain only lowercase letters, numbers, and dashes, start with a letter, and end with a letter or number. Example: some-sa",
		)
	}

	if policy == vedro.PrincipalManagementPolicyReference {
		if !emailPattern.MatchString(principalName) {
			return validation.Invalid(
				"Referenced serviceAccount names must be valid email addresses without the serviceAccount: prefix. Example: some-sa@some-project.iam.gserviceaccount.com",
			)
		}
	}

	return validation.Valid()
}

func validateGcpUserAndGroup(principal vedro.CloudPrincipal) validation.ValidationResult {
	kind := principal.Spec.Kind
	principalName := helpers.PrincipalNameFromCR(principal)

	if kind != vedro.PrincipalKindGroup && kind != vedro.PrincipalKindUser {
		return validation.Valid()
	}

	if !emailPattern.MatchString(principalName) {
		return validation.Invalid(
			fmt.Sprintf("%s names must be valid email addresses without the GCP IAM prefix. Example: user@example.com", kind),
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

	v := validateGcpServiceAccount(principal)

	if !v.Valid {
		return v
	}

	v = validateGcpUserAndGroup(principal)

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
