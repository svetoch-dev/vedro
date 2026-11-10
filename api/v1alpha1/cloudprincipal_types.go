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
	// PrincipalUnsupportedManagedSA indicates that the selected provider does not support
	// management of service accounts.
	PrincipalUnsupportedManagedSA UnsupportedFeatureReason = "PrincipalUnsupportedManagedSA"
	// PrincipalUnsupportedManagedUser indicates that the selected provider does not support
	// management of users.
	PrincipalUnsupportedManagedUser UnsupportedFeatureReason = "PrincipalUnsupportedManagedUser"
	// PrincipalUnsupportedManagedRole indicates that the selected provider does not support
	// management of roles.
	PrincipalUnsupportedManagedRole UnsupportedFeatureReason = "PrincipalUnsupportedManagedRole"
	// PrincipalUnsupportedManagedAllUsers indicates that the selected provider does not
	// support management of the AllUsers identity.
	PrincipalUnsupportedManagedAllUsers UnsupportedFeatureReason = "PrincipalUnsupportedManagedAllUsers"
	// PrincipalUnsupportedManagedGroup indicates that the selected provider does not support
	// management of groups.
	PrincipalUnsupportedManagedGroup UnsupportedFeatureReason = "PrincipalUnsupportedManagedGroup"
	// PrincipalUnsupportedReferencedSA indicates that the selected provider does not support
	// references to service accounts.
	PrincipalUnsupportedReferencedSA UnsupportedFeatureReason = "PrincipalUnsupportedReferencedSA"
	// PrincipalUnsupportedReferencedUser indicates that the selected provider does not support
	// references to users.
	PrincipalUnsupportedReferencedUser UnsupportedFeatureReason = "PrincipalUnsupportedReferencedUser"
	// PrincipalUnsupportedReferencedRole indicates that the selected provider does not support
	// references to roles.
	PrincipalUnsupportedReferencedRole UnsupportedFeatureReason = "PrincipalUnsupportedReferencedRole"
	// PrincipalUnsupportedReferencedGroup indicates that the selected provider does not
	// support references to groups.
	PrincipalUnsupportedReferencedGroup UnsupportedFeatureReason = "PrincipalUnsupportedReferencedGroup"
	// PrincipalUnsupportedReferencedAllUsers indicates that the selected provider does not
	// support references to the AllUsers identity.
	PrincipalUnsupportedReferencedAllUsers UnsupportedFeatureReason = "PrincipalUnsupportedReferencedAllUsers"
)

// PrincipalKind identifies a cloud identity type. Support varies by provider and management
// policy.
type PrincipalKind string

const (
	// PrincipalKindServiceAccount identifies a cloud service account.
	PrincipalKindServiceAccount PrincipalKind = "ServiceAccount"
	// PrincipalKindRole identifies a cloud IAM role; support depends on the provider.
	PrincipalKindRole PrincipalKind = "Role"
	// PrincipalKindUser identifies a cloud user identity.
	PrincipalKindUser PrincipalKind = "User"
	// PrincipalKindGroup identifies a cloud group identity.
	PrincipalKindGroup PrincipalKind = "Group"
	// PrincipalKindAllUsers represents public access for all users, including unauthenticated
	// users. It can only be referenced, not managed.
	PrincipalKindAllUsers PrincipalKind = "AllUsers"
)

func (k PrincipalKind) Valid() bool {
	switch k {
	case PrincipalKindServiceAccount,
		PrincipalKindRole,
		PrincipalKindUser,
		PrincipalKindGroup,
		PrincipalKindAllUsers:
		return true
	default:
		return false
	}
}

// PrincipalManagementPolicy selects whether the controller manages an external principal or
// only resolves a reference to it.
type PrincipalManagementPolicy string

const (
	// PrincipalManagementPolicyManaged creates and reconciles the external principal according
	// to spec.managed.
	PrincipalManagementPolicyManaged PrincipalManagementPolicy = "Managed"
	// PrincipalManagementPolicyReference resolves an existing identity without creating or
	// deleting it.
	PrincipalManagementPolicyReference PrincipalManagementPolicy = "Reference"
)

// ManagedPrincipalSpec defines an external principal whose lifecycle is managed by the
// controller.
type ManagedPrincipalSpec struct {
	// Name is the desired external principal name, subject to provider naming rules. It is
	// required; the API does not default it from metadata.name. For a GCP ServiceAccount, use
	// the short account ID rather than its email address. The resolved name cannot change
	// after provisioning.
	//
	// +kubebuilder:validation:MinLength=1
	Name string `json:"name"`

	// DeletionPolicy controls cleanup of the managed external principal when the
	// CloudPrincipal is deleted. Delete removes the external principal and is the default.
	// Retain leaves it in the cloud provider.
	//
	// +kubebuilder:validation:Enum=Delete;Retain
	// +kubebuilder:default:=Delete
	// +optional
	DeletionPolicy DeletionPolicy `json:"deletionPolicy,omitempty"`
}

