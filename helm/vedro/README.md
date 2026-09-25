# Vedro Resources Chart

Creates provider configurations, cloud principals, principal authentication,
buckets, and bucket access grants for the Vedro operator.

## Usage

```yaml
providers:
  primary:
    type: gcp
    projectId: my-project
    region: europe-west1
    method: WorkloadIdentity
    usagePolicy:
      allowedNamespaces:
        names: ["{{ .Release.Namespace }}"]
      bucketPolicy:
        allowedNamePatterns: ["^{{ .Release.Name }}-.*$"]
      principalPolicy:
        allowedNamePatterns: ["^{{ .Release.Name }}-.*$"]
        allowedReferencePatterns: ["^.*$"]

principals:
  app:
    provider: primary
    kind: ServiceAccount
    type: Managed
    managed:
      name: example-app
    auth:
      method: WorkloadIdentity
      workloadIdentity:
        serviceAccountRef:
          name: app

buckets:
  data:
    provider: primary
    location: europe-west1
    prefixReleaseName: true
    deletionPolicy: Retain
    versioning:
      enabled: true
    readers:
      - name: app
```

Save the example as `values.yaml`, then install it with:

```sh
helm install example helm/vedro --namespace app --create-namespace -f values.yaml
```

## Requirements

| Name | Version |
|------|---------|
| Helm | 3.x |
| Vedro CRDs | `vedro.svetoch.dev/v1alpha1` |
| Vedro controller | Compatible with the installed CRDs |

Install the Vedro CRDs and controller before installing this chart.

## Inputs

| Name | Description | Type | Default | Required |
|------|-------------|------|---------|:--------:|
| `providers` | ProviderConfig resources, keyed by resource name. | `map(object)` | `{}` | no |
| `principals` | CloudPrincipal resources and optional CloudPrincipalAuth resources, keyed by resource name. | `map(object)` | `{}` | no |
| `buckets` | Bucket resources and their BucketAccess grants, keyed by resource name. | `map(object)` | `{}` | no |

## Notes

- Each map key becomes the Kubernetes resource name unless the entry sets `name`.
- Every principal and bucket must set `provider` to the exact name of a `ProviderConfig`. The provider may be created by this chart or already exist in the cluster.
- Every bucket must set `location`. The provider's `region` is not used as a fallback.
- Strings inside `providers[*].usagePolicy` are evaluated as Helm templates, so they may use `.Release.Name` and `.Release.Namespace`.
- Principals and buckets default to the Helm release namespace. ProviderConfig is cluster scoped.
- All generated resources receive the chart's common metadata labels.
- Principal authentication references a Secret or ServiceAccount in the principal's namespace; this chart does not create those objects.
- Bucket access entries default their principal namespace to the Helm release namespace. Their `BucketAccess` resources are created in the bucket's namespace.

## Type Details

### `providers{}`

| Field | Type | Required | Description |
|-------|------|:--------:|-------------|
| `name` | `string` | no | ProviderConfig name; defaults to the map key. |
| `type` | `string` | yes | Provider type: `gcp` or `yc`. |
| `projectId` | `string` | yes | Cloud project, folder, or account identifier. |
| `region` | `string` | yes | Provider region. |
| `method` | `string` | yes | `WorkloadIdentity` or `StaticCredentials`. |
| `credentialsSecretRef` | `object` | for `StaticCredentials` | Existing Secret reference; omitted from the CR for `WorkloadIdentity`. |
| `usagePolicy` | `object` | yes | Namespace, bucket, and principal restrictions. |

### `providers{}.credentialsSecretRef`

| Field | Type | Required | Description |
|-------|------|:--------:|-------------|
| `name` | `string` | yes | Secret name. |
| `namespace` | `string` | yes | Secret namespace. |

### `providers{}.usagePolicy`

| Field | Type | Required | Description |
|-------|------|:--------:|-------------|
| `allowedNamespaces` | `object` | yes | Set `names` to allowed namespace names or `all: true` to allow every namespace. |
| `bucketPolicy.allowedNamePatterns` | `list(string)` | yes | Regular expressions allowed for cloud bucket names. |
| `principalPolicy.allowedNamePatterns` | `list(string)` | yes | Regular expressions allowed for managed principal names. |
| `principalPolicy.allowedReferencePatterns` | `list(string)` | no | Regular expressions allowed for referenced principal names. |

### `providers{}.usagePolicy.allowedNamespaces`

| Field | Type | Required | Description |
|-------|------|:--------:|-------------|
| `names` | `list(string)` | no | Exact namespace names allowed to use the provider. |
| `all` | `bool` | no | Allow every namespace; cannot be combined with nonempty `names`. |

### `providers{}.usagePolicy.principalPolicy`

| Field | Type | Required | Description |
|-------|------|:--------:|-------------|
| `allowedNamePatterns` | `list(string)` | yes | Patterns for managed principal names. |
| `allowedReferencePatterns` | `list(string)` | no | Patterns for referenced principal names; the CRD defaults to `[".*"]`. |
| `allowManaged` | `bool` | no | Permit managed principals; the CRD defaults to `true`. |
| `allowReferences` | `bool` | no | Permit referenced principals; the CRD defaults to `true`. |
| `allowedKinds` | `list(string)` | no | Allowed principal kinds; the CRD defaults to all supported kinds. |

