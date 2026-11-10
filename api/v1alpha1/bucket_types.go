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
	// BucketUnsupportedVersioning indicates that the selected provider does not support object
	// versioning.
	BucketUnsupportedVersioning UnsupportedFeatureReason = "BucketUnsupportedVersioning"
	// BucketUnsupportedLabels indicates that the selected provider does not support bucket
	// labels or tags.
	BucketUnsupportedLabels UnsupportedFeatureReason = "BucketUnsupportedLabels"
	// BucketUnsupportedPublicAccessPrevention indicates that the selected provider does not
	// support public access prevention.
	BucketUnsupportedPublicAccessPrevention UnsupportedFeatureReason = "BucketUnsupportedPublicAccessPrevention"
	// BucketUnsupportedStorageClass indicates that the selected provider does not support the
	// requested storage class.
	BucketUnsupportedStorageClass UnsupportedFeatureReason = "BucketUnsupportedStorageClass"
	// BucketUnsupportedLifecycleExpiration indicates that the selected provider does not
	// support age-based lifecycle expiration.
	BucketUnsupportedLifecycleExpiration UnsupportedFeatureReason = "BucketUnsupportedLifecycleExpiration"
	// BucketUnsupportedLifecycle indicates that the selected provider does not support object
	// lifecycle rules.
	BucketUnsupportedLifecycle UnsupportedFeatureReason = "BucketUnsupportedLifecycle"
	// BucketUnsupportedLifecycleNamed indicates that the selected provider does not support
	// lifecycle rule names.
	BucketUnsupportedLifecycleNamed UnsupportedFeatureReason = "BucketUnsupportedLifecycleNamed"
	// BucketUnsupportedLifecycleEnabled indicates that the selected provider does not support
	// lifecycle rule enablement.
	BucketUnsupportedLifecycleEnabled UnsupportedFeatureReason = "BucketUnsupportedLifecycleEnabled"
)

// BucketStorageClass is a provider-neutral storage tier. The provider maps each supported
// value to its own storage class.
type BucketStorageClass string

const (
	// BucketStorageClassStandard maps to STANDARD on GCS and Yandex Cloud.
	BucketStorageClassStandard BucketStorageClass = "Standard"
	// BucketStorageClassWarm maps to GCS NEARLINE and is unsupported by Yandex Cloud.
	BucketStorageClassWarm BucketStorageClass = "Warm"
	// BucketStorageClassCold maps to GCS COLDLINE or Yandex Cloud COLD.
	BucketStorageClassCold BucketStorageClass = "Cold"
	// BucketStorageClassIce maps to GCS ARCHIVE or Yandex Cloud ICE.
	BucketStorageClassIce BucketStorageClass = "Ice"
)

// BucketLifecycleAction is the operation performed by an object lifecycle rule.
type BucketLifecycleAction string

const (
	// BucketLifecycleActionDelete requests deletion of objects matching the lifecycle rule.
	BucketLifecycleActionDelete BucketLifecycleAction = "Delete"
)

// BucketCloudSpecificConfig contains settings that are specific to the selected cloud
// provider.
type BucketCloudSpecificConfig struct {
	// Gcp configures Google Cloud Storage settings. It may only be set when the ProviderConfig
	// type is gcp.
	//
	// +optional
	Gcp *BucketGcpConfig `json:"gcp,omitempty"`

	// Yc reserves a configuration block for Yandex Cloud extensions. It currently has no
	// settings and may only be set when the ProviderConfig type is yc.
	//
	// +optional
	Yc *BucketYcConfig `json:"yc,omitempty"`
}

// BucketYcConfig contains Yandex Cloud-specific bucket configuration.
// It is currently empty and reserved for future extensions.
type BucketYcConfig struct {
}

// BucketGcpConfig configures Google Cloud Storage features without a provider-neutral
// equivalent.
type BucketGcpConfig struct {
	// SoftDeletePolicy configures retention of soft-deleted objects. When omitted, the
	// controller applies a seven-day retention period. Set retentionDuration to "0s" to
	// disable soft delete.
	//
	// +optional
	SoftDeletePolicy *SoftDeletePolicy `json:"softDeletePolicy,omitempty"`
}

// SoftDeletePolicy controls Google Cloud Storage retention of deleted objects.
type SoftDeletePolicy struct {
	// RetentionDuration is how long GCS retains soft-deleted objects before permanent
	// deletion. Use "0s" to disable soft delete, or a Go duration representing a whole number
	// of days from 7 through 90, such as "168h" for seven days. Defaults to "168h".
	//
	// +kubebuilder:default:="168h"
	// +kubebuilder:validation:XValidation:rule="duration(self) == duration('0s') || (duration(self) >= duration('168h') && duration(self) <= duration('2160h') && duration(self).getSeconds() % 86400 == 0)",message="retentionDuration must be 0 or a whole number of days between 7 and 90"
	RetentionDuration metav1.Duration `json:"retentionDuration"`
}

