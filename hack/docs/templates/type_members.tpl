{{- define "cellDoc" -}}
{{- range $index, $paragraph := splitList "\n\n" (trim .) -}}
{{- if $index }}<br /><br />{{ end -}}
{{- markdownRenderFieldDoc (regexReplaceAll "\\s*\n\\s*" $paragraph " ") -}}
{{- end -}}
{{- end -}}

{{- define "type_members" -}}
{{ if eq .Name "metadata" }}Refer to Kubernetes API documentation for fields of `metadata`.{{ else }}{{ template "cellDoc" .Doc }}{{ end }}
{{- end -}}
