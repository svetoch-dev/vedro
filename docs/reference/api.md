# API Reference

> Generated with `crd-ref-docs`. Run `make generate` to update this reference.

## Packages

- [vedro.svetoch.dev/v1alpha1](#vedrosvetochdevv1alpha1)

## vedro.svetoch.dev/v1alpha1

Package v1alpha1 contains API Schema definitions for the vedro v1alpha1 API group.

### Resource Types

- [Bucket](#bucket)
- [BucketAccess](#bucketaccess)
- [CloudPrincipal](#cloudprincipal)
- [CloudPrincipalAuth](#cloudprincipalauth)
- [ProviderConfig](#providerconfig)

#### Access

Access configures the permission level granted to a cloud principal on a bucket.

_Appears in:_

- [BucketAccessSpec](#bucketaccessspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `level` _[BucketAccessLevel](#bucketaccesslevel)_ | Level selects ObjectReader, ObjectWriter, ObjectAdmin, or BucketAdmin. The selected level is mapped to a provider-specific role on the bucket. |  | Enum: [ObjectReader ObjectWriter ObjectAdmin BucketAdmin] |

#### AllowedNamePatterns

_Underlying type:_ _string array_

AllowedNamePatterns is a list of Go regular expressions. A name is allowed
when at least one expression matches. Expressions are not implicitly anchored;
use ^ and $ to match the entire name. An empty list allows no names.

_Appears in:_

- [BucketPolicySpec](#bucketpolicyspec)
- [PrincipalPolicySpec](#principalpolicyspec)

#### AllowedNamespacesSpec

AllowedNamespacesSpec selects Kubernetes namespaces that may use a ProviderConfig.

_Appears in:_

- [UsagePolicySpec](#usagepolicyspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `names` _string array_ | Names lists exact Kubernetes namespace names allowed to use this provider. Must be empty when all is true. If all is false and names is empty, no namespaces are allowed. |  | Optional: \{\} |
| `all` _boolean_ | All allows every Kubernetes namespace to use this provider. Defaults to false and cannot be combined with a non-empty names list. |  | Optional: \{\} |

#### AuthMethod

_Underlying type:_ _string_

AuthMethod selects static credentials or workload identity authentication. ProviderConfig
uses it for operator credentials; CloudPrincipalAuth uses it for application credentials.

_Appears in:_

- [CloudPrincipalAuthProperties](#cloudprincipalauthproperties)
- [CloudPrincipalAuthSpec](#cloudprincipalauthspec)
- [ProviderConfigSpec](#providerconfigspec)

| Value | Description |
| --- | --- |
| `StaticCredentials` | AuthMethodStaticCredentials uses a provider key or issues static application credentials, depending on the resource. |
| `WorkloadIdentity` | AuthMethodWorkloadIdentity uses ambient operator credentials or configures a Kubernetes workload identity, depending on the resource. |

#### AuthObjectReference

AuthObjectReference identifies a Kubernetes object by name within the CloudPrincipalAuth namespace.

_Appears in:_

- [StaticCredentialsSpec](#staticcredentialsspec)
- [WorkloadIdentitySpec](#workloadidentityspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `name` _string_ | Name is the referenced Kubernetes object name in the CloudPrincipalAuth namespace. The name is immutable. |  | MinLength: 1 |

#### Bucket

Bucket manages a cloud object storage bucket through a ProviderConfig.

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `apiVersion` _string_ | `vedro.svetoch.dev/v1alpha1` | | |
| `kind` _string_ | `Bucket` | | |
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.37/#objectmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |  |  |
| `spec` _[BucketSpec](#bucketspec)_ | Spec defines the desired state of the Bucket. |  |  |
| `status` _[BucketStatus](#bucketstatus)_ | Status records the observed state of the Bucket and is maintained by the controller. |  | Optional: \{\} |

#### BucketAccess

BucketAccess grants a CloudPrincipal access to a Bucket through their shared
ProviderConfig.

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `apiVersion` _string_ | `vedro.svetoch.dev/v1alpha1` | | |
| `kind` _string_ | `BucketAccess` | | |
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.37/#objectmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |  |  |
| `spec` _[BucketAccessSpec](#bucketaccessspec)_ | Spec defines the desired state of the BucketAccess. |  |  |
| `status` _[BucketAccessStatus](#bucketaccessstatus)_ | Status records the observed state of the BucketAccess and is maintained by the controller. |  | Optional: \{\} |

#### BucketAccessLevel

_Underlying type:_ _string_

BucketAccessLevel selects a provider-neutral bucket permission level.

_Appears in:_

- [Access](#access)
- [BucketAccessProperties](#bucketaccessproperties)

| Value | Description |
| --- | --- |
| `ObjectReader` | ObjectReader maps to GCS roles/storage.objectViewer or Yandex Cloud storage.viewer on the bucket. |
| `ObjectWriter` | ObjectWriter maps to GCS roles/storage.objectCreator or Yandex Cloud storage.uploader on the bucket. It is not a combined reader and writer role. |
| `ObjectAdmin` | ObjectAdmin maps to GCS roles/storage.objectAdmin or Yandex Cloud storage.editor on the bucket. |
| `BucketAdmin` | BucketAdmin maps to GCS roles/storage.admin or Yandex Cloud storage.admin on the bucket. |

#### BucketAccessProperties

BucketAccessProperties records the cloud identities and access level of an applied grant.

_Appears in:_

- [BucketAccessStatus](#bucketaccessstatus)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `bucketName` _string_ | BucketName is the external name of the bucket on which access was granted. |  |  |
| `bucketId` _string_ | BucketId is the provider-side bucket identifier used for access management. It may be the same as BucketName. |  |  |
| `principalId` _string_ | PrincipalId is the provider-side identity to which access was granted. |  |  |
| `grantedAccess` _[BucketAccessLevel](#bucketaccesslevel)_ | GrantedAccess is the Vedro access level last applied to the bucket, rather than the provider-specific role name. |  |  |

#### BucketAccessSpec

BucketAccessSpec defines a permission grant between a Bucket and a CloudPrincipal.

_Appears in:_

- [BucketAccess](#bucketaccess)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `bucketRef` _[BucketReference](#bucketreference)_ | BucketRef identifies the Bucket on which access is granted. The reference, including its explicit namespace, is immutable. The bucket and principal must reference the same ProviderConfig. |  |  |
| `principalRef` _[PrincipalReference](#principalreference)_ | PrincipalRef identifies the CloudPrincipal receiving access, including its explicit namespace. Changing the principal causes the previously recorded grant to be revoked before the new grant is applied. |  |  |
| `deletionPolicy` _[DeletionPolicy](#deletionpolicy)_ | DeletionPolicy controls the grant when this Kubernetes resource is deleted. Delete revokes the recorded permission and is the default. Retain leaves the permission in place. Neither policy deletes the bucket or principal. | Delete | Enum: [Delete Retain]; Optional: \{\} |
| `access` _[Access](#access)_ | Access specifies the desired permission level on the referenced bucket. |  |  |

#### BucketAccessStatus

BucketAccessStatus records the applied grant and reconciliation state.

_Appears in:_

- [BucketAccess](#bucketaccess)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `applied` _[BucketAccessProperties](#bucketaccessproperties)_ | Applied records the external bucket, principal, and permission level from the last successful reconciliation. These values identify the grant to revoke during updates or deletion. |  | Optional: \{\} |
| `observedGeneration` _integer_ | ObservedGeneration is the latest metadata.generation observed by the controller. |  | Optional: \{\} |
| `unsupported` _[UnsupportedFeature](#unsupportedfeature) array_ | UnsupportedFeatures lists requested access levels that the selected provider does not support. |  | Optional: \{\} |
| `observedProvider` _string_ | ObservedProvider is the name of the ProviderConfig recorded during successful access reconciliation and used to revoke the grant during deletion. |  | Optional: \{\} |
| `conditions` _[Condition](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.37/#condition-v1-meta) array_ | Conditions represent the latest available observations of the BucketAccess state. |  | Optional: \{\} |

#### BucketCloudSpecificConfig

BucketCloudSpecificConfig contains settings that are specific to the selected cloud
provider.

_Appears in:_

- [BucketProperties](#bucketproperties)
- [BucketSpec](#bucketspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `gcp` _[BucketGcpConfig](#bucketgcpconfig)_ | Gcp configures Google Cloud Storage settings. It may only be set when the ProviderConfig type is gcp. |  | Optional: \{\} |
| `yc` _[BucketYcConfig](#bucketycconfig)_ | Yc reserves a configuration block for Yandex Cloud extensions. It currently has no settings and may only be set when the ProviderConfig type is yc. |  | Optional: \{\} |

#### BucketGcpConfig

BucketGcpConfig configures Google Cloud Storage features without a provider-neutral
equivalent.

_Appears in:_

- [BucketCloudSpecificConfig](#bucketcloudspecificconfig)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `softDeletePolicy` _[SoftDeletePolicy](#softdeletepolicy)_ | SoftDeletePolicy configures retention of soft-deleted objects. When omitted, the controller applies a seven-day retention period. Set retentionDuration to "0s" to disable soft delete. |  | Optional: \{\} |

#### BucketLifecycle

BucketLifecycle defines the desired object lifecycle rules for a bucket.

_Appears in:_

- [BucketProperties](#bucketproperties)
- [BucketSpec](#bucketspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `rules` _[BucketLifecycleRule](#bucketlifecyclerule) array_ | Rules is the desired set of object lifecycle rules. An empty list configures no lifecycle rules. Rule names and other capabilities depend on the selected provider. |  | Optional: \{\} |

#### BucketLifecycleAction

_Underlying type:_ _string_

BucketLifecycleAction is the operation performed by an object lifecycle rule.

_Appears in:_

- [BucketLifecycleRule](#bucketlifecyclerule)

| Value | Description |
| --- | --- |
| `Delete` | BucketLifecycleActionDelete requests deletion of objects matching the lifecycle rule. |

#### BucketLifecycleRule

BucketLifecycleRule selects objects by age and configures a lifecycle action.

_Appears in:_

- [BucketLifecycle](#bucketlifecycle)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `name` _string_ | Name is an optional stable identifier for this lifecycle rule. Yandex Cloud supports rule names; GCS does not. |  | MinLength: 1; Optional: \{\} |
| `enabled` _boolean_ | Enabled controls whether this rule is active. Defaults to true. Disabled rules are excluded from the lifecycle configuration sent to GCS. | true |  |
| `ageDays` _integer_ | AgeDays selects objects by age in days for the lifecycle action. When set, it must be at least 1. The provider determines how object age is evaluated. |  | Minimum: 1; Optional: \{\} |
| `action` _[BucketLifecycleAction](#bucketlifecycleaction)_ | Action is the operation to perform on matching objects. The only supported value is Delete; execution follows the cloud provider's lifecycle scheduling. |  | Enum: [Delete] |

#### BucketPolicySpec

BucketPolicySpec restricts the cloud buckets managed through a ProviderConfig.

_Appears in:_

- [UsagePolicySpec](#usagepolicyspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `allowedNamePatterns` _[AllowedNamePatterns](#allowednamepatterns)_ | AllowedNamePatterns contains Go regular expressions for cloud bucket names. Checks spec.name, falling back to metadata.name when spec.name is empty. During deletion, the recorded external name is used when available. At least one pattern must match; an empty list allows no names. Use ^ and $ to match the entire name. |  |  |

#### BucketProperties

BucketProperties records bucket settings returned by a provider during reconciliation.

_Appears in:_

- [BucketStatus](#bucketstatus)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `publicAccessPrevention` _boolean_ | PublicAccessPrevention records the provider-reported setting for preventing public access. |  |  |
| `versioning` _[BucketVersioning](#bucketversioning)_ | Versioning records the object versioning configuration reported or applied by the provider. |  |  |
| `lifecycle` _[BucketLifecycle](#bucketlifecycle)_ | Lifecycle records the lifecycle rules reported or applied by the provider. |  |  |
| `storageClass` _[BucketStorageClass](#bucketstorageclass)_ | StorageClass records the bucket storage class using Vedro storage class names. |  |  |
| `labels` _object (keys:string, values:string)_ | Labels records the cloud bucket labels or tags reported or applied by the provider. |  |  |
| `cloudSpecificConfig` _[BucketCloudSpecificConfig](#bucketcloudspecificconfig)_ | CloudSpecificConfig records provider-specific bucket settings reported or applied by the provider, including normalized defaults when available. |  |  |

#### BucketReference

BucketReference identifies a namespaced Bucket resource.

_Appears in:_

- [BucketAccessSpec](#bucketaccessspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `name` _string_ | Name is the Kubernetes resource name of the Bucket, not its external cloud name. |  | MinLength: 1 |
| `namespace` _string_ | Namespace is the Kubernetes namespace containing the Bucket. It must be specified explicitly, including for references within the same namespace. |  | MinLength: 1 |

#### BucketSpec

BucketSpec defines the desired cloud bucket and its lifecycle settings.

_Appears in:_

- [Bucket](#bucket)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `providerRef` _[ProviderConfigReference](#providerconfigreference)_ | ProviderRef selects the cluster-scoped ProviderConfig used to manage this bucket. The reference is immutable, and the bucket must satisfy the provider's usage policy. |  |  |
| `name` _string_ | Name is the external cloud bucket name. When omitted, metadata.name is used. The resolved name must satisfy provider naming rules and cannot change after the external name has been recorded in status. |  | MinLength: 1; Optional: \{\} |
| `location` _string_ | Location is the cloud bucket location, such as europe-west1 for GCS or ru-central1 for Yandex Cloud. GCS also accepts multi-regions such as EU and predefined dual-regions such as EUR4. An existing bucket must match the requested location; the controller does not relocate buckets. |  | MinLength: 1 |
| `storageClass` _[BucketStorageClass](#bucketstorageclass)_ | StorageClass sets the bucket's default storage class. Defaults to Standard. GCS maps Standard, Warm, Cold, and Ice to STANDARD, NEARLINE, COLDLINE, and ARCHIVE. Yandex Cloud supports Standard, Cold, and Ice, mapped to STANDARD, COLD, and ICE; Warm is unsupported. | Standard | Enum: [Standard Warm Cold Ice]; Optional: \{\} |
| `deletionPolicy` _[DeletionPolicy](#deletionpolicy)_ | DeletionPolicy controls cleanup when this Kubernetes resource is deleted. Retain leaves the external bucket and its objects intact and is the default. Delete removes object versions before deleting the bucket; cloud retention restrictions may prevent cleanup. | Retain | Enum: [Delete Retain]; Optional: \{\} |
| `publicAccessPrevention` _boolean_ | PublicAccessPrevention enforces prevention of public access when true. GCS treats false or omission as inherited public access prevention, which may still be enforced by a parent policy. Yandex Cloud does not support this setting. |  | Optional: \{\} |
| `versioning` _[BucketVersioning](#bucketversioning)_ | Versioning configures object versioning. Omission requests disabled versioning; existing object versions are not removed by disabling versioning. |  | Optional: \{\} |
| `lifecycle` _[BucketLifecycle](#bucketlifecycle)_ | Lifecycle configures object expiration rules. Omission or an empty rules list requests no lifecycle rules. Support for individual rule fields depends on the provider. |  | Optional: \{\} |
| `labels` _object (keys:string, values:string)_ | Labels is the desired map of cloud bucket labels or tags, separate from Kubernetes metadata.labels. The controller reconciles the entire map; labels absent from it are removed. |  | Optional: \{\} |
| `cloudSpecificConfig` _[BucketCloudSpecificConfig](#bucketcloudspecificconfig)_ | CloudSpecificConfig configures settings without a cloud-agnostic equivalent. Set only the block matching the selected ProviderConfig type; a block for another provider is rejected during reconciliation. GCS applies a seven-day soft-delete retention period when its soft-delete settings are omitted. |  |  |

#### BucketStatus

BucketStatus records the external bucket identity, applied settings, and reconciliation
state.

_Appears in:_

- [Bucket](#bucket)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `externalName` _string_ | ExternalName is the external cloud bucket name recorded after provisioning. It is also used to identify the bucket during deletion. |  | Optional: \{\} |
| `externalId` _string_ | ExternalId is the provider-side bucket identifier. It may be the same as ExternalName and is used when configuring bucket access. |  | Optional: \{\} |
| `location` _string_ | Location is the provider-reported bucket location, including a region, multi-region, or dual-region where supported. |  | Optional: \{\} |
| `observedProvider` _string_ | ObservedProvider is the name of the ProviderConfig recorded during successful bucket reconciliation and used for cleanup. |  | Optional: \{\} |
| `observedGeneration` _integer_ | ObservedGeneration is the latest metadata.generation observed by the controller. |  | Optional: \{\} |
| `applied` _[BucketProperties](#bucketproperties)_ | Applied records the bucket properties returned by the last successful provider reconciliation. Provider defaults and normalization may make these differ from the requested spec. |  | Optional: \{\} |
| `unsupported` _[UnsupportedFeature](#unsupportedfeature) array_ | UnsupportedFeatures lists requested settings that the selected provider cannot apply. Unsupported settings prevent normal reconciliation until the request is corrected. |  | Optional: \{\} |
| `conditions` _[Condition](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.37/#condition-v1-meta) array_ | Conditions represent the latest available observations of the bucket state. |  | Optional: \{\} |

#### BucketStorageClass

_Underlying type:_ _string_

BucketStorageClass is a provider-neutral storage tier. The provider maps each supported
value to its own storage class.

_Appears in:_

- [BucketProperties](#bucketproperties)
- [BucketSpec](#bucketspec)

| Value | Description |
| --- | --- |
| `Standard` | BucketStorageClassStandard maps to STANDARD on GCS and Yandex Cloud. |
| `Warm` | BucketStorageClassWarm maps to GCS NEARLINE and is unsupported by Yandex Cloud. |
| `Cold` | BucketStorageClassCold maps to GCS COLDLINE or Yandex Cloud COLD. |
| `Ice` | BucketStorageClassIce maps to GCS ARCHIVE or Yandex Cloud ICE. |

#### BucketVersioning

BucketVersioning configures preservation of object versions.

_Appears in:_

- [BucketProperties](#bucketproperties)
- [BucketSpec](#bucketspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `enabled` _boolean_ | Enabled enables object versioning when true and disables or suspends it when false, according to provider semantics. Disabling versioning does not remove existing object versions. |  |  |

#### BucketYcConfig

BucketYcConfig contains Yandex Cloud-specific bucket configuration.
It is currently empty and reserved for future extensions.

_Appears in:_

- [BucketCloudSpecificConfig](#bucketcloudspecificconfig)

#### CloudPrincipal

CloudPrincipal manages or references a cloud identity used for bucket access and
authentication.

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `apiVersion` _string_ | `vedro.svetoch.dev/v1alpha1` | | |
| `kind` _string_ | `CloudPrincipal` | | |
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.37/#objectmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |  |  |
| `spec` _[CloudPrincipalSpec](#cloudprincipalspec)_ | Spec defines the desired state of the CloudPrincipal. |  |  |
| `status` _[CloudPrincipalStatus](#cloudprincipalstatus)_ | Status records the observed state of the CloudPrincipal and is maintained by the controller. |  | Optional: \{\} |

#### CloudPrincipalAuth

CloudPrincipalAuth provisions authentication for a managed CloudPrincipal and publishes
or configures its Kubernetes authentication resources.

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `apiVersion` _string_ | `vedro.svetoch.dev/v1alpha1` | | |
| `kind` _string_ | `CloudPrincipalAuth` | | |
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.37/#objectmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |  |  |
| `spec` _[CloudPrincipalAuthSpec](#cloudprincipalauthspec)_ | Spec defines the desired state of the CloudPrincipalAuth. |  |  |
| `status` _[CloudPrincipalAuthStatus](#cloudprincipalauthstatus)_ | Status records the observed state of the CloudPrincipalAuth and is maintained by the controller. |  | Optional: \{\} |

#### CloudPrincipalAuthProperties

CloudPrincipalAuthProperties records the authentication resources created or configured by the controller.

_Appears in:_

- [CloudPrincipalAuthStatus](#cloudprincipalauthstatus)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `method` _[AuthMethod](#authmethod)_ | Method is the authentication method applied to the cloud principal. |  |  |
| `credentialsId` _string_ | CredentialsId is the provider-side identifier used to look up or revoke the issued credentials or workload identity binding. Its format depends on the provider and method. |  |  |
| `principalId` _string_ | PrincipalId is the provider-side identifier of the cloud principal. |  |  |
| `serviceAccountRef` _[NamespacedName](#namespacedname)_ | ServiceAccountRef identifies the configured Kubernetes ServiceAccount, if any. |  | Optional: \{\} |
| `secretRef` _[NamespacedName](#namespacedname)_ | SecretRef identifies the Kubernetes Secret containing static credentials, if any. |  | Optional: \{\} |
| `createdAt` _[Time](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.37/#time-v1-meta)_ | CreatedAt is the time the controller recorded creation of the current authentication material. It is refreshed when static credentials are recreated and is used to allow for GCP key propagation delays. |  |  |

#### CloudPrincipalAuthSpec

CloudPrincipalAuthSpec defines the desired authentication method for a CloudPrincipal.

_Appears in:_

- [CloudPrincipalAuth](#cloudprincipalauth)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `principalRef` _[PrincipalReference](#principalreference)_ | PrincipalRef identifies the CloudPrincipal for which authentication is configured, including its explicit namespace. The reference is immutable. The principal must use managementPolicy Managed. Both providers currently support authentication only for ServiceAccount principals. |  |  |
| `method` _[AuthMethod](#authmethod)_ | Method selects StaticCredentials to issue cloud credentials into a Secret, or WorkloadIdentity to configure a Kubernetes ServiceAccount. This field is immutable. Only the matching configuration block may be set. |  | Enum: [StaticCredentials WorkloadIdentity] |
| `staticCredentials` _[StaticCredentialsSpec](#staticcredentialsspec)_ | StaticCredentials configures credential issuance and the output Secret. Required when method is StaticCredentials and forbidden when method is WorkloadIdentity. |  | Optional: \{\} |
| `workloadIdentity` _[WorkloadIdentitySpec](#workloadidentityspec)_ | WorkloadIdentity configures the Kubernetes ServiceAccount and cloud identity binding. Required when method is WorkloadIdentity and forbidden when method is StaticCredentials. |  | Optional: \{\} |

#### CloudPrincipalAuthStatus

CloudPrincipalAuthStatus defines the observed state of a CloudPrincipalAuth.

_Appears in:_

- [CloudPrincipalAuth](#cloudprincipalauth)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `applied` _[CloudPrincipalAuthProperties](#cloudprincipalauthproperties)_ | Applied records the authentication method and resources applied by the controller. |  | Optional: \{\} |
| `conditions` _[Condition](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.37/#condition-v1-meta) array_ | Conditions represent the latest available observations of the CloudPrincipalAuth state. |  | Optional: \{\} |
| `observedGeneration` _integer_ | ObservedGeneration is the latest metadata.generation observed by the controller. |  | Optional: \{\} |
| `observedProvider` _string_ | ObservedProvider is the ProviderConfig used for the last successful reconciliation. |  | Optional: \{\} |
| `unsupported` _[UnsupportedFeature](#unsupportedfeature) array_ | UnsupportedFeatures lists requested features that the selected provider does not support. |  | Optional: \{\} |

#### CloudPrincipalSpec

CloudPrincipalSpec defines a cloud principal to manage or reference.

_Appears in:_

- [CloudPrincipal](#cloudprincipal)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `providerRef` _[ProviderConfigReference](#providerconfigreference)_ | ProviderRef selects the cluster-scoped ProviderConfig used to manage or resolve this principal. The reference is immutable, and the principal must satisfy the provider's usage policy. |  |  |
| `kind` _[PrincipalKind](#principalkind)_ | Kind selects ServiceAccount, User, Group, Role, or AllUsers. Both providers can manage ServiceAccount and reference ServiceAccount, User, and AllUsers. GCP additionally supports Group references. Role is currently unsupported by both providers. AllUsers represents unauthenticated public access and requires Reference management with both managed and reference omitted. |  | Enum: [ServiceAccount User Group Role AllUsers] |
| `managementPolicy` _[PrincipalManagementPolicy](#principalmanagementpolicy)_ | ManagementPolicy selects Managed to create and reconcile an external principal, or Reference to resolve an existing principal without managing its lifecycle. The policy is immutable. Both providers currently support Managed only for ServiceAccount. |  | Enum: [Managed Reference] |
| `managed` _[ManagedPrincipalSpec](#managedprincipalspec)_ | Managed supplies the name and deletion policy for an external principal created or reconciled by the controller. Required when managementPolicy is Managed and forbidden otherwise. |  | Optional: \{\} |
| `reference` _[ReferencedPrincipalSpec](#referencedprincipalspec)_ | Reference identifies an existing external principal. Required when managementPolicy is Reference, except for AllUsers, where it must be omitted. Forbidden when managementPolicy is Managed. |  | Optional: \{\} |

#### CloudPrincipalStatus

CloudPrincipalStatus defines the observed state of a CloudPrincipal.

_Appears in:_

- [CloudPrincipal](#cloudprincipal)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `externalName` _string_ | ExternalName is the provider-side principal name. |  | Optional: \{\} |
| `observedProvider` _string_ | ObservedProvider is the ProviderConfig used for the last successful reconciliation. |  | Optional: \{\} |
| `observedGeneration` _integer_ | ObservedGeneration is the latest metadata.generation observed by the controller. |  | Optional: \{\} |
| `unsupported` _[UnsupportedFeature](#unsupportedfeature) array_ | UnsupportedFeatures lists requested features that the selected provider does not support. |  | Optional: \{\} |
| `externalId` _string_ | ExternalId is the provider-side immutable principal identifier. |  | Optional: \{\} |
| `conditions` _[Condition](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.37/#condition-v1-meta) array_ | Conditions represent the latest available observations of the CloudPrincipal state. |  | Optional: \{\} |
| `kind` _[PrincipalKind](#principalkind)_ | Kind is the kind reported for the external principal. |  | Optional: \{\} |
| `managementPolicy` _[PrincipalManagementPolicy](#principalmanagementpolicy)_ | ManagementPolicy is the policy used to reconcile the external principal. |  | Optional: \{\} |

#### DeletionPolicy

_Underlying type:_ _string_

DeletionPolicy determines whether external resources or authentication material are
deleted or retained when their Kubernetes resource is deleted.

_Appears in:_

- [BucketAccessSpec](#bucketaccessspec)
- [BucketSpec](#bucketspec)
- [ManagedPrincipalSpec](#managedprincipalspec)
- [StaticCredentialsSpec](#staticcredentialsspec)
- [WorkloadIdentitySpec](#workloadidentityspec)

| Value | Description |
| --- | --- |
| `Delete` | DeletionPolicyDelete requests removal of the associated external resource or authentication material. |
| `Retain` | DeletionPolicyRetain preserves the associated external resource or authentication material after Kubernetes resource deletion. |

#### ManagedPrincipalSpec

ManagedPrincipalSpec defines an external principal whose lifecycle is managed by the
controller.

_Appears in:_

- [CloudPrincipalSpec](#cloudprincipalspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `name` _string_ | Name is the desired external principal name, subject to provider naming rules. It is required; the API does not default it from metadata.name. For a GCP ServiceAccount, use the short account ID rather than its email address. The resolved name cannot change after provisioning. |  | MinLength: 1 |
| `deletionPolicy` _[DeletionPolicy](#deletionpolicy)_ | DeletionPolicy controls cleanup of the managed external principal when the CloudPrincipal is deleted. Delete removes the external principal and is the default. Retain leaves it in the cloud provider. | Delete | Enum: [Delete Retain]; Optional: \{\} |

#### NamespacedName

NamespacedName identifies a Kubernetes object by name and namespace.

_Appears in:_

- [CloudPrincipalAuthProperties](#cloudprincipalauthproperties)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `name` _string_ | Name is the name of the referenced Kubernetes object. |  |  |
| `namespace` _string_ | Namespace is the namespace of the referenced Kubernetes object. |  |  |

#### PrincipalKind

_Underlying type:_ _string_

PrincipalKind identifies a cloud identity type. Support varies by provider and management
policy.

_Appears in:_

- [CloudPrincipalSpec](#cloudprincipalspec)
- [CloudPrincipalStatus](#cloudprincipalstatus)
- [PrincipalPolicySpec](#principalpolicyspec)

| Value | Description |
| --- | --- |
| `ServiceAccount` | PrincipalKindServiceAccount identifies a cloud service account. |
| `Role` | PrincipalKindRole identifies a cloud IAM role; support depends on the provider. |
| `User` | PrincipalKindUser identifies a cloud user identity. |
| `Group` | PrincipalKindGroup identifies a cloud group identity. |
| `AllUsers` | PrincipalKindAllUsers represents public access for all users, including unauthenticated users. It can only be referenced, not managed. |

#### PrincipalManagementPolicy

_Underlying type:_ _string_

PrincipalManagementPolicy selects whether the controller manages an external principal or
only resolves a reference to it.

_Appears in:_

- [CloudPrincipalSpec](#cloudprincipalspec)
- [CloudPrincipalStatus](#cloudprincipalstatus)

| Value | Description |
| --- | --- |
| `Managed` | PrincipalManagementPolicyManaged creates and reconciles the external principal according to spec.managed. |
| `Reference` | PrincipalManagementPolicyReference resolves an existing identity without creating or deleting it. |

#### PrincipalPolicySpec

PrincipalPolicySpec restricts managed and referenced cloud principals.

_Appears in:_

- [UsagePolicySpec](#usagepolicyspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `allowedNamePatterns` _[AllowedNamePatterns](#allowednamepatterns)_ | AllowedNamePatterns contains Go regular expressions for managed principal names from spec.managed.name. During deletion, the recorded external name is used when available. At least one pattern must match; an empty list allows no names. Use ^ and $ to match the entire name. |  |  |
| `allowedReferencePatterns` _[AllowedNamePatterns](#allowednamepatterns)_ | AllowedReferencePatterns contains Go regular expressions for referenced principal names from spec.reference.name. Defaults to [".*"], allowing any reference name. An empty list allows no reference names. Use ^ and $ to match the entire name. AllUsers has no reference name and is exempt from this check. | [.*] |  |
| `allowManaged` _boolean_ | AllowManaged permits CloudPrincipals with managementPolicy Managed. Defaults to true. Kind and name restrictions still apply. | true |  |
| `allowReferences` _boolean_ | AllowReferences permits CloudPrincipals with managementPolicy Reference, including AllUsers. Defaults to true. Kind and applicable reference-name restrictions still apply. | true |  |
| `allowedKinds` _[PrincipalKind](#principalkind) array_ | AllowedKinds lists permitted CloudPrincipal kinds for both management policies. Defaults to ServiceAccount, Role, User, Group, and AllUsers. An empty list allows no kinds. Provider capability restrictions still apply. During deletion, the recorded status.kind is checked. | [ServiceAccount Role User Group AllUsers] |  |

#### PrincipalReference

PrincipalReference identifies a namespaced CloudPrincipal resource.

_Appears in:_

- [BucketAccessSpec](#bucketaccessspec)
- [CloudPrincipalAuthSpec](#cloudprincipalauthspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `name` _string_ | Name is the Kubernetes resource name of the CloudPrincipal, not its external cloud name or identifier. |  | MinLength: 1 |
| `namespace` _string_ | Namespace is the Kubernetes namespace containing the CloudPrincipal. It must be specified explicitly, including for references within the same namespace. |  | MinLength: 1 |

#### ProviderConfig

ProviderConfig is a cluster-scoped resource defining cloud connectivity and policies for
resources that use it.

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `apiVersion` _string_ | `vedro.svetoch.dev/v1alpha1` | | |
| `kind` _string_ | `ProviderConfig` | | |
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.37/#objectmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |  |  |
| `spec` _[ProviderConfigSpec](#providerconfigspec)_ | Spec defines the desired state of the ProviderConfig. |  |  |
| `status` _[ProviderConfigStatus](#providerconfigstatus)_ | Status records the observed state of the ProviderConfig and is maintained by the controller. |  | Optional: \{\} |

#### ProviderConfigReference

ProviderConfigReference selects a cluster-scoped provider configuration.

_Appears in:_

- [BucketSpec](#bucketspec)
- [CloudPrincipalSpec](#cloudprincipalspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `name` _string_ | Name is the name of the cluster-scoped ProviderConfig. No namespace is required. |  | MinLength: 1 |

#### ProviderConfigSpec

ProviderConfigSpec defines cloud scope, operator authentication, and resource usage
restrictions.

_Appears in:_

- [ProviderConfig](#providerconfig)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `type` _[ProviderType](#providertype)_ | Type selects the cloud implementation: gcp for Google Cloud or yc for Yandex Cloud. This field is immutable. |  | Enum: [gcp yc] |
| `projectId` _string_ | ProjectId is the Google Cloud project ID for gcp or the Yandex Cloud folder ID for yc. It determines where managed cloud resources are created and is immutable. |  | MinLength: 1 |
| `region` _string_ | Region is the provider region, such as europe-west1 or ru-central1. It must be a non-empty regional name. Bucket placement is selected separately by Bucket.spec.location. |  |  |
| `method` _[AuthMethod](#authmethod)_ | Method selects how the operator authenticates to the provider. StaticCredentials reads a service account key from credentialsSecretRef. WorkloadIdentity uses the operator's ambient Google Cloud credentials or the Yandex Cloud instance service account. |  | Enum: [StaticCredentials WorkloadIdentity] |
| `credentialsSecretRef` _[SecretReference](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.37/#secretreference-v1-core)_ | CredentialsSecretRef identifies an existing Kubernetes Secret by name and namespace. Required for StaticCredentials and forbidden for WorkloadIdentity. The Secret must contain a key entry with a Google Cloud service account JSON key for gcp or a Yandex Cloud authorized service account key in JSON format for yc. |  | Optional: \{\} |
| `usagePolicy` _[UsagePolicySpec](#usagepolicyspec)_ | UsagePolicy defines namespace and resource restrictions for this provider. Resources must satisfy all applicable restrictions to reconcile. |  |  |

#### ProviderConfigStatus

ProviderConfigStatus defines the observed provider configuration state.

_Appears in:_

- [ProviderConfig](#providerconfig)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `conditions` _[Condition](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.37/#condition-v1-meta) array_ | Conditions represent the latest available observations of the ProviderConfig state. |  | Optional: \{\} |
| `observedGeneration` _integer_ | ObservedGeneration is the latest metadata.generation observed by the controller. |  | Optional: \{\} |

#### ProviderType

_Underlying type:_ _string_

ProviderType identifies a supported cloud implementation.

_Appears in:_

- [ProviderConfigSpec](#providerconfigspec)

| Value | Description |
| --- | --- |
| `gcp` | ProviderTypeGCP selects Google Cloud Storage and IAM. |
| `yc` | ProviderTypeYandexCloud selects Yandex Cloud Object Storage and IAM. |

#### ReferencedPrincipalSpec

ReferencedPrincipalSpec identifies an existing external principal without managing its
lifecycle.

_Appears in:_

- [CloudPrincipalSpec](#cloudprincipalspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `name` _string_ | Name identifies an existing external principal in the format expected by the provider and kind. For a GCP ServiceAccount, use its email address. For a Yandex Cloud ServiceAccount, use <folder-name>:<service-account-name>, such as prod:some-sa. User and GCP Group references use email addresses. References are resolved without creating or deleting the external principal. |  | MinLength: 1 |

#### SoftDeletePolicy

SoftDeletePolicy controls Google Cloud Storage retention of deleted objects.

_Appears in:_

- [BucketGcpConfig](#bucketgcpconfig)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `retentionDuration` _[Duration](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.37/#duration-v1-meta)_ | RetentionDuration is how long GCS retains soft-deleted objects before permanent deletion. Use "0s" to disable soft delete, or a Go duration representing a whole number of days from 7 through 90, such as "168h" for seven days. Defaults to "168h". | 168h |  |

#### StaticCredentialsSpec

StaticCredentialsSpec configures credentials stored in a Kubernetes Secret.

_Appears in:_

- [CloudPrincipalAuthSpec](#cloudprincipalauthspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `secretRef` _[AuthObjectReference](#authobjectreference)_ | SecretRef names the output Secret in the CloudPrincipalAuth namespace. The controller stores a GCP service account JSON key under credentials.json, or Yandex Cloud S3 credentials under AWS_ACCESS_KEY_ID and AWS_SECRET_ACCESS_KEY. The Secret is created and managed by the controller; it is not an input credential Secret. |  |  |
| `deletionPolicy` _[DeletionPolicy](#deletionpolicy)_ | DeletionPolicy controls cleanup when CloudPrincipalAuth is deleted. Delete revokes the cloud credentials and allows the managed Secret to be deleted. Retain preserves both the credentials and Secret, removing the Secret's owner references. Defaults to Retain. | Retain | Enum: [Delete Retain]; Optional: \{\} |

#### UnsupportedFeature

UnsupportedFeature describes a spec field that cannot be applied by the selected
provider.

_Appears in:_

- [BucketAccessStatus](#bucketaccessstatus)
- [BucketStatus](#bucketstatus)
- [CloudPrincipalAuthStatus](#cloudprincipalauthstatus)
- [CloudPrincipalStatus](#cloudprincipalstatus)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `field` _string_ | Field is the path to the unsupported field in the requested spec, including a list index when applicable. |  |  |
| `message` _string_ | Message explains why the selected provider cannot apply the requested feature. |  |  |
| `reason` _[UnsupportedFeatureReason](#unsupportedfeaturereason)_ | Reason is a machine-readable code identifying the unsupported capability. |  |  |

#### UnsupportedFeatureReason

_Underlying type:_ _string_

UnsupportedFeatureReason identifies a requested capability that the selected provider
does not support.

_Appears in:_

- [UnsupportedFeature](#unsupportedfeature)

| Value | Description |
| --- | --- |
| `BucketUnsupportedVersioning` | BucketUnsupportedVersioning indicates that the selected provider does not support object versioning. |
| `BucketUnsupportedLabels` | BucketUnsupportedLabels indicates that the selected provider does not support bucket labels or tags. |
| `BucketUnsupportedPublicAccessPrevention` | BucketUnsupportedPublicAccessPrevention indicates that the selected provider does not support public access prevention. |
| `BucketUnsupportedStorageClass` | BucketUnsupportedStorageClass indicates that the selected provider does not support the requested storage class. |
| `BucketUnsupportedLifecycleExpiration` | BucketUnsupportedLifecycleExpiration indicates that the selected provider does not support age-based lifecycle expiration. |
| `BucketUnsupportedLifecycle` | BucketUnsupportedLifecycle indicates that the selected provider does not support object lifecycle rules. |
| `BucketUnsupportedLifecycleNamed` | BucketUnsupportedLifecycleNamed indicates that the selected provider does not support lifecycle rule names. |
| `BucketUnsupportedLifecycleEnabled` | BucketUnsupportedLifecycleEnabled indicates that the selected provider does not support lifecycle rule enablement. |
| `BucketAccessUnsupportedObjectReader` | BucketAccessUnsupportedObjectReader indicates that the selected provider does not support ObjectReader grants. |
| `BucketAccessUnsupportedObjectWriter` | BucketAccessUnsupportedObjectWriter indicates that the selected provider does not support ObjectWriter grants. |
| `BucketAccessUnsupportedObjectAdmin` | BucketAccessUnsupportedObjectAdmin indicates that the selected provider does not support ObjectAdmin grants. |
| `BucketAccessUnsupportedBucketAdmin` | BucketAccessUnsupportedBucketAdmin indicates that the selected provider does not support BucketAdmin grants. |
| `PrincipalAuthUnsupportedStaticCredentials` | PrincipalAuthUnsupportedStaticCredentials indicates that the provider does not support static credentials. |
| `PrincipalAuthUnsupportedWorkloadIdentity` | PrincipalAuthUnsupportedWorkloadIdentity indicates that the provider does not support workload identity. |
| `PrincipalAuthUnsupportedWorkloadIdentityKind` | PrincipalAuthUnsupportedWorkloadIdentityKind indicates that workload identity is unavailable for this principal kind. |
| `PrincipalAuthUnsupportedStaticCredentialsKind` | PrincipalAuthUnsupportedStaticCredentialsKind indicates that static credentials are unavailable for this principal kind. |
| `PrincipalUnsupportedManagedSA` | PrincipalUnsupportedManagedSA indicates that the selected provider does not support management of service accounts. |
| `PrincipalUnsupportedManagedUser` | PrincipalUnsupportedManagedUser indicates that the selected provider does not support management of users. |
| `PrincipalUnsupportedManagedRole` | PrincipalUnsupportedManagedRole indicates that the selected provider does not support management of roles. |
| `PrincipalUnsupportedManagedAllUsers` | PrincipalUnsupportedManagedAllUsers indicates that the selected provider does not support management of the AllUsers identity. |
| `PrincipalUnsupportedManagedGroup` | PrincipalUnsupportedManagedGroup indicates that the selected provider does not support management of groups. |
| `PrincipalUnsupportedReferencedSA` | PrincipalUnsupportedReferencedSA indicates that the selected provider does not support references to service accounts. |
| `PrincipalUnsupportedReferencedUser` | PrincipalUnsupportedReferencedUser indicates that the selected provider does not support references to users. |
| `PrincipalUnsupportedReferencedRole` | PrincipalUnsupportedReferencedRole indicates that the selected provider does not support references to roles. |
| `PrincipalUnsupportedReferencedGroup` | PrincipalUnsupportedReferencedGroup indicates that the selected provider does not support references to groups. |
| `PrincipalUnsupportedReferencedAllUsers` | PrincipalUnsupportedReferencedAllUsers indicates that the selected provider does not support references to the AllUsers identity. |

#### UsagePolicySpec

UsagePolicySpec controls which resources may use a ProviderConfig.

_Appears in:_

- [ProviderConfigSpec](#providerconfigspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `allowedNamespaces` _[AllowedNamespacesSpec](#allowednamespacesspec)_ | AllowedNamespaces restricts the namespaces of Bucket, CloudPrincipal, BucketAccess, and CloudPrincipalAuth resources using this provider. |  |  |
| `bucketPolicy` _[BucketPolicySpec](#bucketpolicyspec)_ | BucketPolicy restricts cloud bucket names managed through this provider. |  |  |
| `principalPolicy` _[PrincipalPolicySpec](#principalpolicyspec)_ | PrincipalPolicy restricts principal kinds, management policies, and names used through this provider. |  |  |

#### WorkloadIdentitySpec

WorkloadIdentitySpec configures authentication through a Kubernetes ServiceAccount.

_Appears in:_

- [CloudPrincipalAuthSpec](#cloudprincipalauthspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `serviceAccountRef` _[AuthObjectReference](#authobjectreference)_ | ServiceAccountRef names an existing Kubernetes ServiceAccount in the CloudPrincipalAuth namespace. For GCP, the controller grants workload identity access to the cloud service account and configures the Kubernetes ServiceAccount annotation. Workload identity for principals is currently unsupported by the Yandex Cloud provider. |  |  |
| `deletionPolicy` _[DeletionPolicy](#deletionpolicy)_ | DeletionPolicy controls cleanup when CloudPrincipalAuth is deleted. Delete removes the cloud authentication binding and managed ServiceAccount annotation. Retain preserves the configuration. The Kubernetes ServiceAccount itself is not deleted. Defaults to Retain. | Retain | Enum: [Delete Retain]; Optional: \{\} |
