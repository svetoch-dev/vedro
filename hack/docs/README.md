# API reference generation

Run `make generate` to regenerate DeepCopy code and `docs/reference/api.md`, or
`make generate-api-reference` to update only the reference. The generator is
installed into `bin/crd-ref-docs` when missing and reused on later runs.

The [crd-ref-docs](https://github.com/elastic/crd-ref-docs) version is pinned by
`CRD_REF_DOCS_VERSION` in the Makefile and installed using the same local tool
helper as controller-gen. It does not add dependencies to the operator's Go module.

The reference uses comments and JSON field names from `api/`. Update those Go
types to change API descriptions. `config.yaml` hides list wrappers and embedded
TypeMeta fields while retaining resource status fields. Keep its Kubernetes
version aligned with the k8s.io dependencies so external API links target the
matching Kubernetes reference.

The generator's embedded Markdown templates render resource lists, field tables,
defaults, validation rules, and enum descriptions. No local templates are needed.
