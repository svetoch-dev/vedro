{{- define "gvList" -}}
# API Reference

> Generated with `crd-ref-docs`. Run `make generate` to update this reference.

## Packages

{{ range . -}}
- {{ markdownRenderGVLink . }}
{{ end }}
{{ range . -}}
{{ template "gvDetails" . }}
{{ end -}}
{{- end -}}
