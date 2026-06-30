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
	BucketUnsupportedVersioning             UnsupportedFeatureReason = "BucketUnsupportedVersioning"
	BucketUnsupportedLabels                 UnsupportedFeatureReason = "BucketUnsupportedLabels"
	BucketUnsupportedPublicAccessPrevention UnsupportedFeatureReason = "BucketUnsupportedPublicAccessPrevention"
	BucketUnsupportedStorageClass           UnsupportedFeatureReason = "BucketUnsupportedStorageClass"
	BucketUnsupportedLifecycleExpiration    UnsupportedFeatureReason = "BucketUnsupportedLifecycleExpiration"
	BucketUnsupportedLifecycle              UnsupportedFeatureReason = "BucketUnsupportedLifecycle"
	BucketUnsupportedLifecycleNamed         UnsupportedFeatureReason = "BucketUnsupportedLifecycleNamed"
	BucketUnsupportedLifecycleEnabled       UnsupportedFeatureReason = "BucketUnsupportedLifecycleEnabled"
)

type BucketStorageClass string

const (
	BucketStorageClassStandard BucketStorageClass = "Standard"
	BucketStorageClassWarm     BucketStorageClass = "Warm"
	BucketStorageClassCold     BucketStorageClass = "Cold"
	BucketStorageClassIce      BucketStorageClass = "Ice"
)

type BucketLifecycleAction string

const (
	BucketLifecycleActionDelete BucketLifecycleAction = "Delete"
)

type BucketCloudSpecificConfig struct {
	// GCP-specific bucket configuration.
	// +optional
	Gcp *BucketGcpConfig `json:"gcp,omitempty"`

	// Yandex cloud-specific bucket configuration.
	// Reserved for future Yandex Cloud extensions.
	// +optional
	Yc *BucketYcConfig `json:"yc,omitempty"`
}

// BucketYcConfig contains Yandex Cloud-specific bucket configuration.
// It is currently empty and reserved for future extensions.
type BucketYcConfig struct {
}

type BucketGcpConfig struct {
	// +optional
	SoftDeletePolicy *SoftDeletePolicy `json:"softDeletePolicy,omitempty"`
}

type SoftDeletePolicy struct {
	// RetentionDuration is how long deleted objects are retained before permanent deletion.
	// Must be 0 or a whole number of days between 7 and 90 for GCS.
	// Example: "168h" for 7 days.
	// +kubebuilder:default:="168h"
	// +kubebuilder:validation:XValidation:rule="duration(self) == duration('0s') || (duration(self) >= duration('168h') && duration(self) <= duration('2160h') && duration(self).getSeconds() % 86400 == 0)",message="retentionDuration must be 0 or a whole number of days between 7 and 90"
	RetentionDuration metav1.Duration `json:"retentionDuration"`
}

type BucketVersioning struct {
	// Enabled controls bucket object versioning.
	Enabled bool `json:"enabled"`
}

type BucketLifecycle struct {
	// +optional
	Rules []BucketLifecycleRule `json:"rules,omitempty"`
}

type BucketProperties struct {
	PublicAccessPrevention *bool                      `json:"publicAccessPrevention,omitempty"`
	Versioning             *BucketVersioning          `json:"versioning,omitempty"`
	Lifecycle              *BucketLifecycle           `json:"lifecycle,omitempty"`
	StorageClass           BucketStorageClass         `json:"storageClass,omitempty"`
	Labels                 map[string]string          `json:"labels,omitempty"`
	CloudSpecificConfig    *BucketCloudSpecificConfig `json:"cloudSpecificConfig,omitempty"`
}

type BucketLifecycleRule struct {
	// Name is a stable identifier for this lifecycle rule.
	//
	// +kubebuilder:validation:MinLength=1
	// +optional
	Name *string `json:"name,omitempty"`

	// Enabled controls whether this lifecycle rule should be active.
	// +kubebuilder:default:=true
	Enabled bool `json:"enabled"`

	// AgeDays matches objects older than this many days.
	//
	// +kubebuilder:validation:Minimum=1
	// +optional
	AgeDays *int64 `json:"ageDays"`

	// Action describes what happens to matching objects.
	//
	// +kubebuilder:validation:Enum=Delete
	Action BucketLifecycleAction `json:"action"`
}

