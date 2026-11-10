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
	// Name is the referenced Kubernetes object name in the CloudPrincipalAuth namespace. The
	// name is immutable.
	//
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
	// SecretRef names the output Secret in the CloudPrincipalAuth namespace. The controller
	// stores a GCP service account JSON key under credentials.json, or Yandex Cloud S3
	// credentials under AWS_ACCESS_KEY_ID and AWS_SECRET_ACCESS_KEY. The Secret is created and
	// managed by the controller; it is not an input credential Secret.
	SecretRef AuthObjectReference `json:"secretRef"`

	// DeletionPolicy controls cleanup when CloudPrincipalAuth is deleted. Delete revokes the
	// cloud credentials and allows the managed Secret to be deleted. Retain preserves both the
	// credentials and Secret, removing the Secret's owner references. Defaults to Retain.
	//
	// +kubebuilder:validation:Enum=Delete;Retain
	// +kubebuilder:default:=Retain
	// +optional
	DeletionPolicy DeletionPolicy `json:"deletionPolicy,omitempty"`
}

// WorkloadIdentitySpec configures authentication through a Kubernetes ServiceAccount.
type WorkloadIdentitySpec struct {
	// ServiceAccountRef names an existing Kubernetes ServiceAccount in the CloudPrincipalAuth
	// namespace. For GCP, the controller grants workload identity access to the cloud service
	// account and configures the Kubernetes ServiceAccount annotation. Workload identity for
	// principals is currently unsupported by the Yandex Cloud provider.
	ServiceAccountRef AuthObjectReference `json:"serviceAccountRef"`

	// DeletionPolicy controls cleanup when CloudPrincipalAuth is deleted. Delete removes the
	// cloud authentication binding and managed ServiceAccount annotation. Retain preserves the
	// configuration. The Kubernetes ServiceAccount itself is not deleted. Defaults to Retain.
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

	// PrincipalRef identifies the CloudPrincipal for which authentication is configured,
	// including its explicit namespace. The reference is immutable. The principal must use
	// managementPolicy Managed. Both providers currently support authentication only for
	// ServiceAccount principals.
	//
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="principalRef is immutable"
	PrincipalRef PrincipalReference `json:"principalRef"`

	// Method selects StaticCredentials to issue cloud credentials into a Secret, or
	// WorkloadIdentity to configure a Kubernetes ServiceAccount. This field is immutable. Only
	// the matching configuration block may be set.
	//
	// +kubebuilder:validation:Enum=StaticCredentials;WorkloadIdentity
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="method is immutable"
	Method AuthMethod `json:"method"`

	// StaticCredentials configures credential issuance and the output Secret. Required when
	// method is StaticCredentials and forbidden when method is WorkloadIdentity.
	//
	// +optional
	StaticCredentials *StaticCredentialsSpec `json:"staticCredentials,omitempty"`

	// WorkloadIdentity configures the Kubernetes ServiceAccount and cloud identity binding.
	// Required when method is WorkloadIdentity and forbidden when method is StaticCredentials.
	//
	// +optional
	WorkloadIdentity *WorkloadIdentitySpec `json:"workloadIdentity,omitempty"`
}

// CloudPrincipalAuthProperties records the authentication resources created or configured by the controller.
type CloudPrincipalAuthProperties struct {
	// Method is the authentication method applied to the cloud principal.
	Method AuthMethod `json:"method"`

	// CredentialsId is the provider-side identifier used to look up or revoke the issued
	// credentials or workload identity binding. Its format depends on the provider and method.
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

	// CreatedAt is the time the controller recorded creation of the current authentication
	// material. It is refreshed when static credentials are recreated and is used to allow for
	// GCP key propagation delays.
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

// +genclient
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Namespaced,categories=vedro
// +kubebuilder:printcolumn:name="Principal",type=string,JSONPath=`.spec.principalRef.name`
// +kubebuilder:printcolumn:name="Ready",type=string,JSONPath=`.status.conditions[?(@.type=="Ready")].status`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`

// CloudPrincipalAuth provisions authentication for a managed CloudPrincipal and publishes
// or configures its Kubernetes authentication resources.
type CloudPrincipalAuth struct {
	// TypeMeta identifies the API version and kind of this resource.
	metav1.TypeMeta `json:",inline"`

	// ObjectMeta contains Kubernetes identity, labels, annotations, and ownership metadata.
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Spec defines the desired state of the CloudPrincipalAuth.
	Spec CloudPrincipalAuthSpec `json:"spec,omitempty"`

	// Status records the observed state of the CloudPrincipalAuth and is maintained by the
	// controller.
	//
	// +optional
	Status CloudPrincipalAuthStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// CloudPrincipalAuthList contains a list of CloudPrincipalAuth resources.
type CloudPrincipalAuthList struct {
	// TypeMeta identifies the API version and kind of this resource.
	metav1.TypeMeta `json:",inline"`

	// ListMeta contains Kubernetes list metadata, including pagination and resource version.
	metav1.ListMeta `json:"metadata,omitempty"`

	// Items contains the CloudPrincipalAuth resources in this list.
	Items []CloudPrincipalAuth `json:"items"`
}
