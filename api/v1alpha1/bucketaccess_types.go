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

// BucketAccessLevel selects a provider-neutral bucket permission level.
type BucketAccessLevel string

const (
	// ObjectReader maps to GCS roles/storage.objectViewer or Yandex Cloud storage.viewer on
	// the bucket.
	ObjectReader BucketAccessLevel = "ObjectReader"
	// ObjectWriter maps to GCS roles/storage.objectCreator or Yandex Cloud storage.uploader on
	// the bucket. It is not a combined reader and writer role.
	ObjectWriter BucketAccessLevel = "ObjectWriter"
	// ObjectAdmin maps to GCS roles/storage.objectAdmin or Yandex Cloud storage.editor on the
	// bucket.
	ObjectAdmin BucketAccessLevel = "ObjectAdmin"
	// BucketAdmin maps to GCS roles/storage.admin or Yandex Cloud storage.admin on the bucket.
	BucketAdmin BucketAccessLevel = "BucketAdmin"
)

const (
	// BucketAccessUnsupportedObjectReader indicates that the selected provider does not
	// support ObjectReader grants.
	BucketAccessUnsupportedObjectReader UnsupportedFeatureReason = "BucketAccessUnsupportedObjectReader"
	// BucketAccessUnsupportedObjectWriter indicates that the selected provider does not
	// support ObjectWriter grants.
	BucketAccessUnsupportedObjectWriter UnsupportedFeatureReason = "BucketAccessUnsupportedObjectWriter"
	// BucketAccessUnsupportedObjectAdmin indicates that the selected provider does not support
	// ObjectAdmin grants.
	BucketAccessUnsupportedObjectAdmin UnsupportedFeatureReason = "BucketAccessUnsupportedObjectAdmin"
	// BucketAccessUnsupportedBucketAdmin indicates that the selected provider does not support
	// BucketAdmin grants.
	BucketAccessUnsupportedBucketAdmin UnsupportedFeatureReason = "BucketAccessUnsupportedBucketAdmin"
)

// Access configures the permission level granted to a cloud principal on a bucket.
type Access struct {
	// Level selects ObjectReader, ObjectWriter, ObjectAdmin, or BucketAdmin. The selected
	// level is mapped to a provider-specific role on the bucket.
	//
	// +kubebuilder:validation:Enum=ObjectReader;ObjectWriter;ObjectAdmin;BucketAdmin
	Level BucketAccessLevel `json:"level"`
}

// BucketAccessProperties records the cloud identities and access level of an applied grant.
type BucketAccessProperties struct {
	// BucketName is the external name of the bucket on which access was granted.
	BucketName string `json:"bucketName"`

	// BucketId is the provider-side bucket identifier used for access management. It may be
	// the same as BucketName.
	BucketId string `json:"bucketId"`

	// PrincipalId is the provider-side identity to which access was granted.
	PrincipalId string `json:"principalId"`

	// GrantedAccess is the Vedro access level last applied to the bucket, rather than the
	// provider-specific role name.
	GrantedAccess BucketAccessLevel `json:"grantedAccess"`
}

// BucketAccessSpec defines a permission grant between a Bucket and a CloudPrincipal.
type BucketAccessSpec struct {
	// BucketRef identifies the Bucket on which access is granted. The reference, including its
	// explicit namespace, is immutable. The bucket and principal must reference the same
	// ProviderConfig.
	//
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="bucketRef is immutable"
	BucketRef BucketReference `json:"bucketRef"`

	// PrincipalRef identifies the CloudPrincipal receiving access, including its explicit
	// namespace. Changing the principal causes the previously recorded grant to be revoked
	// before the new grant is applied.
	PrincipalRef PrincipalReference `json:"principalRef"`

	// DeletionPolicy controls the grant when this Kubernetes resource is deleted. Delete
	// revokes the recorded permission and is the default. Retain leaves the permission in
	// place. Neither policy deletes the bucket or principal.
	//
	// +kubebuilder:validation:Enum=Delete;Retain
	// +kubebuilder:default:=Delete
	// +optional
	DeletionPolicy DeletionPolicy `json:"deletionPolicy,omitempty"`

	// Access specifies the desired permission level on the referenced bucket.
	Access Access `json:"access"`
}

// BucketAccessStatus records the applied grant and reconciliation state.
type BucketAccessStatus struct {
	// Applied records the external bucket, principal, and permission level from the last
	// successful reconciliation. These values identify the grant to revoke during updates or
	// deletion.
	//
	// +optional
	Applied *BucketAccessProperties `json:"applied,omitempty"`

	// ObservedGeneration is the latest metadata.generation observed by the controller.
	//
	// +optional
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`

	// UnsupportedFeatures lists requested access levels that the selected provider does not
	// support.
	//
	// +optional
	UnsupportedFeatures []UnsupportedFeature `json:"unsupported,omitempty"`

	// ObservedProvider is the name of the ProviderConfig recorded during successful access
	// reconciliation and used to revoke the grant during deletion.
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

// BucketAccess grants a CloudPrincipal access to a Bucket through their shared
// ProviderConfig.
type BucketAccess struct {
	// TypeMeta identifies the API version and kind of this resource.
	metav1.TypeMeta `json:",inline"`

	// ObjectMeta contains Kubernetes identity, labels, annotations, and ownership metadata.
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Spec defines the desired state of the BucketAccess.
	Spec BucketAccessSpec `json:"spec,omitempty"`

	// Status records the observed state of the BucketAccess and is maintained by the
	// controller.
	//
	// +optional
	Status BucketAccessStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// BucketAccessList contains a list of BucketAccess resources.
type BucketAccessList struct {
	// TypeMeta identifies the API version and kind of this resource.
	metav1.TypeMeta `json:",inline"`

	// ListMeta contains Kubernetes list metadata, including pagination and resource version.
	metav1.ListMeta `json:"metadata,omitempty"`

	// Items contains the BucketAccess resources in this list.
	Items []BucketAccess `json:"items"`
}
