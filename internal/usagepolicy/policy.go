package usagepolicy

import (
	"fmt"
	"regexp"
	"slices"

	vedro "github.com/svetoch-dev/vedro/api/v1alpha1"
	"github.com/svetoch-dev/vedro/internal/helpers"
)

type Decision struct {
	Allowed bool
	Message string
}

func Allowed() Decision {
	return Decision{
		Allowed: true,
		Message: "Allowed",
	}
}

func Restricted(message string) Decision {
	return Decision{
		Allowed: false,
		Message: message,
	}
}

func isNamespaceAllowed(namespace string, allowedNamespaces vedro.AllowedNamespacesSpec) Decision {
	if allowedNamespaces.All {
		return Allowed()
	}

	if slices.Contains(allowedNamespaces.Names, namespace) {
		return Allowed()
	}

	return Restricted(
		fmt.Sprintf("Namespace %s is not allowed by ProviderConfig usagePolicy", namespace),
	)
}

func isNameAllowed(name string, patterns []string) Decision {
	for _, p := range patterns {
		regex := regexp.MustCompile(p)
		if regex.MatchString(name) {
			return Allowed()
		}
	}
	return Restricted(
		fmt.Sprintf("Name %s is not allowed by ProviderConfig usagePolicy", name),
	)
}

func CheckBucket(spec vedro.UsagePolicySpec, bucket vedro.Bucket) Decision {
	d := isNamespaceAllowed(bucket.Namespace, spec.AllowedNamespaces)

	if !d.Allowed {
		return d
	}

	var bucketName string
	if bucket.DeletionTimestamp.IsZero() {
		bucketName = helpers.BucketNameFromCR(bucket)
	} else {
		bucketName = helpers.BucketNameForDelete(bucket)
	}

	d = isNameAllowed(bucketName, spec.BucketPolicy.AllowedNamePatterns)

	if !d.Allowed {
		return d
	}

	return Allowed()
}

func CheckPrincipal(spec vedro.UsagePolicySpec, principal vedro.CloudPrincipal) Decision {
	d := isNamespaceAllowed(principal.Namespace, spec.AllowedNamespaces)

	if !d.Allowed {
		return d
	}

	var principalName string
	var kind vedro.PrincipalKind
	if principal.DeletionTimestamp.IsZero() {
		principalName = helpers.PrincipalNameFromCR(principal)
		kind = principal.Spec.Kind
	} else {
		principalName = helpers.PrincipalNameForDelete(principal)
		kind = principal.Status.Kind
	}

	if !slices.Contains(spec.PrincipalPolicy.AllowedKinds, kind) {
		return Restricted(
			fmt.Sprintf("Kind %s is not allowed by ProviderConfig usagePolicy", principal.Spec.Kind),
		)
	}

	if principal.Spec.ManagementPolicy == vedro.PrincipalManagementPolicyManaged {
		if !spec.PrincipalPolicy.AllowManaged {
			return Restricted("Managed CloudPrincipals are not allowed by ProviderConfig usagePolicy")
		}

		d = isNameAllowed(principalName, spec.PrincipalPolicy.AllowedNamePatterns)

		if !d.Allowed {
			return d
		}
	}

	if principal.Spec.ManagementPolicy == vedro.PrincipalManagementPolicyReference {
		if !spec.PrincipalPolicy.AllowReferences {
			return Restricted("Referenced CloudPrincipals are not allowed by ProviderConfig usagePolicy")
		}

		if kind != vedro.PrincipalKindAllUsers {
			d = isNameAllowed(principalName, spec.PrincipalPolicy.AllowedReferencePatterns)

			if !d.Allowed {
				return d
			}
		}
	}

	return Allowed()
}
