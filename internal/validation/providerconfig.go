package validation

import (
	"fmt"
	"regexp"

	vedro "github.com/svetoch-dev/vedro/api/v1alpha1"
)

func ValidateUsagePolicy(policy vedro.UsagePolicySpec) ValidationResult {
	if policy.AllowedNamespaces.All && len(policy.AllowedNamespaces.Names) > 0 {
		return Invalid("AllowedNamespaces.All cant be used together with AllowedNamespaces.Names")
	}

	for _, k := range policy.PrincipalPolicy.AllowedKinds {
		if !k.Valid() {
			return Invalid(fmt.Sprintf("Kind %s is not a valid PrincipalKind", k))
		}
	}

	allowedNames := append(
		policy.BucketPolicy.AllowedNamePatterns,
		policy.PrincipalPolicy.AllowedNamePatterns...,
	)
	allowedNames = append(allowedNames, policy.PrincipalPolicy.AllowedReferencePatterns...)

	for _, p := range allowedNames {
		_, err := regexp.Compile(p)
		if err != nil {
			return Invalid(fmt.Sprintf("String %s does not compile to regex", p))
		}
	}

	return Valid()
}
