package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	// PrincipalAuthUnsupportedStaticCredentials indicates that the provider does not support static credentials.
	PrincipalAuthUnsupportedStaticCredentials UnsupportedFeatureReason = "PrincipalAuthUnsupportedStaticCredentials"
	// PrincipalAuthUnsupportedWorkloadIdentity indicates that the provider does not support workload identity.
	PrincipalAuthUnsupportedWorkloadIdentity UnsupportedFeatureReason = "PrincipalAuthUnsupportedWorkloadIdentity"
	// PrincipalAuthUnsupportedWorkloadIdentityKind indicates that workload identity is unavailable for this principal kind.
	PrincipalAuthUnsupportedWorkloadIdentityKind UnsupportedFeatureReason = "PrincipalAuthUnsupportedWorkloadIdentityKind"
	// PrincipalAuthUnsupportedStaticCredentialsKind indicates that static credentials are unavailable for this principal kind.
	PrincipalAuthUnsupportedStaticCredentialsKind UnsupportedFeatureReason = "PrincipalAuthUnsupportedStaticCredentialsKind"
)

// AuthObjectReference identifies a Kubernetes object by name within the CloudPrincipalAuth namespace.
type AuthObjectReference struct {
	// Name is the name of the referenced Kubernetes object.
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="name is immutable"
	// +kubebuilder:validation:MinLength=1
	Name string `json:"name"`
}

// NamespacedName identifies a Kubernetes object by name and namespace.
type NamespacedName struct {
	// Name is the name of the referenced Kubernetes object.
	Name string `json:"name"`
	// Namespace is the namespace of the referenced Kubernetes object.
	Namespace string `json:"namespace"`
}

// StaticCredentialsSpec configures credentials stored in a Kubernetes Secret.
type StaticCredentialsSpec struct {
	// SecretRef names the Kubernetes Secret in which static credentials are stored.
	//
	// Required when method is StaticCredentials.
	// Usually empty when method is WorkloadIdentity.
	SecretRef AuthObjectReference `json:"secretRef"`

	// DeletionPolicy controls what happens to the auth material and secret
	// when this Kubernetes object is deleted.
	//
	// +kubebuilder:validation:Enum=Delete;Retain
	// +kubebuilder:default:=Retain
	// +optional
	DeletionPolicy DeletionPolicy `json:"deletionPolicy,omitempty"`
}

// WorkloadIdentitySpec configures authentication through a Kubernetes ServiceAccount.
type WorkloadIdentitySpec struct {
	// ServiceAccountRef names the Kubernetes ServiceAccount to configure for workload identity.
	//
	// Required when method is WorkloadIdentity.
	// Usually empty when method is StaticCredentials.
	ServiceAccountRef AuthObjectReference `json:"serviceAccountRef"`

	// DeletionPolicy controls what happens to the auth material and serviceaccount annotation
	// when this Kubernetes object is deleted.
	//
	// +kubebuilder:validation:Enum=Delete;Retain
	// +kubebuilder:default:=Retain
	// +optional
	DeletionPolicy DeletionPolicy `json:"deletionPolicy,omitempty"`
}

// CloudPrincipalAuthSpec defines the desired authentication method for a CloudPrincipal.
// +kubebuilder:validation:XValidation:rule="self.method != 'StaticCredentials' || (has(self.staticCredentials) && !has(self.workloadIdentity))",message="staticCredentials must be set and workloadIdentity must not be set when method is StaticCredentials"
// +kubebuilder:validation:XValidation:rule="self.method != 'WorkloadIdentity' || (has(self.workloadIdentity) && !has(self.staticCredentials))",message="workloadIdentity must be set and staticCredentials must not be set when method is WorkloadIdentity"
type CloudPrincipalAuthSpec struct {

	// PrincipalRef references the CloudPrincipal for which authentication is configured.
	//
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="principalRef is immutable"
	PrincipalRef PrincipalReference `json:"principalRef"`

	// Method selects the authentication method.
	//
	// +kubebuilder:validation:Enum=StaticCredentials;WorkloadIdentity
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="method is immutable"
	Method AuthMethod `json:"method"`

	// StaticCredentials configures the Secret used when Method is StaticCredentials.
	//
	// +optional
	StaticCredentials *StaticCredentialsSpec `json:"staticCredentials,omitempty"`

	// WorkloadIdentity configures the ServiceAccount used when Method is WorkloadIdentity.
	//
	// +optional
	WorkloadIdentity *WorkloadIdentitySpec `json:"workloadIdentity,omitempty"`
}

// CloudPrincipalAuthProperties records the authentication resources created or configured by the controller.
type CloudPrincipalAuthProperties struct {
	// Method is the authentication method applied to the cloud principal.
	Method AuthMethod `json:"method"`

	// CredentialsId is the provider-side identifier of the credentials.
	CredentialsId string `json:"credentialsId"`
	// PrincipalId is the provider-side identifier of the cloud principal.
	PrincipalId string `json:"principalId"`

	// ServiceAccountRef identifies the configured Kubernetes ServiceAccount, if any.
	//
	// +optional
	ServiceAccountRef *NamespacedName `json:"serviceAccountRef,omitempty"`

	// SecretRef identifies the Kubernetes Secret containing static credentials, if any.
	//
	// +optional
	SecretRef *NamespacedName `json:"secretRef,omitempty"`

	CreatedAt metav1.Time `json:"createdAt"`
}

// CloudPrincipalAuthStatus defines the observed state of a CloudPrincipalAuth.
type CloudPrincipalAuthStatus struct {
	// Applied records the authentication method and resources applied by the controller.
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
	// ObservedProvider is the ProviderConfig used for the last successful reconciliation.
	//
	// +optional
	ObservedProvider string `json:"observedProvider,omitempty"`
	// UnsupportedFeatures lists requested features that the selected provider does not support.
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

	// Spec defines the desired state of the CloudPrincipalAuth.
	Spec CloudPrincipalAuthSpec `json:"spec,omitempty"`

	// Status defines the observed state of the CloudPrincipalAuth.
	//
	// +optional
	Status CloudPrincipalAuthStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// CloudPrincipalAuthList contains a list of CloudPrincipalAuth resources.
type CloudPrincipalAuthList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	// Items contains the CloudPrincipalAuth resources in this list.
	Items []CloudPrincipalAuth `json:"items"`
}
