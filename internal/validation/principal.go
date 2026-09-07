package validation

import (
	"regexp"

	vedro "github.com/svetoch-dev/vedro/api/v1alpha1"
)

var EmailPattern = regexp.MustCompile(`^[^@\s]+@[^@\s]+\.[^@\s]+$`)

func ValidatePrincipalAllUsers(principal vedro.CloudPrincipal) ValidationResult {
	if principal.Spec.Kind != vedro.PrincipalKindAllUsers {
		return Valid()
	}

	if principal.Spec.Reference != nil {
		return Invalid("Reference can not be set when Kind is AllUsers")
	}

	return Valid()
}
