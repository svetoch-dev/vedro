package v1alpha1

type ProviderType string

const (
	ProviderTypeGCP         ProviderType = "gcp"
	ProviderTypeYandexCloud ProviderType = "yc"
)

type ProviderConfigReference struct {
	// Name is the name of the ProviderConfig.
	//
	// +kubebuilder:validation:MinLength=1
	Name string `json:"name"`
}

type BucketReference struct {
	// Name is the name of the Bucket.
	//
	// +kubebuilder:validation:MinLength=1
	Name string `json:"name"`

	// Namespace of the Bucket.
	//
	// +kubebuilder:validation:MinLength=1
	Namespace string `json:"namespace"`
}

type PrincipalReference struct {
	// Name is the name of the CloudPrincipal.
	//
	// +kubebuilder:validation:MinLength=1
	Name string `json:"name"`

	// Namespace of the CloudPrincipal.
	//
	// +kubebuilder:validation:MinLength=1
	Namespace string `json:"namespace"`
}

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

type UnsupportedFeatureReason string

type UnsupportedFeature struct {
	Field   string                   `json:"field"`
	Message string                   `json:"message"`
	Reason  UnsupportedFeatureReason `json:"reason"`
}
