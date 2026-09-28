{{- define "prometheus.labels.constructor" -}}
{{- $ := index . 0 }}
{{- $labels := index . 1 }}
{{- $obj := index . 2 }}
{{- if $obj.labels }}
{{- $labels = merge $labels $obj.labels }}
{{- end }}
{{- if $.Values.labels }}
{{- $labels = merge $labels $.Values.labels }}
{{- end }}
{{- tpl (toYaml $labels ) $ }}
{{- end }}

{{- define "prometheus.obj.enricher" -}}
{{- $ := index . 0 }}
{{- $labels := index . 1 }}
{{- $obj := index . 2 }}
{{- if eq (hasKey $obj "namespace") false }}
{{- $obj = set $obj "namespace" $.Release.Namespace }}
{{- end }}
{{- if eq (hasKey $obj "enabled") false }}
{{- $obj = set $obj "enabled" true }}
{{- end }}
{{- toYaml $obj }} 
{{- end }}
