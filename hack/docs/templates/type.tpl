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
| `{{ .Name }}` _{{ markdownRenderType .Type }}_ | {{ template "type_members" . }} | {{ markdownRenderDefault .Default }} | {{ markdownRenderFieldDoc (join "; " .Validation) }} |
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
