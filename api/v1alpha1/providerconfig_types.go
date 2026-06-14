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

// If StaticCredentials is set credentialsSecretRef should also be set
// +kubebuilder:validation:XValidation:rule="self.method != 'StaticCredentials' || has(self.credentialsSecretRef)",message="credentialsSecretRef is required when method is StaticCredentials"
// If WorkloadIdentity is set credentialsSecretRef should not be set
// +kubebuilder:validation:XValidation:rule="self.method != 'WorkloadIdentity' || !has(self.credentialsSecretRef)",message="credentialsSecretRef must not be set when method is WorkloadIdentity"
type ProviderConfigSpec struct {
	// Type is the cloud provider type.
	//
	// +kubebuilder:validation:Enum=GCP;Yandex
	Type ProviderType `json:"type"`

	// ProjectID identifies the cloud project/account/folder.
	//
	// For GCP this can be the project ID.
	// For Yandex this can be the folder ID or cloud ID, depending on your design.
	//
	// +kubebuilder:validation:MinLength=1
	ProjectId string `json:"projectId"`

	Region string `json:"region"`

	// Method describes the authentication method.
	//
	// +kubebuilder:validation:Enum=StaticCredentials;WorkloadIdentity
	Method AuthMethod `json:"method"`

	// CredentialsSecretRef references a Kubernetes Secret containing provider credentials.
	//
	// Required when method is StaticCredentials.
	// Usually empty when method is WorkloadIdentity.
	//
	// +optional
	CredentialsSecretRef *corev1.SecretReference `json:"credentialsSecretRef,omitempty"`
}

// ProviderConfigStatus defines the observed provider configuration state.
type ProviderConfigStatus struct {
	// Conditions represent the latest available observations of the ProviderConfig state.
	//
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Cluster,categories=vedro
// +kubebuilder:printcolumn:name="Type",type=string,JSONPath=`.spec.type`
// +kubebuilder:printcolumn:name="Method",type=string,JSONPath=`.spec.method`
// +kubebuilder:printcolumn:name="Region",type=string,JSONPath=`.spec.region`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`

// ProviderConfig is the Schema for the providerconfigs API.
type ProviderConfig struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec ProviderConfigSpec `json:"spec,omitempty"`

	// +optional
	Status ProviderConfigStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// ProviderConfigList contains a list of ProviderConfig.
type ProviderConfigList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []ProviderConfig `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ProviderConfig{}, &ProviderConfigList{})
}
