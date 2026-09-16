package capabilities

import (
	vedro "github.com/svetoch-dev/vedro/api/v1alpha1"
	"github.com/svetoch-dev/vedro/internal/cloud"
)

func ValidatePrincipalAuthCapabilities(
	caps cloud.PrincipalAuthCapabilities,
	spec vedro.CloudPrincipalAuthSpec,
	kind vedro.PrincipalKind,
) []vedro.UnsupportedFeature {
	var unsupported []vedro.UnsupportedFeature

	if spec.Method == vedro.AuthMethodStaticCredentials &&
		caps.StaticCredentials == false {
		unsupported = append(unsupported, unsupportedFeatures["MethodStaticCredentials"])

	}

	if spec.Method == vedro.AuthMethodWorkloadIdentity &&
		caps.WorkloadIdentity == false {
		unsupported = append(unsupported, unsupportedFeatures["MethodWorkloadIdentity"])
	}

	if caps.WorkloadIdentity == true {
		supported, ok := caps.WorkloadIdentityKinds[kind]
		if !ok || !supported {
			unsupported = append(unsupported, unsupportedFeatures["WorkloadIdentityKind"])
		}
	}

	if caps.StaticCredentials == true {
		supported, ok := caps.StaticCredentialsKinds[kind]
		if !ok || !supported {
			unsupported = append(unsupported, unsupportedFeatures["StaticCredentialsKind"])
		}
	}

	return unsupported
}
