# API Reference

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
| `level` _[BucketAccessLevel](#bucketaccesslevel)_ | Level selects ObjectReader, ObjectWriter, ObjectAdmin, or BucketAdmin. The selected<br />level is mapped to a provider-specific role on the bucket. |  | Enum: [ObjectReader ObjectWriter ObjectAdmin BucketAdmin] <br /> |


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
| `names` _string array_ | Names lists exact Kubernetes namespace names allowed to use this provider.<br />Must be empty when all is true. If all is false and names is empty,<br />no namespaces are allowed. |  | Optional: \{\} <br /> |
| `all` _boolean_ | All allows every Kubernetes namespace to use this provider.<br />Defaults to false and cannot be combined with a non-empty names list. |  | Optional: \{\} <br /> |


#### AuthMethod

_Underlying type:_ _string_

AuthMethod selects static credentials or workload identity authentication. ProviderConfig
uses it for operator credentials; CloudPrincipalAuth uses it for application credentials.



_Appears in:_
- [CloudPrincipalAuthProperties](#cloudprincipalauthproperties)
- [CloudPrincipalAuthSpec](#cloudprincipalauthspec)
- [ProviderConfigSpec](#providerconfigspec)

| Field | Description |
| --- | --- |
| `StaticCredentials` | AuthMethodStaticCredentials uses a provider key or issues static application<br />credentials, depending on the resource.<br /> |
| `WorkloadIdentity` | AuthMethodWorkloadIdentity uses ambient operator credentials or configures a Kubernetes<br />workload identity, depending on the resource.<br /> |


#### AuthObjectReference



AuthObjectReference identifies a Kubernetes object by name within the CloudPrincipalAuth namespace.



_Appears in:_
- [StaticCredentialsSpec](#staticcredentialsspec)
- [WorkloadIdentitySpec](#workloadidentityspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `name` _string_ | Name is the referenced Kubernetes object name in the CloudPrincipalAuth namespace. The<br />name is immutable. |  | MinLength: 1 <br /> |


#### Bucket



Bucket manages a cloud object storage bucket through a ProviderConfig.





| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `apiVersion` _string_ | `vedro.svetoch.dev/v1alpha1` | | |
| `kind` _string_ | `Bucket` | | |
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.37/#objectmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |  |  |
| `spec` _[BucketSpec](#bucketspec)_ | Spec defines the desired state of the Bucket. |  |  |
| `status` _[BucketStatus](#bucketstatus)_ | Status records the observed state of the Bucket and is maintained by the controller. |  | Optional: \{\} <br /> |


#### BucketAccess



BucketAccess grants a CloudPrincipal access to a Bucket through their shared
ProviderConfig.





| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `apiVersion` _string_ | `vedro.svetoch.dev/v1alpha1` | | |
| `kind` _string_ | `BucketAccess` | | |
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.37/#objectmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |  |  |
| `spec` _[BucketAccessSpec](#bucketaccessspec)_ | Spec defines the desired state of the BucketAccess. |  |  |
| `status` _[BucketAccessStatus](#bucketaccessstatus)_ | Status records the observed state of the BucketAccess and is maintained by the<br />controller. |  | Optional: \{\} <br /> |


#### BucketAccessLevel

_Underlying type:_ _string_

BucketAccessLevel selects a provider-neutral bucket permission level.



_Appears in:_
- [Access](#access)
- [BucketAccessProperties](#bucketaccessproperties)

| Field | Description |
| --- | --- |
| `ObjectReader` | ObjectReader maps to GCS roles/storage.objectViewer or Yandex Cloud storage.viewer on<br />the bucket.<br /> |
| `ObjectWriter` | ObjectWriter maps to GCS roles/storage.objectCreator or Yandex Cloud storage.uploader on<br />the bucket. It is not a combined reader and writer role.<br /> |
| `ObjectAdmin` | ObjectAdmin maps to GCS roles/storage.objectAdmin or Yandex Cloud storage.editor on the<br />bucket.<br /> |
| `BucketAdmin` | BucketAdmin maps to GCS roles/storage.admin or Yandex Cloud storage.admin on the bucket.<br /> |


#### BucketAccessProperties



BucketAccessProperties records the cloud identities and access level of an applied grant.



_Appears in:_
- [BucketAccessStatus](#bucketaccessstatus)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `bucketName` _string_ | BucketName is the external name of the bucket on which access was granted. |  |  |
| `bucketId` _string_ | BucketId is the provider-side bucket identifier used for access management. It may be<br />the same as BucketName. |  |  |
| `principalId` _string_ | PrincipalId is the provider-side identity to which access was granted. |  |  |
| `grantedAccess` _[BucketAccessLevel](#bucketaccesslevel)_ | GrantedAccess is the Vedro access level last applied to the bucket, rather than the<br />provider-specific role name. |  |  |


#### BucketAccessSpec



BucketAccessSpec defines a permission grant between a Bucket and a CloudPrincipal.



_Appears in:_
- [BucketAccess](#bucketaccess)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `bucketRef` _[BucketReference](#bucketreference)_ | BucketRef identifies the Bucket on which access is granted. The reference, including its<br />explicit namespace, is immutable. The bucket and principal must reference the same<br />ProviderConfig. |  |  |
| `principalRef` _[PrincipalReference](#principalreference)_ | PrincipalRef identifies the CloudPrincipal receiving access, including its explicit<br />namespace. Changing the principal causes the previously recorded grant to be revoked<br />before the new grant is applied. |  |  |
| `deletionPolicy` _[DeletionPolicy](#deletionpolicy)_ | DeletionPolicy controls the grant when this Kubernetes resource is deleted. Delete<br />revokes the recorded permission and is the default. Retain leaves the permission in<br />place. Neither policy deletes the bucket or principal. | Delete | Enum: [Delete Retain] <br />Optional: \{\} <br /> |
| `access` _[Access](#access)_ | Access specifies the desired permission level on the referenced bucket. |  |  |


#### BucketAccessStatus



BucketAccessStatus records the applied grant and reconciliation state.



_Appears in:_
- [BucketAccess](#bucketaccess)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `applied` _[BucketAccessProperties](#bucketaccessproperties)_ | Applied records the external bucket, principal, and permission level from the last<br />successful reconciliation. These values identify the grant to revoke during updates or<br />deletion. |  | Optional: \{\} <br /> |
| `observedGeneration` _integer_ | ObservedGeneration is the latest metadata.generation observed by the controller. |  | Optional: \{\} <br /> |
| `unsupported` _[UnsupportedFeature](#unsupportedfeature) array_ | UnsupportedFeatures lists requested access levels that the selected provider does not<br />support. |  | Optional: \{\} <br /> |
| `observedProvider` _string_ | ObservedProvider is the name of the ProviderConfig recorded during successful access<br />reconciliation and used to revoke the grant during deletion. |  | Optional: \{\} <br /> |
| `conditions` _[Condition](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.37/#condition-v1-meta) array_ | Conditions represent the latest available observations of the BucketAccess state. |  | Optional: \{\} <br /> |


#### BucketCloudSpecificConfig



BucketCloudSpecificConfig contains settings that are specific to the selected cloud
provider.



_Appears in:_
- [BucketProperties](#bucketproperties)
- [BucketSpec](#bucketspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `gcp` _[BucketGcpConfig](#bucketgcpconfig)_ | Gcp configures Google Cloud Storage settings. It may only be set when the ProviderConfig<br />type is gcp. |  | Optional: \{\} <br /> |
| `yc` _[BucketYcConfig](#bucketycconfig)_ | Yc reserves a configuration block for Yandex Cloud extensions. It currently has no<br />settings and may only be set when the ProviderConfig type is yc. |  | Optional: \{\} <br /> |


#### BucketGcpConfig



BucketGcpConfig configures Google Cloud Storage features without a provider-neutral
equivalent.



_Appears in:_
- [BucketCloudSpecificConfig](#bucketcloudspecificconfig)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `softDeletePolicy` _[SoftDeletePolicy](#softdeletepolicy)_ | SoftDeletePolicy configures retention of soft-deleted objects. When omitted, the<br />controller applies a seven-day retention period. Set retentionDuration to "0s" to<br />disable soft delete. |  | Optional: \{\} <br /> |


#### BucketLifecycle



BucketLifecycle defines the desired object lifecycle rules for a bucket.



_Appears in:_
- [BucketProperties](#bucketproperties)
- [BucketSpec](#bucketspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `rules` _[BucketLifecycleRule](#bucketlifecyclerule) array_ | Rules is the desired set of object lifecycle rules. An empty list configures no<br />lifecycle rules. Rule names and other capabilities depend on the selected provider. |  | Optional: \{\} <br /> |


#### BucketLifecycleAction

_Underlying type:_ _string_

BucketLifecycleAction is the operation performed by an object lifecycle rule.



_Appears in:_
- [BucketLifecycleRule](#bucketlifecyclerule)

| Field | Description |
| --- | --- |
| `Delete` | BucketLifecycleActionDelete requests deletion of objects matching the lifecycle rule.<br /> |


#### BucketLifecycleRule



BucketLifecycleRule selects objects by age and configures a lifecycle action.



_Appears in:_
- [BucketLifecycle](#bucketlifecycle)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `name` _string_ | Name is an optional stable identifier for this lifecycle rule. Yandex Cloud supports<br />rule names; GCS does not. |  | MinLength: 1 <br />Optional: \{\} <br /> |
| `enabled` _boolean_ | Enabled controls whether this rule is active. Defaults to true. Disabled rules are<br />excluded from the lifecycle configuration sent to GCS. | true |  |
| `ageDays` _integer_ | AgeDays selects objects by age in days for the lifecycle action. When set, it must be at<br />least 1. The provider determines how object age is evaluated. |  | Minimum: 1 <br />Optional: \{\} <br /> |
| `action` _[BucketLifecycleAction](#bucketlifecycleaction)_ | Action is the operation to perform on matching objects. The only supported value is<br />Delete; execution follows the cloud provider's lifecycle scheduling. |  | Enum: [Delete] <br /> |


#### BucketPolicySpec



BucketPolicySpec restricts the cloud buckets managed through a ProviderConfig.



_Appears in:_
- [UsagePolicySpec](#usagepolicyspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `allowedNamePatterns` _[AllowedNamePatterns](#allowednamepatterns)_ | AllowedNamePatterns contains Go regular expressions for cloud bucket names.<br />Checks spec.name, falling back to metadata.name when spec.name is empty.<br />During deletion, the recorded external name is used when available.<br />At least one pattern must match; an empty list allows no names.<br />Use ^ and $ to match the entire name. |  |  |


#### BucketProperties



BucketProperties records bucket settings returned by a provider during reconciliation.



_Appears in:_
- [BucketStatus](#bucketstatus)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `publicAccessPrevention` _boolean_ | PublicAccessPrevention records the provider-reported setting for preventing public<br />access. |  |  |
| `versioning` _[BucketVersioning](#bucketversioning)_ | Versioning records the object versioning configuration reported or applied by the<br />provider. |  |  |
| `lifecycle` _[BucketLifecycle](#bucketlifecycle)_ | Lifecycle records the lifecycle rules reported or applied by the provider. |  |  |
| `storageClass` _[BucketStorageClass](#bucketstorageclass)_ | StorageClass records the bucket storage class using Vedro storage class names. |  |  |
| `labels` _object (keys:string, values:string)_ | Labels records the cloud bucket labels or tags reported or applied by the provider. |  |  |
| `cloudSpecificConfig` _[BucketCloudSpecificConfig](#bucketcloudspecificconfig)_ | CloudSpecificConfig records provider-specific bucket settings reported or applied by the<br />provider, including normalized defaults when available. |  |  |


#### BucketReference



BucketReference identifies a namespaced Bucket resource.



_Appears in:_
- [BucketAccessSpec](#bucketaccessspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `name` _string_ | Name is the Kubernetes resource name of the Bucket, not its external cloud name. |  | MinLength: 1 <br /> |
| `namespace` _string_ | Namespace is the Kubernetes namespace containing the Bucket. It must be specified<br />explicitly, including for references within the same namespace. |  | MinLength: 1 <br /> |


#### BucketSpec



BucketSpec defines the desired cloud bucket and its lifecycle settings.



_Appears in:_
- [Bucket](#bucket)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `providerRef` _[ProviderConfigReference](#providerconfigreference)_ | ProviderRef selects the cluster-scoped ProviderConfig used to manage this bucket. The<br />reference is immutable, and the bucket must satisfy the provider's usage policy. |  |  |
| `name` _string_ | Name is the external cloud bucket name. When omitted, metadata.name is used. The<br />resolved name must satisfy provider naming rules and cannot change after the external<br />name has been recorded in status. |  | MinLength: 1 <br />Optional: \{\} <br /> |
| `location` _string_ | Location is the cloud bucket location, such as europe-west1 for GCS or ru-central1 for<br />Yandex Cloud. GCS also accepts multi-regions such as EU and predefined dual-regions such<br />as EUR4. An existing bucket must match the requested location; the controller does not<br />relocate buckets. |  | MinLength: 1 <br /> |
| `storageClass` _[BucketStorageClass](#bucketstorageclass)_ | StorageClass sets the bucket's default storage class. Defaults to Standard. GCS maps<br />Standard, Warm, Cold, and Ice to STANDARD, NEARLINE, COLDLINE, and ARCHIVE. Yandex Cloud<br />supports Standard, Cold, and Ice, mapped to STANDARD, COLD, and ICE; Warm is<br />unsupported. | Standard | Enum: [Standard Warm Cold Ice] <br />Optional: \{\} <br /> |
| `deletionPolicy` _[DeletionPolicy](#deletionpolicy)_ | DeletionPolicy controls cleanup when this Kubernetes resource is deleted. Retain leaves<br />the external bucket and its objects intact and is the default. Delete removes object<br />versions before deleting the bucket; cloud retention restrictions may prevent cleanup. | Retain | Enum: [Delete Retain] <br />Optional: \{\} <br /> |
| `publicAccessPrevention` _boolean_ | PublicAccessPrevention enforces prevention of public access when true. GCS treats false<br />or omission as inherited public access prevention, which may still be enforced by a<br />parent policy. Yandex Cloud does not support this setting. |  | Optional: \{\} <br /> |
| `versioning` _[BucketVersioning](#bucketversioning)_ | Versioning configures object versioning. Omission requests disabled versioning; existing<br />object versions are not removed by disabling versioning. |  | Optional: \{\} <br /> |
| `lifecycle` _[BucketLifecycle](#bucketlifecycle)_ | Lifecycle configures object expiration rules. Omission or an empty rules list requests<br />no lifecycle rules. Support for individual rule fields depends on the provider. |  | Optional: \{\} <br /> |
| `labels` _object (keys:string, values:string)_ | Labels is the desired map of cloud bucket labels or tags, separate from Kubernetes<br />metadata.labels. The controller reconciles the entire map; labels absent from it are<br />removed. |  | Optional: \{\} <br /> |
| `cloudSpecificConfig` _[BucketCloudSpecificConfig](#bucketcloudspecificconfig)_ | CloudSpecificConfig configures settings without a cloud-agnostic equivalent. Set only<br />the block matching the selected ProviderConfig type; a block for another provider is<br />rejected during reconciliation. GCS applies a seven-day soft-delete retention period<br />when its soft-delete settings are omitted. |  |  |


#### BucketStatus



BucketStatus records the external bucket identity, applied settings, and reconciliation
state.



_Appears in:_
- [Bucket](#bucket)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `externalName` _string_ | ExternalName is the external cloud bucket name recorded after provisioning. It is also<br />used to identify the bucket during deletion. |  | Optional: \{\} <br /> |
| `externalId` _string_ | ExternalId is the provider-side bucket identifier. It may be the same as ExternalName<br />and is used when configuring bucket access. |  | Optional: \{\} <br /> |
| `location` _string_ | Location is the provider-reported bucket location, including a region, multi-region,<br />or dual-region where supported. |  | Optional: \{\} <br /> |
| `observedProvider` _string_ | ObservedProvider is the name of the ProviderConfig recorded during successful bucket<br />reconciliation and used for cleanup. |  | Optional: \{\} <br /> |
| `observedGeneration` _integer_ | ObservedGeneration is the latest metadata.generation observed by the controller. |  | Optional: \{\} <br /> |
| `applied` _[BucketProperties](#bucketproperties)_ | Applied records the bucket properties returned by the last successful provider<br />reconciliation. Provider defaults and normalization may make these differ from the<br />requested spec. |  | Optional: \{\} <br /> |
| `unsupported` _[UnsupportedFeature](#unsupportedfeature) array_ | UnsupportedFeatures lists requested settings that the selected provider cannot apply.<br />Unsupported settings prevent normal reconciliation until the request is corrected. |  | Optional: \{\} <br /> |
| `conditions` _[Condition](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.37/#condition-v1-meta) array_ | Conditions represent the latest available observations of the bucket state. |  | Optional: \{\} <br /> |


#### BucketStorageClass

_Underlying type:_ _string_

BucketStorageClass is a provider-neutral storage tier. The provider maps each supported
value to its own storage class.



_Appears in:_
- [BucketProperties](#bucketproperties)
- [BucketSpec](#bucketspec)

| Field | Description |
| --- | --- |
| `Standard` | BucketStorageClassStandard maps to STANDARD on GCS and Yandex Cloud.<br /> |
| `Warm` | BucketStorageClassWarm maps to GCS NEARLINE and is unsupported by Yandex Cloud.<br /> |
| `Cold` | BucketStorageClassCold maps to GCS COLDLINE or Yandex Cloud COLD.<br /> |
| `Ice` | BucketStorageClassIce maps to GCS ARCHIVE or Yandex Cloud ICE.<br /> |


#### BucketVersioning



BucketVersioning configures preservation of object versions.



_Appears in:_
- [BucketProperties](#bucketproperties)
- [BucketSpec](#bucketspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `enabled` _boolean_ | Enabled enables object versioning when true and disables or suspends it when false,<br />according to provider semantics. Disabling versioning does not remove existing object<br />versions. |  |  |


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
| `status` _[CloudPrincipalStatus](#cloudprincipalstatus)_ | Status records the observed state of the CloudPrincipal and is maintained by the<br />controller. |  | Optional: \{\} <br /> |


#### CloudPrincipalAuth



CloudPrincipalAuth provisions authentication for a managed CloudPrincipal and publishes
or configures its Kubernetes authentication resources.





| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `apiVersion` _string_ | `vedro.svetoch.dev/v1alpha1` | | |
| `kind` _string_ | `CloudPrincipalAuth` | | |
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.37/#objectmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |  |  |
| `spec` _[CloudPrincipalAuthSpec](#cloudprincipalauthspec)_ | Spec defines the desired state of the CloudPrincipalAuth. |  |  |
| `status` _[CloudPrincipalAuthStatus](#cloudprincipalauthstatus)_ | Status records the observed state of the CloudPrincipalAuth and is maintained by the<br />controller. |  | Optional: \{\} <br /> |


#### CloudPrincipalAuthProperties



CloudPrincipalAuthProperties records the authentication resources created or configured by the controller.



_Appears in:_
- [CloudPrincipalAuthStatus](#cloudprincipalauthstatus)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `method` _[AuthMethod](#authmethod)_ | Method is the authentication method applied to the cloud principal. |  |  |
| `credentialsId` _string_ | CredentialsId is the provider-side identifier used to look up or revoke the issued<br />credentials or workload identity binding. Its format depends on the provider and method. |  |  |
| `principalId` _string_ | PrincipalId is the provider-side identifier of the cloud principal. |  |  |
| `serviceAccountRef` _[NamespacedName](#namespacedname)_ | ServiceAccountRef identifies the configured Kubernetes ServiceAccount, if any. |  | Optional: \{\} <br /> |
| `secretRef` _[NamespacedName](#namespacedname)_ | SecretRef identifies the Kubernetes Secret containing static credentials, if any. |  | Optional: \{\} <br /> |
| `createdAt` _[Time](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.37/#time-v1-meta)_ | CreatedAt is the time the controller recorded creation of the current authentication<br />material. It is refreshed when static credentials are recreated and is used to allow for<br />GCP key propagation delays. |  |  |


#### CloudPrincipalAuthSpec



CloudPrincipalAuthSpec defines the desired authentication method for a CloudPrincipal.



_Appears in:_
- [CloudPrincipalAuth](#cloudprincipalauth)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `principalRef` _[PrincipalReference](#principalreference)_ | PrincipalRef identifies the CloudPrincipal for which authentication is configured,<br />including its explicit namespace. The reference is immutable. The principal must use<br />managementPolicy Managed. Both providers currently support authentication only for<br />ServiceAccount principals. |  |  |
| `method` _[AuthMethod](#authmethod)_ | Method selects StaticCredentials to issue cloud credentials into a Secret, or<br />WorkloadIdentity to configure a Kubernetes ServiceAccount. This field is immutable. Only<br />the matching configuration block may be set. |  | Enum: [StaticCredentials WorkloadIdentity] <br /> |
| `staticCredentials` _[StaticCredentialsSpec](#staticcredentialsspec)_ | StaticCredentials configures credential issuance and the output Secret. Required when<br />method is StaticCredentials and forbidden when method is WorkloadIdentity. |  | Optional: \{\} <br /> |
| `workloadIdentity` _[WorkloadIdentitySpec](#workloadidentityspec)_ | WorkloadIdentity configures the Kubernetes ServiceAccount and cloud identity binding.<br />Required when method is WorkloadIdentity and forbidden when method is StaticCredentials. |  | Optional: \{\} <br /> |


#### CloudPrincipalAuthStatus



CloudPrincipalAuthStatus defines the observed state of a CloudPrincipalAuth.



_Appears in:_
- [CloudPrincipalAuth](#cloudprincipalauth)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `applied` _[CloudPrincipalAuthProperties](#cloudprincipalauthproperties)_ | Applied records the authentication method and resources applied by the controller. |  | Optional: \{\} <br /> |
| `conditions` _[Condition](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.37/#condition-v1-meta) array_ | Conditions represent the latest available observations of the CloudPrincipalAuth state. |  | Optional: \{\} <br /> |
| `observedGeneration` _integer_ | ObservedGeneration is the latest metadata.generation observed by the controller. |  | Optional: \{\} <br /> |
| `observedProvider` _string_ | ObservedProvider is the ProviderConfig used for the last successful reconciliation. |  | Optional: \{\} <br /> |
| `unsupported` _[UnsupportedFeature](#unsupportedfeature) array_ | UnsupportedFeatures lists requested features that the selected provider does not support. |  | Optional: \{\} <br /> |


#### CloudPrincipalSpec



CloudPrincipalSpec defines a cloud principal to manage or reference.



_Appears in:_
- [CloudPrincipal](#cloudprincipal)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `providerRef` _[ProviderConfigReference](#providerconfigreference)_ | ProviderRef selects the cluster-scoped ProviderConfig used to manage or resolve this<br />principal. The reference is immutable, and the principal must satisfy the provider's<br />usage policy. |  |  |
| `kind` _[PrincipalKind](#principalkind)_ | Kind selects ServiceAccount, User, Group, Role, or AllUsers. Both providers can manage<br />ServiceAccount and reference ServiceAccount, User, and AllUsers. GCP additionally<br />supports Group references. Role is currently unsupported by both providers.<br />AllUsers represents unauthenticated public access and requires Reference management<br />with both managed and reference omitted. |  | Enum: [ServiceAccount User Group Role AllUsers] <br /> |
| `managementPolicy` _[PrincipalManagementPolicy](#principalmanagementpolicy)_ | ManagementPolicy selects Managed to create and reconcile an external principal, or<br />Reference to resolve an existing principal without managing its lifecycle. The policy is<br />immutable. Both providers currently support Managed only for ServiceAccount. |  | Enum: [Managed Reference] <br /> |
| `managed` _[ManagedPrincipalSpec](#managedprincipalspec)_ | Managed supplies the name and deletion policy for an external principal created or<br />reconciled by the controller. Required when managementPolicy is Managed and forbidden<br />otherwise. |  | Optional: \{\} <br /> |
| `reference` _[ReferencedPrincipalSpec](#referencedprincipalspec)_ | Reference identifies an existing external principal. Required when managementPolicy is<br />Reference, except for AllUsers, where it must be omitted. Forbidden when<br />managementPolicy is Managed. |  | Optional: \{\} <br /> |


#### CloudPrincipalStatus



CloudPrincipalStatus defines the observed state of a CloudPrincipal.



_Appears in:_
- [CloudPrincipal](#cloudprincipal)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `externalName` _string_ | ExternalName is the provider-side principal name. |  | Optional: \{\} <br /> |
| `observedProvider` _string_ | ObservedProvider is the ProviderConfig used for the last successful reconciliation. |  | Optional: \{\} <br /> |
| `observedGeneration` _integer_ | ObservedGeneration is the latest metadata.generation observed by the controller. |  | Optional: \{\} <br /> |
| `unsupported` _[UnsupportedFeature](#unsupportedfeature) array_ | UnsupportedFeatures lists requested features that the selected provider does not support. |  | Optional: \{\} <br /> |
| `externalId` _string_ | ExternalId is the provider-side immutable principal identifier. |  | Optional: \{\} <br /> |
| `conditions` _[Condition](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.37/#condition-v1-meta) array_ | Conditions represent the latest available observations of the CloudPrincipal state. |  | Optional: \{\} <br /> |
| `kind` _[PrincipalKind](#principalkind)_ | Kind is the kind reported for the external principal. |  | Optional: \{\} <br /> |
| `managementPolicy` _[PrincipalManagementPolicy](#principalmanagementpolicy)_ | ManagementPolicy is the policy used to reconcile the external principal. |  | Optional: \{\} <br /> |


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

| Field | Description |
| --- | --- |
| `Delete` | DeletionPolicyDelete requests removal of the associated external resource or<br />authentication material.<br /> |
| `Retain` | DeletionPolicyRetain preserves the associated external resource or authentication<br />material after Kubernetes resource deletion.<br /> |


#### ManagedPrincipalSpec



ManagedPrincipalSpec defines an external principal whose lifecycle is managed by the
controller.



_Appears in:_
- [CloudPrincipalSpec](#cloudprincipalspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `name` _string_ | Name is the desired external principal name, subject to provider naming rules. It is<br />required; the API does not default it from metadata.name. For a GCP ServiceAccount, use<br />the short account ID rather than its email address. The resolved name cannot change<br />after provisioning. |  | MinLength: 1 <br /> |
| `deletionPolicy` _[DeletionPolicy](#deletionpolicy)_ | DeletionPolicy controls cleanup of the managed external principal when the<br />CloudPrincipal is deleted. Delete removes the external principal and is the default.<br />Retain leaves it in the cloud provider. | Delete | Enum: [Delete Retain] <br />Optional: \{\} <br /> |


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

| Field | Description |
| --- | --- |
| `ServiceAccount` | PrincipalKindServiceAccount identifies a cloud service account.<br /> |
| `Role` | PrincipalKindRole identifies a cloud IAM role; support depends on the provider.<br /> |
| `User` | PrincipalKindUser identifies a cloud user identity.<br /> |
| `Group` | PrincipalKindGroup identifies a cloud group identity.<br /> |
| `AllUsers` | PrincipalKindAllUsers represents public access for all users, including unauthenticated<br />users. It can only be referenced, not managed.<br /> |


#### PrincipalManagementPolicy

_Underlying type:_ _string_

PrincipalManagementPolicy selects whether the controller manages an external principal or
only resolves a reference to it.



_Appears in:_
- [CloudPrincipalSpec](#cloudprincipalspec)
- [CloudPrincipalStatus](#cloudprincipalstatus)

| Field | Description |
| --- | --- |
| `Managed` | PrincipalManagementPolicyManaged creates and reconciles the external principal according<br />to spec.managed.<br /> |
| `Reference` | PrincipalManagementPolicyReference resolves an existing identity without creating or<br />deleting it.<br /> |


#### PrincipalPolicySpec



PrincipalPolicySpec restricts managed and referenced cloud principals.



_Appears in:_
- [UsagePolicySpec](#usagepolicyspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `allowedNamePatterns` _[AllowedNamePatterns](#allowednamepatterns)_ | AllowedNamePatterns contains Go regular expressions for managed principal<br />names from spec.managed.name. During deletion, the recorded external name<br />is used when available. At least one pattern must match; an empty list<br />allows no names. Use ^ and $ to match the entire name. |  |  |
| `allowedReferencePatterns` _[AllowedNamePatterns](#allowednamepatterns)_ | AllowedReferencePatterns contains Go regular expressions for referenced<br />principal names from spec.reference.name. Defaults to [".*"], allowing<br />any reference name. An empty list allows no reference names.<br />Use ^ and $ to match the entire name. AllUsers has no reference name<br />and is exempt from this check. | [.*] |  |
| `allowManaged` _boolean_ | AllowManaged permits CloudPrincipals with managementPolicy Managed.<br />Defaults to true. Kind and name restrictions still apply. | true |  |
| `allowReferences` _boolean_ | AllowReferences permits CloudPrincipals with managementPolicy Reference,<br />including AllUsers. Defaults to true. Kind and applicable reference-name<br />restrictions still apply. | true |  |
| `allowedKinds` _[PrincipalKind](#principalkind) array_ | AllowedKinds lists permitted CloudPrincipal kinds for both management<br />policies. Defaults to ServiceAccount, Role, User, Group, and AllUsers.<br />An empty list allows no kinds. Provider capability restrictions still apply.<br />During deletion, the recorded status.kind is checked. | [ServiceAccount Role User Group AllUsers] |  |


#### PrincipalReference



PrincipalReference identifies a namespaced CloudPrincipal resource.



_Appears in:_
- [BucketAccessSpec](#bucketaccessspec)
- [CloudPrincipalAuthSpec](#cloudprincipalauthspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `name` _string_ | Name is the Kubernetes resource name of the CloudPrincipal, not its external cloud name<br />or identifier. |  | MinLength: 1 <br /> |
| `namespace` _string_ | Namespace is the Kubernetes namespace containing the CloudPrincipal. It must be<br />specified explicitly, including for references within the same namespace. |  | MinLength: 1 <br /> |


#### ProviderConfig



ProviderConfig is a cluster-scoped resource defining cloud connectivity and policies for
resources that use it.





| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `apiVersion` _string_ | `vedro.svetoch.dev/v1alpha1` | | |
| `kind` _string_ | `ProviderConfig` | | |
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.37/#objectmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |  |  |
| `spec` _[ProviderConfigSpec](#providerconfigspec)_ | Spec defines the desired state of the ProviderConfig. |  |  |
| `status` _[ProviderConfigStatus](#providerconfigstatus)_ | Status records the observed state of the ProviderConfig and is maintained by the<br />controller. |  | Optional: \{\} <br /> |


#### ProviderConfigReference



ProviderConfigReference selects a cluster-scoped provider configuration.



_Appears in:_
- [BucketSpec](#bucketspec)
- [CloudPrincipalSpec](#cloudprincipalspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `name` _string_ | Name is the name of the cluster-scoped ProviderConfig. No namespace is required. |  | MinLength: 1 <br /> |


#### ProviderConfigSpec



ProviderConfigSpec defines cloud scope, operator authentication, and resource usage
restrictions.



_Appears in:_
- [ProviderConfig](#providerconfig)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `type` _[ProviderType](#providertype)_ | Type selects the cloud implementation: gcp for Google Cloud or yc for Yandex Cloud. This<br />field is immutable. |  | Enum: [gcp yc] <br /> |
| `projectId` _string_ | ProjectId is the Google Cloud project ID for gcp or the Yandex Cloud folder ID for yc.<br />It determines where managed cloud resources are created and is immutable. |  | MinLength: 1 <br /> |
| `region` _string_ | Region is the provider region, such as europe-west1 or ru-central1. It must be a<br />non-empty regional name. Bucket placement is selected separately by Bucket.spec.location. |  |  |
| `method` _[AuthMethod](#authmethod)_ | Method selects how the operator authenticates to the provider. StaticCredentials reads a<br />service account key from credentialsSecretRef. WorkloadIdentity uses the operator's<br />ambient Google Cloud credentials or the Yandex Cloud instance service account. |  | Enum: [StaticCredentials WorkloadIdentity] <br /> |
| `credentialsSecretRef` _[SecretReference](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.37/#secretreference-v1-core)_ | CredentialsSecretRef identifies an existing Kubernetes Secret by name and namespace.<br />Required for StaticCredentials and forbidden for WorkloadIdentity. The Secret must<br />contain a key entry with a Google Cloud service account JSON key for gcp or a Yandex<br />Cloud authorized service account key in JSON format for yc. |  | Optional: \{\} <br /> |
| `usagePolicy` _[UsagePolicySpec](#usagepolicyspec)_ | UsagePolicy defines namespace and resource restrictions for this provider.<br />Resources must satisfy all applicable restrictions to reconcile. |  |  |


#### ProviderConfigStatus



ProviderConfigStatus defines the observed provider configuration state.



_Appears in:_
- [ProviderConfig](#providerconfig)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `conditions` _[Condition](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.37/#condition-v1-meta) array_ | Conditions represent the latest available observations of the ProviderConfig state. |  | Optional: \{\} <br /> |
| `observedGeneration` _integer_ | ObservedGeneration is the latest metadata.generation observed by the controller. |  | Optional: \{\} <br /> |


#### ProviderType

_Underlying type:_ _string_

ProviderType identifies a supported cloud implementation.



_Appears in:_
- [ProviderConfigSpec](#providerconfigspec)

| Field | Description |
| --- | --- |
| `gcp` | ProviderTypeGCP selects Google Cloud Storage and IAM.<br /> |
| `yc` | ProviderTypeYandexCloud selects Yandex Cloud Object Storage and IAM.<br /> |


#### ReferencedPrincipalSpec



ReferencedPrincipalSpec identifies an existing external principal without managing its
lifecycle.



_Appears in:_
- [CloudPrincipalSpec](#cloudprincipalspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `name` _string_ | Name identifies an existing external principal in the format expected by the provider<br />and kind. For a GCP ServiceAccount, use its email address. For a Yandex Cloud<br />ServiceAccount, use <folder-name>:<service-account-name>, such as prod:some-sa.<br />User and GCP Group references use email addresses. References are resolved without<br />creating or deleting the external principal. |  | MinLength: 1 <br /> |


#### SoftDeletePolicy



SoftDeletePolicy controls Google Cloud Storage retention of deleted objects.



_Appears in:_
- [BucketGcpConfig](#bucketgcpconfig)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `retentionDuration` _[Duration](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.37/#duration-v1-meta)_ | RetentionDuration is how long GCS retains soft-deleted objects before permanent<br />deletion. Use "0s" to disable soft delete, or a Go duration representing a whole number<br />of days from 7 through 90, such as "168h" for seven days. Defaults to "168h". | 168h |  |


#### StaticCredentialsSpec



StaticCredentialsSpec configures credentials stored in a Kubernetes Secret.



_Appears in:_
- [CloudPrincipalAuthSpec](#cloudprincipalauthspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `secretRef` _[AuthObjectReference](#authobjectreference)_ | SecretRef names the output Secret in the CloudPrincipalAuth namespace. The controller<br />stores a GCP service account JSON key under credentials.json, or Yandex Cloud S3<br />credentials under AWS_ACCESS_KEY_ID and AWS_SECRET_ACCESS_KEY. The Secret is created and<br />managed by the controller; it is not an input credential Secret. |  |  |
| `deletionPolicy` _[DeletionPolicy](#deletionpolicy)_ | DeletionPolicy controls cleanup when CloudPrincipalAuth is deleted. Delete revokes the<br />cloud credentials and allows the managed Secret to be deleted. Retain preserves both the<br />credentials and Secret, removing the Secret's owner references. Defaults to Retain. | Retain | Enum: [Delete Retain] <br />Optional: \{\} <br /> |


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
| `field` _string_ | Field is the path to the unsupported field in the requested spec, including a list index<br />when applicable. |  |  |
| `message` _string_ | Message explains why the selected provider cannot apply the requested feature. |  |  |
| `reason` _[UnsupportedFeatureReason](#unsupportedfeaturereason)_ | Reason is a machine-readable code identifying the unsupported capability. |  |  |


#### UnsupportedFeatureReason

_Underlying type:_ _string_

UnsupportedFeatureReason identifies a requested capability that the selected provider
does not support.



_Appears in:_
- [UnsupportedFeature](#unsupportedfeature)

| Field | Description |
| --- | --- |
| `BucketUnsupportedVersioning` | BucketUnsupportedVersioning indicates that the selected provider does not support object<br />versioning.<br /> |
| `BucketUnsupportedLabels` | BucketUnsupportedLabels indicates that the selected provider does not support bucket<br />labels or tags.<br /> |
| `BucketUnsupportedPublicAccessPrevention` | BucketUnsupportedPublicAccessPrevention indicates that the selected provider does not<br />support public access prevention.<br /> |
| `BucketUnsupportedStorageClass` | BucketUnsupportedStorageClass indicates that the selected provider does not support the<br />requested storage class.<br /> |
| `BucketUnsupportedLifecycleExpiration` | BucketUnsupportedLifecycleExpiration indicates that the selected provider does not<br />support age-based lifecycle expiration.<br /> |
| `BucketUnsupportedLifecycle` | BucketUnsupportedLifecycle indicates that the selected provider does not support object<br />lifecycle rules.<br /> |
| `BucketUnsupportedLifecycleNamed` | BucketUnsupportedLifecycleNamed indicates that the selected provider does not support<br />lifecycle rule names.<br /> |
| `BucketUnsupportedLifecycleEnabled` | BucketUnsupportedLifecycleEnabled indicates that the selected provider does not support<br />lifecycle rule enablement.<br /> |
| `BucketAccessUnsupportedObjectReader` | BucketAccessUnsupportedObjectReader indicates that the selected provider does not<br />support ObjectReader grants.<br /> |
| `BucketAccessUnsupportedObjectWriter` | BucketAccessUnsupportedObjectWriter indicates that the selected provider does not<br />support ObjectWriter grants.<br /> |
| `BucketAccessUnsupportedObjectAdmin` | BucketAccessUnsupportedObjectAdmin indicates that the selected provider does not support<br />ObjectAdmin grants.<br /> |
| `BucketAccessUnsupportedBucketAdmin` | BucketAccessUnsupportedBucketAdmin indicates that the selected provider does not support<br />BucketAdmin grants.<br /> |
| `PrincipalAuthUnsupportedStaticCredentials` | PrincipalAuthUnsupportedStaticCredentials indicates that the provider does not support static credentials.<br /> |
| `PrincipalAuthUnsupportedWorkloadIdentity` | PrincipalAuthUnsupportedWorkloadIdentity indicates that the provider does not support workload identity.<br /> |
| `PrincipalAuthUnsupportedWorkloadIdentityKind` | PrincipalAuthUnsupportedWorkloadIdentityKind indicates that workload identity is unavailable for this principal kind.<br /> |
| `PrincipalAuthUnsupportedStaticCredentialsKind` | PrincipalAuthUnsupportedStaticCredentialsKind indicates that static credentials are unavailable for this principal kind.<br /> |
| `PrincipalUnsupportedManagedSA` | PrincipalUnsupportedManagedSA indicates that the selected provider does not support<br />management of service accounts.<br /> |
| `PrincipalUnsupportedManagedUser` | PrincipalUnsupportedManagedUser indicates that the selected provider does not support<br />management of users.<br /> |
| `PrincipalUnsupportedManagedRole` | PrincipalUnsupportedManagedRole indicates that the selected provider does not support<br />management of roles.<br /> |
| `PrincipalUnsupportedManagedAllUsers` | PrincipalUnsupportedManagedAllUsers indicates that the selected provider does not<br />support management of the AllUsers identity.<br /> |
| `PrincipalUnsupportedManagedGroup` | PrincipalUnsupportedManagedGroup indicates that the selected provider does not support<br />management of groups.<br /> |
| `PrincipalUnsupportedReferencedSA` | PrincipalUnsupportedReferencedSA indicates that the selected provider does not support<br />references to service accounts.<br /> |
| `PrincipalUnsupportedReferencedUser` | PrincipalUnsupportedReferencedUser indicates that the selected provider does not support<br />references to users.<br /> |
| `PrincipalUnsupportedReferencedRole` | PrincipalUnsupportedReferencedRole indicates that the selected provider does not support<br />references to roles.<br /> |
| `PrincipalUnsupportedReferencedGroup` | PrincipalUnsupportedReferencedGroup indicates that the selected provider does not<br />support references to groups.<br /> |
| `PrincipalUnsupportedReferencedAllUsers` | PrincipalUnsupportedReferencedAllUsers indicates that the selected provider does not<br />support references to the AllUsers identity.<br /> |


#### UsagePolicySpec



UsagePolicySpec controls which resources may use a ProviderConfig.



_Appears in:_
- [ProviderConfigSpec](#providerconfigspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `allowedNamespaces` _[AllowedNamespacesSpec](#allowednamespacesspec)_ | AllowedNamespaces restricts the namespaces of Bucket, CloudPrincipal,<br />BucketAccess, and CloudPrincipalAuth resources using this provider. |  |  |
| `bucketPolicy` _[BucketPolicySpec](#bucketpolicyspec)_ | BucketPolicy restricts cloud bucket names managed through this provider. |  |  |
| `principalPolicy` _[PrincipalPolicySpec](#principalpolicyspec)_ | PrincipalPolicy restricts principal kinds, management policies, and names<br />used through this provider. |  |  |


#### WorkloadIdentitySpec



WorkloadIdentitySpec configures authentication through a Kubernetes ServiceAccount.



_Appears in:_
- [CloudPrincipalAuthSpec](#cloudprincipalauthspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `serviceAccountRef` _[AuthObjectReference](#authobjectreference)_ | ServiceAccountRef names an existing Kubernetes ServiceAccount in the CloudPrincipalAuth<br />namespace. For GCP, the controller grants workload identity access to the cloud service<br />account and configures the Kubernetes ServiceAccount annotation. Workload identity for<br />principals is currently unsupported by the Yandex Cloud provider. |  |  |
| `deletionPolicy` _[DeletionPolicy](#deletionpolicy)_ | DeletionPolicy controls cleanup when CloudPrincipalAuth is deleted. Delete removes the<br />cloud authentication binding and managed ServiceAccount annotation. Retain preserves the<br />configuration. The Kubernetes ServiceAccount itself is not deleted. Defaults to Retain. | Retain | Enum: [Delete Retain] <br />Optional: \{\} <br /> |
