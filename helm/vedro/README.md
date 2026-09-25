# Vedro resources chart

This chart creates `ProviderConfig`, `CloudPrincipal`, `CloudPrincipalAuth`,
`Bucket`, and `BucketAccess` resources. Install the Vedro CRDs and controller
before installing this chart.

```sh
helm install example helm/vedro --namespace app --create-namespace -f values.yaml
```

The top-level `providers`, `principals`, and `buckets` values are maps. The map
key becomes the Kubernetes resource name unless `name` is set. A principal or
bucket can refer to a provider by map key or by its resulting `ProviderConfig`
name. When exactly one provider is defined, `provider` may be omitted.

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
    prefixReleaseName: true
    deletionPolicy: Retain
    versioning:
      enabled: true
    readers:
      - name: app
```

`usagePolicy` strings are evaluated as Helm templates, so release name and
namespace expressions work there. `Bucket.spec.location` defaults to the
selected provider's `region`; set `location` on a bucket to override it.
`nameOverride` sets the external bucket name in `Bucket.spec.name`.
`prefixReleaseName` prefixes the Kubernetes Bucket name; when `nameOverride`
is absent, that is also the external bucket name.

Principal `kind` defaults to `ServiceAccount`, `type` defaults to `Managed`,
and `managed.name` defaults to the Kubernetes principal name. Use
`type: Referenced` with `reference.name` for an existing cloud principal.
Authentication is only created when `auth` is set; it references a Secret or
ServiceAccount in the principal's namespace. The chart does not create those
Secret or ServiceAccount objects.

Bucket `admins`, `objAdmins`, `readers`, and `writers` lists create
`BucketAccess` resources with `BucketAdmin`, `ObjectAdmin`, `ObjectReader`,
and `ObjectWriter` levels. Each entry needs a principal `name`; `namespace`
defaults to the Helm release namespace. Buckets and principals also default
to the release namespace unless their own `namespace` is set.

The chart sets deletion policies to `Delete` for managed principals and
principal authentication, and `Retain` for buckets. Bucket fields
`storageClass`, `publicAccessPrevention`, `versioning`, `lifecycle`, `labels`,
and `cloudSpecificConfig` map directly to `Bucket.spec`. Set a provider's
`credentialsSecretRef` when using `method: StaticCredentials`.
