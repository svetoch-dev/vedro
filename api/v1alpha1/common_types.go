package v1alpha1

// ProviderType identifies a supported cloud implementation.
type ProviderType string

const (
	// ProviderTypeGCP selects Google Cloud Storage and IAM.
	ProviderTypeGCP ProviderType = "gcp"
	// ProviderTypeYandexCloud selects Yandex Cloud Object Storage and IAM.
	ProviderTypeYandexCloud ProviderType = "yc"
)

// ProviderConfigReference selects a cluster-scoped provider configuration.
type ProviderConfigReference struct {
	// Name is the name of the cluster-scoped ProviderConfig. No namespace is required.
	//
	// +kubebuilder:validation:MinLength=1
	Name string `json:"name"`
}

// BucketReference identifies a namespaced Bucket resource.
type BucketReference struct {
	// Name is the Kubernetes resource name of the Bucket, not its external cloud name.
	//
	// +kubebuilder:validation:MinLength=1
	Name string `json:"name"`

	// Namespace is the Kubernetes namespace containing the Bucket. It must be specified
	// explicitly, including for references within the same namespace.
	//
	// +kubebuilder:validation:MinLength=1
	Namespace string `json:"namespace"`
}

// PrincipalReference identifies a namespaced CloudPrincipal resource.
type PrincipalReference struct {
	// Name is the Kubernetes resource name of the CloudPrincipal, not its external cloud name
	// or identifier.
	//
	// +kubebuilder:validation:MinLength=1
	Name string `json:"name"`

	// Namespace is the Kubernetes namespace containing the CloudPrincipal. It must be
	// specified explicitly, including for references within the same namespace.
	//
	// +kubebuilder:validation:MinLength=1
	Namespace string `json:"namespace"`
}

// AuthMethod selects static credentials or workload identity authentication. ProviderConfig
// uses it for operator credentials; CloudPrincipalAuth uses it for application credentials.
type AuthMethod string

const (
	// AuthMethodStaticCredentials uses a provider key or issues static application
	// credentials, depending on the resource.
	AuthMethodStaticCredentials AuthMethod = "StaticCredentials"
	// AuthMethodWorkloadIdentity uses ambient operator credentials or configures a Kubernetes
	// workload identity, depending on the resource.
	AuthMethodWorkloadIdentity AuthMethod = "WorkloadIdentity"
)

// DeletionPolicy determines whether external resources or authentication material are
// deleted or retained when their Kubernetes resource is deleted.
type DeletionPolicy string

const (
	// DeletionPolicyDelete requests removal of the associated external resource or
	// authentication material.
	DeletionPolicyDelete DeletionPolicy = "Delete"
	// DeletionPolicyRetain preserves the associated external resource or authentication
	// material after Kubernetes resource deletion.
	DeletionPolicyRetain DeletionPolicy = "Retain"
)

// UnsupportedFeatureReason identifies a requested capability that the selected provider
// does not support.
type UnsupportedFeatureReason string

// UnsupportedFeature describes a spec field that cannot be applied by the selected
// provider.
type UnsupportedFeature struct {
	// Field is the path to the unsupported field in the requested spec, including a list index
	// when applicable.
	Field string `json:"field"`

	// Message explains why the selected provider cannot apply the requested feature.
	Message string `json:"message"`

	// Reason is a machine-readable code identifying the unsupported capability.
	Reason UnsupportedFeatureReason `json:"reason"`
}
