package capabilities

import (
	"fmt"

	vedro "github.com/svetoch-dev/vedro/api/v1alpha1"
	"github.com/svetoch-dev/vedro/internal/cloud"
)

func ValidatePrincipalCapabilities(
	caps cloud.PrincipalCapabilities,
	spec vedro.CloudPrincipalSpec,
) []vedro.UnsupportedFeature {
	var unsupported []vedro.UnsupportedFeature

	if spec.ManagementPolicy == vedro.PrincipalManagementPolicyManaged {
		supported, ok := caps.ManagedKinds[spec.Kind]
		if !ok || !supported {
			unsupported = append(unsupported, unsupportedFeatures[fmt.Sprintf("Managed%s", spec.Kind)])
		}

	}
	if spec.ManagementPolicy == vedro.PrincipalManagementPolicyReference {
		supported, ok := caps.ReferencedKinds[spec.Kind]
		if !ok || !supported {
			unsupported = append(unsupported, unsupportedFeatures[fmt.Sprintf("Referenced%s", spec.Kind)])
		}
	}

	return unsupported
}