### `principals{}`

| Field | Type | Required | Description |
|-------|------|:--------:|-------------|
| `name` | `string` | no | CloudPrincipal name; defaults to the map key. |
| `namespace` | `string` | no | Kubernetes namespace; defaults to the release namespace. |
| `provider` | `string` | yes | Exact ProviderConfig name. |
| `kind` | `string` | yes | `ServiceAccount`, `User`, `Group`, `Role`, or `AllUsers`. |
| `type` | `string` | yes | `Managed` or `Reference`. |
| `managed` | `object` | for `Managed` | External principal name and deletion policy. |
| `reference` | `object` | for `Reference` except `AllUsers` | Existing external principal name. |
| `auth` | `object` | no | Creates CloudPrincipalAuth; valid only for `Managed` principals. |

### `principals{}.managed`

| Field | Type | Required | Description |
|-------|------|:--------:|-------------|
| `name` | `string` | no | External principal name; defaults to the CloudPrincipal name. |
| `deletionPolicy` | `string` | no | `Delete` or `Retain`; defaults to `Delete`. |

### `principals{}.reference`

| Field | Type | Required | Description |
|-------|------|:--------:|-------------|
| `name` | `string` | yes | Existing external principal name. |

### `principals{}.auth`

| Field | Type | Required | Description |
|-------|------|:--------:|-------------|
| `method` | `string` | yes | `WorkloadIdentity` or `StaticCredentials`. |
| `workloadIdentity.serviceAccountRef.name` | `string` | for `WorkloadIdentity` | Existing ServiceAccount name in the principal's namespace. |
| `workloadIdentity.deletionPolicy` | `string` | no | `Delete` or `Retain`; defaults to `Delete`. |
| `staticCredentials.secretRef.name` | `string` | for `StaticCredentials` | Secret name in the principal's namespace. |
| `staticCredentials.deletionPolicy` | `string` | no | `Delete` or `Retain`; defaults to `Delete`. |

### `buckets{}`

| Field | Type | Required | Description |
|-------|------|:--------:|-------------|
| `name` | `string` | no | Bucket CR name; defaults to the map key. |
| `namespace` | `string` | no | Kubernetes namespace; defaults to the release namespace. |
| `provider` | `string` | yes | Exact ProviderConfig name. |
| `location` | `string` | yes | Cloud bucket location. |
| `prefixReleaseName` | `bool` | no | Prefix the Bucket CR name with the release name; defaults to `false`. |
| `nameOverride` | `string` | no | External bucket name (`spec.name`); otherwise the Bucket CR name is used. |
| `deletionPolicy` | `string` | no | `Delete` or `Retain`; defaults to `Retain`. |
| `storageClass` | `string` | no | `Standard`, `Warm`, `Cold`, or `Ice`; the CRD defaults to `Standard`. |
| `publicAccessPrevention` | `bool` | no | Whether public access is prevented. |
| `versioning` | `object` | no | Bucket versioning settings, including `enabled`. |
| `lifecycle` | `object` | no | Bucket lifecycle rules. |
| `labels` | `map(string)` | no | Cloud provider labels on the bucket, separate from CR metadata labels. |
| `cloudSpecificConfig` | `object` | no | Provider-specific bucket settings. |
| `admins` | `list(object)` | no | Principals granted `BucketAdmin`. |
| `objAdmins` | `list(object)` | no | Principals granted `ObjectAdmin`. |
| `readers` | `list(object)` | no | Principals granted `ObjectReader`. |
| `writers` | `list(object)` | no | Principals granted `ObjectWriter`. |

### `buckets{}.versioning`

| Field | Type | Required | Description |
|-------|------|:--------:|-------------|
| `enabled` | `bool` | yes | Enable or disable bucket object versioning. |

### `buckets{}.lifecycle.rules[]`

| Field | Type | Required | Description |
|-------|------|:--------:|-------------|
| `name` | `string` | no | Stable identifier for the rule. |
| `enabled` | `bool` | no | Whether the rule is active; defaults to `true`. |
| `ageDays` | `integer` | no | Match objects older than this number of days. |
| `action` | `string` | yes | Lifecycle action; currently `Delete`. |

### `buckets{}.cloudSpecificConfig.gcp.softDeletePolicy`

| Field | Type | Required | Description |
|-------|------|:--------:|-------------|
| `retentionDuration` | `string` | no | `0s` or a whole number of days from `168h` to `2160h`; the CRD defaults to `168h`. |

### `buckets{}.{admins,objAdmins,readers,writers}[]`

| Field | Type | Required | Description |
|-------|------|:--------:|-------------|
| `name` | `string` | yes | CloudPrincipal CR name. |
| `namespace` | `string` | no | CloudPrincipal namespace; defaults to the release namespace. |

## Outputs

The chart renders Kubernetes resources rather than Helm output values.

| Resource | Created from |
|----------|--------------|
| `ProviderConfig` | Each `providers` entry. |
| `CloudPrincipal` | Each `principals` entry. |
| `CloudPrincipalAuth` | A managed principal with `auth` set. |
| `Bucket` | Each `buckets` entry. |
| `BucketAccess` | Each entry in a bucket's `admins`, `objAdmins`, `readers`, or `writers` list. |