// BucketVersioning configures preservation of object versions.
type BucketVersioning struct {
	// Enabled enables object versioning when true and disables or suspends it when false,
	// according to provider semantics. Disabling versioning does not remove existing object
	// versions.
	Enabled bool `json:"enabled"`
}

// BucketLifecycle defines the desired object lifecycle rules for a bucket.
type BucketLifecycle struct {
	// Rules is the desired set of object lifecycle rules. An empty list configures no
	// lifecycle rules. Rule names and other capabilities depend on the selected provider.
	//
	// +optional
	Rules []BucketLifecycleRule `json:"rules,omitempty"`
}

// BucketProperties records bucket settings returned by a provider during reconciliation.
type BucketProperties struct {
	// PublicAccessPrevention records the provider-reported setting for preventing public
	// access.
	PublicAccessPrevention *bool `json:"publicAccessPrevention,omitempty"`

	// Versioning records the object versioning configuration reported or applied by the
	// provider.
	Versioning *BucketVersioning `json:"versioning,omitempty"`

	// Lifecycle records the lifecycle rules reported or applied by the provider.
	Lifecycle *BucketLifecycle `json:"lifecycle,omitempty"`

	// StorageClass records the bucket storage class using Vedro storage class names.
	StorageClass BucketStorageClass `json:"storageClass,omitempty"`

	// Labels records the cloud bucket labels or tags reported or applied by the provider.
	Labels map[string]string `json:"labels,omitempty"`

	// CloudSpecificConfig records provider-specific bucket settings reported or applied by the
	// provider, including normalized defaults when available.
	CloudSpecificConfig *BucketCloudSpecificConfig `json:"cloudSpecificConfig,omitempty"`
}

// BucketLifecycleRule selects objects by age and configures a lifecycle action.
type BucketLifecycleRule struct {
	// Name is an optional stable identifier for this lifecycle rule. Yandex Cloud supports
	// rule names; GCS does not.
	//
	// +kubebuilder:validation:MinLength=1
	// +optional
	Name *string `json:"name,omitempty"`

	// Enabled controls whether this rule is active. Defaults to true. Disabled rules are
	// excluded from the lifecycle configuration sent to GCS.
	//
	// +kubebuilder:default:=true
	Enabled bool `json:"enabled"`

	// AgeDays selects objects by age in days for the lifecycle action. When set, it must be at
	// least 1. The provider determines how object age is evaluated.
	//
	// +kubebuilder:validation:Minimum=1
	// +optional
	AgeDays *int64 `json:"ageDays"`

	// Action is the operation to perform on matching objects. The only supported value is
	// Delete; execution follows the cloud provider's lifecycle scheduling.
	//
	// +kubebuilder:validation:Enum=Delete
	Action BucketLifecycleAction `json:"action"`
}