// ReferencedPrincipalSpec identifies an existing external principal without managing its
// lifecycle.
type ReferencedPrincipalSpec struct {
	// Name identifies an existing external principal in the format expected by the provider
	// and kind. For a GCP ServiceAccount, use its email address. For a Yandex Cloud
	// ServiceAccount, use <folder-name>:<service-account-name>, such as prod:some-sa.
	// User and GCP Group references use email addresses. References are resolved without
	// creating or deleting the external principal.
	//
	// +kubebuilder:validation:MinLength=1
	Name string `json:"name"`
}

// CloudPrincipalSpec defines a cloud principal to manage or reference.
//
// +kubebuilder:validation:XValidation:rule="self.managementPolicy != 'Managed' || (has(self.managed) && !has(self.reference))",message="managed must be set and reference must not be set when managementPolicy is Managed"
// +kubebuilder:validation:XValidation:rule="self.managementPolicy != 'Reference' || ((self.kind == 'AllUsers' || has(self.reference)) && !has(self.managed))",message="reference must be set unless kind is AllUsers, and managed must not be set when managementPolicy is Reference"
type CloudPrincipalSpec struct {

	// ProviderRef selects the cluster-scoped ProviderConfig used to manage or resolve this
	// principal. The reference is immutable, and the principal must satisfy the provider's
	// usage policy.
	//
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="providerRef is immutable"
	ProviderRef ProviderConfigReference `json:"providerRef"`

	// Kind selects ServiceAccount, User, Group, Role, or AllUsers. Both providers can manage
	// ServiceAccount and reference ServiceAccount, User, and AllUsers. GCP additionally
	// supports Group references. Role is currently unsupported by both providers.
	// AllUsers represents unauthenticated public access and requires Reference management
	// with both managed and reference omitted.
	//
	// +kubebuilder:validation:Enum=ServiceAccount;User;Group;Role;AllUsers
	Kind PrincipalKind `json:"kind"`

	// ManagementPolicy selects Managed to create and reconcile an external principal, or
	// Reference to resolve an existing principal without managing its lifecycle. The policy is
	// immutable. Both providers currently support Managed only for ServiceAccount.
	//
	// +kubebuilder:validation:Enum=Managed;Reference
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="managementPolicy is immutable"
	ManagementPolicy PrincipalManagementPolicy `json:"managementPolicy"`

	// Managed supplies the name and deletion policy for an external principal created or
	// reconciled by the controller. Required when managementPolicy is Managed and forbidden
	// otherwise.
	//
	// +optional
	Managed *ManagedPrincipalSpec `json:"managed,omitempty"`

	// Reference identifies an existing external principal. Required when managementPolicy is
	// Reference, except for AllUsers, where it must be omitted. Forbidden when
	// managementPolicy is Managed.
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

// +genclient
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Namespaced,categories=vedro
// +kubebuilder:printcolumn:name="Provider",type=string,JSONPath=`.spec.providerRef.name`
// +kubebuilder:printcolumn:name="Ready",type=string,JSONPath=`.status.conditions[?(@.type=="Ready")].status`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`

// CloudPrincipal manages or references a cloud identity used for bucket access and
// authentication.
type CloudPrincipal struct {
	// TypeMeta identifies the API version and kind of this resource.
	metav1.TypeMeta `json:",inline"`

	// ObjectMeta contains Kubernetes identity, labels, annotations, and ownership metadata.
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Spec defines the desired state of the CloudPrincipal.
	Spec CloudPrincipalSpec `json:"spec,omitempty"`

	// Status records the observed state of the CloudPrincipal and is maintained by the
	// controller.
	//
	// +optional
	Status CloudPrincipalStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// CloudPrincipalList contains a list of CloudPrincipal resources.
type CloudPrincipalList struct {
	// TypeMeta identifies the API version and kind of this resource.
	metav1.TypeMeta `json:",inline"`

	// ListMeta contains Kubernetes list metadata, including pagination and resource version.
	metav1.ListMeta `json:"metadata,omitempty"`

	// Items contains the CloudPrincipal resources in this list.
	Items []CloudPrincipal `json:"items"`
}
