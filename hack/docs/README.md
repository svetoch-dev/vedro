# API reference generation

Run `make generate` to regenerate DeepCopy code and `docs/reference/api.md`, or
`make generate-api-reference` to update only the reference. The generator is
installed into `bin/gen-crd-api-reference-docs` when missing and reused on later
runs. Changes to this tools module trigger reinstallation.

The generator version is pinned in this directory's `go.mod`, separately from the
operator's dependencies. Its `golang.org/x/tools` dependency is updated to support
the project's Go toolchain. After changing tool dependencies, run `go mod tidy`
from this directory and regenerate the reference.

The reference uses comments and JSON field names from `api/`. Update those Go
types to change API descriptions. `config.json` hides list wrappers and links
Kubernetes metadata types to their upstream documentation. The local templates
produce HTML tables and anchors inside Markdown, in the style of Prometheus
Operator's API reference.
