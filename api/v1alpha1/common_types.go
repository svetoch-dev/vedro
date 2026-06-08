package v1alpha1

type ProviderType string

const (
	ProviderTypeGCP    ProviderType = "GCP"
	ProviderTypeYandex ProviderType = "Yandex"
)

type AuthMethod string

const (
	AuthMethodStaticCredentials AuthMethod = "StaticCredentials"
	AuthMethodWorkloadIdentity  AuthMethod = "WorkloadIdentity"
)

type DeletionPolicy string

const (
	DeletionPolicyDelete DeletionPolicy = "Delete"
	DeletionPolicyRetain DeletionPolicy = "Retain"
)

type UnsupportedFeaturePolicy string

const (
	UnsupportedFeaturePolicyFail UnsupportedFeaturePolicy = "Fail"
	UnsupportedFeaturePolicyWarn UnsupportedFeaturePolicy = "Warn"
)

type ProviderConfigReference struct {
	// Name is the name of the ProviderConfig.
	//
	// +kubebuilder:validation:MinLength=1
	Name string `json:"name"`
}
