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

const (
	PrincipalUnsupportedManagedSA       UnsupportedFeatureReason = "PrincipalUnsupportedManagedSA"
	PrincipalUnsupportedManagedUser     UnsupportedFeatureReason = "PrincipalUnsupportedManagedUser"
	PrincipalUnsupportedManagedRole     UnsupportedFeatureReason = "PrincipalUnsupportedManagedRole"
	PrincipalUnsupportedManagedGroup    UnsupportedFeatureReason = "PrincipalUnsupportedManagedGroup"
	PrincipalUnsupportedReferencedSA    UnsupportedFeatureReason = "PrincipalUnsupportedReferencedSA"
	PrincipalUnsupportedReferencedUser  UnsupportedFeatureReason = "PrincipalUnsupportedReferencedUser"
	PrincipalUnsupportedReferencedRole  UnsupportedFeatureReason = "PrincipalUnsupportedReferencedRole"
	PrincipalUnsupportedReferencedGroup UnsupportedFeatureReason = "PrincipalUnsupportedReferencedGroup"
)

type PrincipalKind string

const (
	PrincipalKindServiceAccount PrincipalKind = "ServiceAccount"
	PrincipalKindRole           PrincipalKind = "Role"
	PrincipalKindUser           PrincipalKind = "User"
	PrincipalKindGroup          PrincipalKind = "Group"
)

type PrincipalManagementPolicy string

const (
	PrincipalManagementPolicyManaged   PrincipalManagementPolicy = "Managed"
	PrincipalManagementPolicyReference PrincipalManagementPolicy = "Reference"
)

type ManagedPrincipalSpec struct {
	// Name is the cloud provider principal name.
	// Name format depends on what Kind of principal it is
	//
	// +kubebuilder:validation:MinLength=1
	Name string `json:"name"`

	// DeletionPolicy controls what happens to the external CloudPrincipal
	// when this Kubernetes object is deleted.
	//
	// +kubebuilder:validation:Enum=Delete;Retain
	// +kubebuilder:default:=Delete
	// +optional
	DeletionPolicy DeletionPolicy `json:"deletionPolicy,omitempty"`
}

type ReferencedPrincipalSpec struct {
	// Name is the cloud provider principal name.
	// Name format depends on what Kind of principal it is
	//
	// +kubebuilder:validation:MinLength=1
	Name string `json:"name"`
}

// +kubebuilder:validation:XValidation:rule="self.managementPolicy != 'Managed' || (has(self.managed) && !has(self.reference))",message="managed must be set and reference must not be set when managementPolicy is Managed"
// +kubebuilder:validation:XValidation:rule="self.managementPolicy != 'Reference' || (has(self.reference) && !has(self.managed))",message="reference must be set and managed must not be set when managementPolicy is Reference"
type CloudPrincipalSpec struct {

	// ProviderRef references the ProviderConfig used to manage this principal.
	//
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="providerRef is immutable"
	ProviderRef ProviderConfigReference `json:"providerRef"`

	// Kind identifies the type of cloud principal.
	//
	// +kubebuilder:validation:Enum=ServiceAccount;User;Group;Role
	Kind PrincipalKind `json:"kind"`

	// ManagementPolicy controls whether the external principal is managed by this
	// resource or references an existing principal.
	//
	// +kubebuilder:validation:Enum=Managed;Reference
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="managementPolicy is immutable"
	ManagementPolicy PrincipalManagementPolicy `json:"managementPolicy"`

	// Managed configures a principal managed by this resource.
	// It must be set only when ManagementPolicy is Managed.
	//
	// +optional
	Managed *ManagedPrincipalSpec `json:"managed,omitempty"`

	// Reference identifies an existing principal that is not managed by this resource.
	// It must be set only when ManagementPolicy is Reference.
	//
	// +optional
	Reference *ReferencedPrincipalSpec `json:"reference,omitempty"`
}

// CloudPrincipalStatus defines the observed state of a CloudPrincipal.
type CloudPrincipalStatus struct {
	// ExternalName is the provider-side principal name.
	//
	// +optional
	ExternalName string `json:"externalName,omitempty"`

	// ObservedProvider is the ProviderConfig used for the last successful reconciliation.
	//
	// +optional
	ObservedProvider string `json:"observedProvider,omitempty"`

	// ObservedGeneration is the latest metadata.generation observed by the controller.
	//
	// +optional
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`

	// UnsupportedFeatures lists requested features that the selected provider does not support.
	//
	// +optional
	UnsupportedFeatures []UnsupportedFeature `json:"unsupported,omitempty"`

	// ExternalId is the provider-side immutable principal identifier.
	//
	// +optional
	ExternalId string `json:"externalId,omitempty"`

	// Conditions represent the latest available observations of the CloudPrincipal state.
	//
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`

	// Kind is the kind reported for the external principal.
	//
	// +optional
	Kind PrincipalKind `json:"kind,omitempty"`

	// ManagementPolicy is the policy used to reconcile the external principal.
	//
	// +optional
	ManagementPolicy PrincipalManagementPolicy `json:"managementPolicy,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Namespaced,categories=vedro
// +kubebuilder:printcolumn:name="Provider",type=string,JSONPath=`.spec.providerRef.name`
// +kubebuilder:printcolumn:name="Ready",type=string,JSONPath=`.status.conditions[?(@.type=="Ready")].status`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`

// CloudPrincipal is the Schema for the cloudprincipals API.
type CloudPrincipal struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Spec defines the desired state of the CloudPrincipal.
	Spec CloudPrincipalSpec `json:"spec,omitempty"`

	// Status defines the observed state of the CloudPrincipal.
	//
	// +optional
	Status CloudPrincipalStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// CloudPrincipalList contains a list of CloudPrincipal resources.
type CloudPrincipalList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	// Items contains the CloudPrincipal resources in this list.
	Items []CloudPrincipal `json:"items"`
}
