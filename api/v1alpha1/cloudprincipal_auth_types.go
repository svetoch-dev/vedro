package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	PrincipalAuthUnsupportedStaticCredentials     UnsupportedFeatureReason = "PrincipalAuthUnsupportedStaticCredentials"
	PrincipalAuthUnsupportedWorkloadIdentity      UnsupportedFeatureReason = "PrincipalAuthUnsupportedWorkloadIdentity"
	PrincipalAuthUnsupportedWorkloadIdentityKind  UnsupportedFeatureReason = "PrincipalAuthUnsupportedWorkloadIdentityKind"
	PrincipalAuthUnsupportedStaticCredentialsKind UnsupportedFeatureReason = "PrincipalAuthUnsupportedStaticCredentialsKind"
)

type StaticCredentialsSpec struct {
	// SecretRef references a Kubernetes Secret where static credentials should be placed.
	//
	// Required when method is StaticCredentials.
	// Usually empty when method is WorkloadIdentity.
	//
	// +optional
	SecretRef *corev1.SecretReference `json:"secretRef,omitempty"`

	// DeletionPolicy controls what happens to the auth material and secret
	// when this Kubernetes object is deleted.
	//
	// +kubebuilder:validation:Enum=Delete;Retain
	// +kubebuilder:default:=Retain
	// +optional
	DeletionPolicy DeletionPolicy `json:"deletionPolicy,omitempty"`
}

type WorkloadIdentitySpec struct {
	// ServiceAccountRef references a Kubernetes ServiceAccount to which
	// needed annotations should be added
	//
	// Required when method is WorkloadIdentity.
	// Usually empty when method is StaticCredentials.
	//
	// +optional
	ServiceAccountRef *ServiceAccountReference `json:"serviceAccountRef,omitempty"`

	// DeletionPolicy controls what happens to the auth material and serviceaccount annotation
	// when this Kubernetes object is deleted.
	//
	// +kubebuilder:validation:Enum=Delete;Retain
	// +kubebuilder:default:=Retain
	// +optional
	DeletionPolicy DeletionPolicy `json:"deletionPolicy,omitempty"`
}

type CloudPrincipalAuthSpec struct {

	// CloudPrincipal reference the cloud principal for whom to create
	// auth material
	//
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="principalRef is immutable"
	PrincipalRef PrincipalReference `json:"providerRef"`

	// +kubebuilder:validation:Enum=StaticCredentials;WorkloadIdentity
	Method AuthMethod `json:"method"`

	// +optional
	StaticCredentials *StaticCredentialsSpec `json:"staticCredentials,omitempty"`

	// +optional
	WorkloadIdentity *WorkloadIdentitySpec `json:"workloadIdentity,omitempty"`
}

type CloudPrincipalAuthProperties struct {
	Method AuthMethod `json:"method"`

	CredentialsId string `json:"credentialsId"`

	// +optional
	ServiceAccountRef *ServiceAccountReference `json:"serviceAccountRef,omitempty"`

	// +optional
	SecretRef *corev1.SecretReference `json:"secretRef,omitempty"`
}

type CloudPrincipalAuthStatus struct {
	// Applied - what has been already applied by this controller
	//
	// +optional
	Applied *CloudPrincipalAuthProperties `json:"applied,omitempty"`
	// Conditions represent the latest available observations of the CloudPrincipalAuth state.
	//
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`
	// ObservedGeneration is the latest metadata.generation observed by the controller.
	//
	// +optional
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`
	// Provider used for this CloudPrincipalAuth
	//
	// +optional
	ObservedProvider string `json:"observedProvider,omitempty"`
	// List of unsupported features set on CloudPrincipalAuth resource
	//
	// +optional
	UnsupportedFeatures []UnsupportedFeature `json:"unsupported,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Namespaced,categories=vedro
// +kubebuilder:printcolumn:name="Principal",type=string,JSONPath=`.spec.principalRef.name`
// +kubebuilder:printcolumn:name="Ready",type=string,JSONPath=`.status.conditions[?(@.type=="Ready")].status`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`

// CloudPrincipalAuth is the Schema for the cloudprincipalauths API.
type CloudPrincipalAuth struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Spec defines the desired state of the CloudPrincipal.
	Spec CloudPrincipalAuthSpec `json:"spec,omitempty"`

	// Status defines the observed state of the CloudPrincipal.
	//
	// +optional
	Status CloudPrincipalAuthStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// CloudPrincipalList contains a list of CloudPrincipal resources.
type CloudPrincipalAuthList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	// Items contains the CloudPrincipal resources in this list.
	Items []CloudPrincipalAuth `json:"items"`
}
