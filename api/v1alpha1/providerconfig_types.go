/*
Copyright 2026.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// AllowedNamePatterns is a list of Go regular expressions. A name is allowed
// when at least one expression matches. Expressions are not implicitly anchored;
// use ^ and $ to match the entire name. An empty list allows no names.
type AllowedNamePatterns []string

// AllowedNamespacesSpec selects Kubernetes namespaces that may use a ProviderConfig.
type AllowedNamespacesSpec struct {
	// Names lists exact Kubernetes namespace names allowed to use this provider.
	// Must be empty when all is true. If all is false and names is empty,
	// no namespaces are allowed.
	//
	// +optional
	Names []string `json:"names,omitempty"`

	// All allows every Kubernetes namespace to use this provider.
	// Defaults to false and cannot be combined with a non-empty names list.
	//
	// +optional
	All bool `json:"all,omitempty"`
}

// BucketPolicySpec restricts the cloud buckets managed through a ProviderConfig.
type BucketPolicySpec struct {
	// AllowedNamePatterns contains Go regular expressions for cloud bucket names.
	// Checks spec.name, falling back to metadata.name when spec.name is empty.
	// During deletion, the recorded external name is used when available.
	// At least one pattern must match; an empty list allows no names.
	// Use ^ and $ to match the entire name.
	AllowedNamePatterns AllowedNamePatterns `json:"allowedNamePatterns"`
}

// PrincipalPolicySpec restricts managed and referenced cloud principals.
type PrincipalPolicySpec struct {
	// AllowedNamePatterns contains Go regular expressions for managed principal
	// names from spec.managed.name. During deletion, the recorded external name
	// is used when available. At least one pattern must match; an empty list
	// allows no names. Use ^ and $ to match the entire name.
	AllowedNamePatterns AllowedNamePatterns `json:"allowedNamePatterns"`

	// AllowedReferencePatterns contains Go regular expressions for referenced
	// principal names from spec.reference.name. Defaults to [".*"], allowing
	// any reference name. An empty list allows no reference names.
	// Use ^ and $ to match the entire name. AllUsers has no reference name
	// and is exempt from this check.
	//
	// +kubebuilder:default:={".*"}
	AllowedReferencePatterns AllowedNamePatterns `json:"allowedReferencePatterns"`

	// AllowManaged permits CloudPrincipals with managementPolicy Managed.
	// Defaults to true. Kind and name restrictions still apply.
	//
	// +kubebuilder:default:=true
	AllowManaged bool `json:"allowManaged"`

	// AllowReferences permits CloudPrincipals with managementPolicy Reference,
	// including AllUsers. Defaults to true. Kind and applicable reference-name
	// restrictions still apply.
	//
	// +kubebuilder:default:=true
	AllowReferences bool `json:"allowReferences"`

	// AllowedKinds lists permitted CloudPrincipal kinds for both management
	// policies. Defaults to ServiceAccount, Role, User, Group, and AllUsers.
	// An empty list allows no kinds. Provider capability restrictions still apply.
	// During deletion, the recorded status.kind is checked.
	//
	// +kubebuilder:default:={"ServiceAccount", "Role", "User", "Group", "AllUsers"}
	AllowedKinds []PrincipalKind `json:"allowedKinds"`
}

// UsagePolicySpec controls which resources may use a ProviderConfig.
type UsagePolicySpec struct {
	// AllowedNamespaces restricts the namespaces of Bucket, CloudPrincipal,
	// BucketAccess, and CloudPrincipalAuth resources using this provider.
	AllowedNamespaces AllowedNamespacesSpec `json:"allowedNamespaces"`

	// BucketPolicy restricts cloud bucket names managed through this provider.
	BucketPolicy BucketPolicySpec `json:"bucketPolicy"`

	// PrincipalPolicy restricts principal kinds, management policies, and names
	// used through this provider.
	PrincipalPolicy PrincipalPolicySpec `json:"principalPolicy"`
}

// ProviderConfigSpec defines cloud scope, operator authentication, and resource usage
// restrictions.
//
// +kubebuilder:validation:XValidation:rule="self.method != 'StaticCredentials' || has(self.credentialsSecretRef)",message="credentialsSecretRef is required when method is StaticCredentials"
// +kubebuilder:validation:XValidation:rule="self.method != 'WorkloadIdentity' || !has(self.credentialsSecretRef)",message="credentialsSecretRef must not be set when method is WorkloadIdentity"
type ProviderConfigSpec struct {
	// Type selects the cloud implementation: gcp for Google Cloud or yc for Yandex Cloud. This
	// field is immutable.
	//
	// +kubebuilder:validation:Enum=gcp;yc
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="type is immutable"
	Type ProviderType `json:"type"`

	// ProjectId is the Google Cloud project ID for gcp or the Yandex Cloud folder ID for yc.
	// It determines where managed cloud resources are created and is immutable.
	//
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="projectId is immutable"
	// +kubebuilder:validation:MinLength=1
	ProjectId string `json:"projectId"`

	// Region is the provider region, such as europe-west1 or ru-central1. It must be a
	// non-empty regional name. Bucket placement is selected separately by Bucket.spec.location.
	Region string `json:"region"`

	// Method selects how the operator authenticates to the provider. StaticCredentials reads a
	// service account key from credentialsSecretRef. WorkloadIdentity uses the operator's
	// ambient Google Cloud credentials or the Yandex Cloud instance service account.
	//
	// +kubebuilder:validation:Enum=StaticCredentials;WorkloadIdentity
	Method AuthMethod `json:"method"`

	// CredentialsSecretRef identifies an existing Kubernetes Secret by name and namespace.
	// Required for StaticCredentials and forbidden for WorkloadIdentity. The Secret must
	// contain a key entry with a Google Cloud service account JSON key for gcp or a Yandex
	// Cloud authorized service account key in JSON format for yc.
	//
	// +optional
	CredentialsSecretRef *corev1.SecretReference `json:"credentialsSecretRef,omitempty"`

	// UsagePolicy defines namespace and resource restrictions for this provider.
	// Resources must satisfy all applicable restrictions to reconcile.
	UsagePolicy UsagePolicySpec `json:"usagePolicy"`
}

// ProviderConfigStatus defines the observed provider configuration state.
type ProviderConfigStatus struct {
	// Conditions represent the latest available observations of the ProviderConfig state.
	//
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`

	// ObservedGeneration is the latest metadata.generation observed by the controller.
	//
	// +optional
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`
}

// +genclient
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Cluster,categories=vedro
// +kubebuilder:printcolumn:name="Type",type=string,JSONPath=`.spec.type`
// +kubebuilder:printcolumn:name="Method",type=string,JSONPath=`.spec.method`
// +kubebuilder:printcolumn:name="Region",type=string,JSONPath=`.spec.region`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`

// ProviderConfig is a cluster-scoped resource defining cloud connectivity and policies for
// resources that use it.
type ProviderConfig struct {
	// TypeMeta identifies the API version and kind of this resource.
	metav1.TypeMeta `json:",inline"`

	// ObjectMeta contains Kubernetes identity, labels, annotations, and ownership metadata.
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Spec defines the desired state of the ProviderConfig.
	Spec ProviderConfigSpec `json:"spec,omitempty"`

	// Status records the observed state of the ProviderConfig and is maintained by the
	// controller.
	//
	// +optional
	Status ProviderConfigStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// ProviderConfigList contains a list of ProviderConfig.
type ProviderConfigList struct {
	// TypeMeta identifies the API version and kind of this resource.
	metav1.TypeMeta `json:",inline"`

	// ListMeta contains Kubernetes list metadata, including pagination and resource version.
	metav1.ListMeta `json:"metadata,omitempty"`

	// Items contains the ProviderConfig resources in this list.
	Items []ProviderConfig `json:"items"`
}