type BucketSpec struct {
	// ProviderRef references the ProviderConfig used to manage this bucket.
	ProviderRef ProviderConfigReference `json:"providerRef"`

	// Name is the real cloud provider bucket name.
	//
	// +kubebuilder:validation:MinLength=1
	// +optional
	Name string `json:"name,omitempty"`

	// Location is the cloud provider location/region.
	//
	// Examples:
	// - GCP: europe-west1
	// - Yandex: ru-central1
	//
	// +kubebuilder:validation:MinLength=1
	Location string `json:"location"`

	// bucket storage class
	//
	// +kubebuilder:validation:Enum=Standard;Warm;Cold;Ice
	// +kubebuilder:default:=Standard
	// +optional
	StorageClass BucketStorageClass `json:"storageClass,omitempty"`

	// DeletionPolicy controls what happens to the external bucket
	// when this Kubernetes object is deleted.
	//
	// +kubebuilder:validation:Enum=Delete;Retain
	// +kubebuilder:default:=Retain
	// +optional
	DeletionPolicy DeletionPolicy `json:"deletionPolicy,omitempty"`

	// PublicAccessPrevention controls whether the bucket may be publicly accessible.
	//
	// +optional
	PublicAccessPrevention *bool `json:"publicAccessPrevention,omitempty"`

	// Versioning configures object versioning.
	//
	// +optional
	Versioning *BucketVersioning `json:"versioning,omitempty"`

	// Lifecycle configures object lifecycle rules.
	//
	// +optional
	Lifecycle *BucketLifecycle `json:"lifecycle,omitempty"`

	// Labels are cloud provider labels/tags applied to the bucket.
	//
	// +optional
	Labels map[string]string `json:"labels,omitempty"`

	// UnsupportedFeaturePolicy controls what the controller does when
	// the selected provider does not support a requested feature.
	//
	// +kubebuilder:validation:Enum=Fail;Warn
	// +kubebuilder:default:=Fail
	// +optional
	UnsupportedFeaturePolicy UnsupportedFeaturePolicy `json:"unsupportedFeaturePolicy,omitempty"`

	// CloudSpecificConfig contains provider-specific bucket settings that do not
	// have a cloud-agnostic equivalent in BucketSpec.
	//
	// These settings are applied only when the selected provider matches the
	// corresponding provider config field. For example, Gcp is used only for GCS
	// buckets.
	CloudSpecificConfig *BucketCloudSpecificConfig `json:"cloudSpecificConfig,omitempty"`
}

type BucketStatus struct {
	// ExternalName is the provider-side bucket name.
	//
	// +optional
	ExternalName string `json:"externalName,omitempty"`
	// Location is the provider-side bucket region.
	//
	// +optional
	Location string `json:"location,omitempty"`

	// Provider used for this bucket
	//
	// +optional
	ObservedProvider string `json:"observedProvider,omitempty"`

	// ObservedGeneration is the latest metadata.generation observed by the controller.
	//
	// +optional
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`

	// Applied is the applied bucket attributes
	//
	// +optional
	Applied *BucketProperties `json:"applied,omitempty"`

	// List of unsupported features set on Bucket resource
	//
	// +optional
	UnsupportedFeatures []UnsupportedFeature `json:"unsupported,omitempty"`

	// Conditions represent the latest available observations of the bucket state.
	//
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Namespaced,categories=vedro
// +kubebuilder:printcolumn:name="Provider",type=string,JSONPath=`.spec.providerRef.name`
// +kubebuilder:printcolumn:name="Bucket",type=string,JSONPath=`.status.externalName`
// +kubebuilder:printcolumn:name="Location",type=string,JSONPath=`.spec.location`
// +kubebuilder:printcolumn:name="Ready",type=string,JSONPath=`.status.conditions[?(@.type=="Ready")].status`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`

// Bucket is the Schema for the buckets API.
type Bucket struct {
	metav1.TypeMeta `json:",inline"`

	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec BucketSpec `json:"spec,omitempty"`

	// +optional
	Status BucketStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// BucketList contains a list of Bucket.
type BucketList struct {
	metav1.TypeMeta `json:",inline"`

	metav1.ListMeta `json:"metadata,omitempty"`

	Items []Bucket `json:"items"`
}
