# Vedro resources chart

This chart creates `ProviderConfig`, `CloudPrincipal`, `CloudPrincipalAuth`,
`Bucket`, and `BucketAccess` resources. Install the Vedro CRDs and controller
before installing this chart.

```sh
helm install example helm/vedro --namespace app --create-namespace -f values.yaml
```

The top-level `providers`, `principals`, and `buckets` values are maps. The map
key becomes the Kubernetes resource name unless `name` is set. Every principal
and bucket must set `provider` to the exact name of a `ProviderConfig`, whether
that resource is created by this chart or already exists in the cluster.

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

`usagePolicy` strings are evaluated as Helm templates, so release name and
namespace expressions work there. Every bucket must set `location` explicitly;
the provider's `region` is not used as a fallback.
`nameOverride` sets the external bucket name in `Bucket.spec.name`.
`prefixReleaseName` prefixes the Kubernetes Bucket name; when `nameOverride`
is absent, that is also the external bucket name.

Every principal must set `kind` and `type`. The only accepted `type` values are
`Managed` and `Reference`. `managed.name` defaults to the Kubernetes principal
name. Use `type: Reference` with `reference.name` for an existing cloud principal.
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
