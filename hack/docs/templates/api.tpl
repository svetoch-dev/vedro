{{- define "cellDoc" -}}
{{- range $index, $paragraph := splitList "\n\n" (trim .) -}}
{{- if $index }}<br /><br />{{ end -}}
{{- markdownRenderFieldDoc (regexReplaceAll "\\s*\n\\s*" $paragraph " ") -}}
{{- end -}}
{{- end -}}

{{- define "gvList" -}}
# API Reference

> Generated with `crd-ref-docs`. Run `make generate` to update this reference.

## Packages

{{ range . -}}
- {{ markdownRenderGVLink . }}
{{ end }}
{{ range . -}}
{{ $gv := . -}}
## {{ .GroupVersionString }}

{{ trim .Doc }}

### Resource Types

{{ range .SortedKinds -}}
- {{ $gv.TypeForKind . | markdownRenderTypeLink }}
{{ end }}
{{ range .SortedTypes -}}
{{ template "type" . }}
{{ end -}}
{{ end -}}
{{- end -}}

{{- define "type" -}}
{{ if markdownShouldRenderType . -}}
#### {{ .Name }}
{{ if .IsAlias }}
_Underlying type:_ _{{ markdownRenderTypeLink .UnderlyingType }}_
{{ end }}
{{ trim .Doc }}
{{ if .Validation }}
_Validation:_

{{ range .Validation -}}
- {{ . }}
{{ end -}}
{{ end -}}
{{ if .References }}
_Appears in:_

{{ range .SortedReferences -}}
- {{ markdownRenderTypeLink . }}
{{ end -}}
{{ end -}}
{{ if .Members }}
| Field | Description | Default | Validation |
| --- | --- | --- | --- |
{{ if .GVK -}}
| `apiVersion` _string_ | `{{ .GVK.Group }}/{{ .GVK.Version }}` | | |
| `kind` _string_ | `{{ .GVK.Kind }}` | | |
{{ end -}}
{{ range .Members -}}
| `{{ .Name }}` _{{ markdownRenderType .Type }}_ | {{ if eq .Name "metadata" }}Refer to Kubernetes API documentation for fields of `metadata`.{{ else }}{{ template "cellDoc" .Doc }}{{ end }} | {{ markdownRenderDefault .Default }} | {{ markdownRenderFieldDoc (join "; " .Validation) }} |
{{ end -}}
{{ end -}}
{{ if .EnumValues }}
| Value | Description |
| --- | --- |
{{ range .EnumValues -}}
| `{{ .Name }}` | {{ template "cellDoc" .Doc }} |
{{ end -}}
{{ end -}}
{{ end -}}
{{- end -}}