// BucketSpec defines the desired cloud bucket and its lifecycle settings.
type BucketSpec struct {
	// ProviderRef selects the cluster-scoped ProviderConfig used to manage this bucket. The
	// reference is immutable, and the bucket must satisfy the provider's usage policy.
	//
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="providerRef is immutable"
	ProviderRef ProviderConfigReference `json:"providerRef"`

	// Name is the external cloud bucket name. When omitted, metadata.name is used. The
	// resolved name must satisfy provider naming rules and cannot change after the external
	// name has been recorded in status.
	//
	// +kubebuilder:validation:MinLength=1
	// +optional
	Name string `json:"name,omitempty"`

	// Location is the cloud bucket location, such as europe-west1 for GCS or ru-central1 for
	// Yandex Cloud. GCS also accepts multi-regions such as EU and predefined dual-regions such
	// as EUR4. An existing bucket must match the requested location; the controller does not
	// relocate buckets.
	//
	// +kubebuilder:validation:MinLength=1
	Location string `json:"location"`

	// StorageClass sets the bucket's default storage class. Defaults to Standard. GCS maps
	// Standard, Warm, Cold, and Ice to STANDARD, NEARLINE, COLDLINE, and ARCHIVE. Yandex Cloud
	// supports Standard, Cold, and Ice, mapped to STANDARD, COLD, and ICE; Warm is
	// unsupported.
	//
	// +kubebuilder:validation:Enum=Standard;Warm;Cold;Ice
	// +kubebuilder:default:=Standard
	// +optional
	StorageClass BucketStorageClass `json:"storageClass,omitempty"`

	// DeletionPolicy controls cleanup when this Kubernetes resource is deleted. Retain leaves
	// the external bucket and its objects intact and is the default. Delete removes object
	// versions before deleting the bucket; cloud retention restrictions may prevent cleanup.
	//
	// +kubebuilder:validation:Enum=Delete;Retain
	// +kubebuilder:default:=Retain
	// +optional
	DeletionPolicy DeletionPolicy `json:"deletionPolicy,omitempty"`

	// PublicAccessPrevention enforces prevention of public access when true. GCS treats false
	// or omission as inherited public access prevention, which may still be enforced by a
	// parent policy. Yandex Cloud does not support this setting.
	//
	// +optional
	PublicAccessPrevention *bool `json:"publicAccessPrevention,omitempty"`

	// Versioning configures object versioning. Omission requests disabled versioning; existing
	// object versions are not removed by disabling versioning.
	//
	// +optional
	Versioning *BucketVersioning `json:"versioning,omitempty"`

	// Lifecycle configures object expiration rules. Omission or an empty rules list requests
	// no lifecycle rules. Support for individual rule fields depends on the provider.
	//
	// +optional
	Lifecycle *BucketLifecycle `json:"lifecycle,omitempty"`

	// Labels is the desired map of cloud bucket labels or tags, separate from Kubernetes
	// metadata.labels. The controller reconciles the entire map; labels absent from it are
	// removed.
	//
	// +optional
	Labels map[string]string `json:"labels,omitempty"`

	// CloudSpecificConfig configures settings without a cloud-agnostic equivalent. Set only
	// the block matching the selected ProviderConfig type; a block for another provider is
	// rejected during reconciliation. GCS applies a seven-day soft-delete retention period
	// when its soft-delete settings are omitted.
	CloudSpecificConfig *BucketCloudSpecificConfig `json:"cloudSpecificConfig,omitempty"`
}

// BucketStatus records the external bucket identity, applied settings, and reconciliation
// state.
type BucketStatus struct {
	// ExternalName is the external cloud bucket name recorded after provisioning. It is also
	// used to identify the bucket during deletion.
	//
	// +optional
	ExternalName string `json:"externalName,omitempty"`

	// ExternalId is the provider-side bucket identifier. It may be the same as ExternalName
	// and is used when configuring bucket access.
	//
	// +optional
	ExternalId string `json:"externalId,omitempty"`

	// Location is the provider-reported bucket location, including a region, multi-region,
	// or dual-region where supported.
	//
	// +optional
	Location string `json:"location,omitempty"`

	// ObservedProvider is the name of the ProviderConfig recorded during successful bucket
	// reconciliation and used for cleanup.
	//
	// +optional
	ObservedProvider string `json:"observedProvider,omitempty"`

	// ObservedGeneration is the latest metadata.generation observed by the controller.
	//
	// +optional
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`

	// Applied records the bucket properties returned by the last successful provider
	// reconciliation. Provider defaults and normalization may make these differ from the
	// requested spec.
	//
	// +optional
	Applied *BucketProperties `json:"applied,omitempty"`

	// UnsupportedFeatures lists requested settings that the selected provider cannot apply.
	// Unsupported settings prevent normal reconciliation until the request is corrected.
	//
	// +optional
	UnsupportedFeatures []UnsupportedFeature `json:"unsupported,omitempty"`

	// Conditions represent the latest available observations of the bucket state.
	//
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// +genclient
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Namespaced,categories=vedro
// +kubebuilder:printcolumn:name="Provider",type=string,JSONPath=`.spec.providerRef.name`
// +kubebuilder:printcolumn:name="Bucket",type=string,JSONPath=`.status.externalName`
// +kubebuilder:printcolumn:name="Location",type=string,JSONPath=`.spec.location`
// +kubebuilder:printcolumn:name="Ready",type=string,JSONPath=`.status.conditions[?(@.type=="Ready")].status`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`

// Bucket manages a cloud object storage bucket through a ProviderConfig.
type Bucket struct {
	// TypeMeta identifies the API version and kind of this resource.
	metav1.TypeMeta `json:",inline"`

	// ObjectMeta contains Kubernetes identity, labels, annotations, and ownership metadata.
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Spec defines the desired state of the Bucket.
	Spec BucketSpec `json:"spec,omitempty"`

	// Status records the observed state of the Bucket and is maintained by the controller.
	//
	// +optional
	Status BucketStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// BucketList contains a list of Bucket.
type BucketList struct {
	// TypeMeta identifies the API version and kind of this resource.
	metav1.TypeMeta `json:",inline"`

	// ListMeta contains Kubernetes list metadata, including pagination and resource version.
	metav1.ListMeta `json:"metadata,omitempty"`

	// Items contains the Bucket resources in this list.
	Items []Bucket `json:"items"`
}
