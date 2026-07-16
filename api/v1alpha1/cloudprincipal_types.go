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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type CloudPrincipalSpec struct {
	// Name is the real cloud provider bucket name.
	//
	// +kubebuilder:validation:MinLength=1
	// +optional
	Name string `json:"name,omitempty"`

	// ProviderRef references the ProviderConfig used to manage this bucket.
	//
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="providerRef is immutable"
	ProviderRef ProviderConfigReference `json:"providerRef"`

	// DeletionPolicy controls what happens to the external CloudPrincipal
	// when this Kubernetes object is deleted.
	//
	// +kubebuilder:validation:Enum=Delete;Retain
	// +kubebuilder:default:=Delete
	// +optional
	DeletionPolicy DeletionPolicy `json:"deletionPolicy,omitempty"`
}

// ProviderConfigStatus defines the observed provider configuration state.
type CloudPrincipalStatus struct {
	// ExternalName is the provider-side principal name.
	//
	// +optional
	ExternalName string `json:"externalName,omitempty"`

	// Provider used for this CloudPrincipal
	//
	// +optional
	ObservedProvider string `json:"observedProvider,omitempty"`

	// ObservedGeneration is the latest metadata.generation observed by the controller.
	//
	// +optional
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`

	// ExternalId is the provider-side principal id.
	//
	// +optional
	ExternalId string `json:"externalId,omitempty"`
	// Conditions represent the latest available observations of the ProviderConfig state.
	//
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Namespaced,categories=vedro
// +kubebuilder:printcolumn:name="Provider",type=string,JSONPath=`.spec.providerRef.name`
// +kubebuilder:printcolumn:name="Ready",type=string,JSONPath=`.status.conditions[?(@.type=="Ready")].status`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`

// ProviderConfig is the Schema for the providerconfigs API.
type CloudPrincipal struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec CloudPrincipalSpec `json:"spec,omitempty"`

	// +optional
	Status CloudPrincipalStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// ProviderConfigList contains a list of ProviderConfig.
type CloudPrincipalList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []CloudPrincipal `json:"items"`
}
