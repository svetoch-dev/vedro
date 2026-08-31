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

type BucketAccessLevel string

const (
	ObjectReader BucketAccessLevel = "ObjectReader"
	ObjectWriter BucketAccessLevel = "ObjectWriter"
	ObjectAdmin  BucketAccessLevel = "ObjectAdmin"
	BucketAdmin  BucketAccessLevel = "BucketAdmin"
)

const (
	BucketAccessUnsupportedObjectReader UnsupportedFeatureReason = "BucketAccessUnsupportedObjectReader"
	BucketAccessUnsupportedObjectWriter UnsupportedFeatureReason = "BucketAccessUnsupportedObjectWriter"
	BucketAccessUnsupportedObjectAdmin  UnsupportedFeatureReason = "BucketAccessUnsupportedObjectAdmin"
	BucketAccessUnsupportedBucketAdmin  UnsupportedFeatureReason = "BucketAccessUnsupportedBucketAdmin"
)

type Access struct {
	// +kubebuilder:validation:Enum=ObjectReader;ObjectWriter;ObjectAdmin;BucketAdmin
	Level BucketAccessLevel `json:"level"`
}

type BucketAccessProperties struct {
	// BucketName name of the bucket to which access is granted
	//
	BucketName string `json:"bucketName"`
	// BucketId id of the bucket to which access is granted. Can
	// be the equal to BucketName
	//
	BucketId string `json:"bucketId"`
	// PrincipalId id of the cloud principal that access is granted too
	//
	PrincipalId string `json:"principalId"`
	// GrantedAccess cloud specific access name that is granted
	//
	GrantedAccess BucketAccessLevel `json:"grantedAccess"`
}

type BucketAccessSpec struct {
	// BucketRef references the Bucket to which access is granted.
	//
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="bucketRef is immutable"
	BucketRef BucketReference `json:"bucketRef"`
	// PrincipalRef references the CloudPrincipal to which access is granted.
	//
	PrincipalRef PrincipalReference `json:"principalRef"`

	// DeletionPolicy controls what happens to the bucketaccess
	// when this Kubernetes object is deleted. If Retain is used
	// when BucketAccess object is deleted permissions are not revoked
	//
	// +kubebuilder:validation:Enum=Delete;Retain
	// +kubebuilder:default:=Delete
	// +optional
	DeletionPolicy DeletionPolicy `json:"deletionPolicy,omitempty"`

	Access Access `json:"access"`
}

// ProviderConfigStatus defines the observed provider configuration state.
type BucketAccessStatus struct {
	// Applied - what has been already applied by this controller
	//
	// +optional
	Applied *BucketAccessProperties `json:"applied,omitempty"`
	// ObservedGeneration is the latest metadata.generation observed by the controller.
	//
	// +optional
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`
	// List of unsupported features set on BucketAccess resource
	//
	// +optional
	UnsupportedFeatures []UnsupportedFeature `json:"unsupported,omitempty"`
	// Provider used for this BucketAccess
	//
	// +optional
	ObservedProvider string `json:"observedProvider,omitempty"`
	// Conditions represent the latest available observations of the BucketAccess state.
	//
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Namespaced,categories=vedro
// +kubebuilder:printcolumn:name="Bucket",type=string,JSONPath=`.spec.bucketRef.name`
// +kubebuilder:printcolumn:name="Principal",type=string,JSONPath=`.spec.principalRef.name`
// +kubebuilder:printcolumn:name="Ready",type=string,JSONPath=`.status.conditions[?(@.type=="Ready")].status`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`

// ProviderConfig is the Schema for the providerconfigs API.
type BucketAccess struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec BucketAccessSpec `json:"spec,omitempty"`

	// +optional
	Status BucketAccessStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// ProviderConfigList contains a list of ProviderConfig.
type BucketAccessList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []BucketAccess `json:"items"`
}
