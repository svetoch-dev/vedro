{{- define "gvDetails" -}}
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
{{- end -}}
