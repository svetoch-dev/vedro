{{- define "core.labels.constructor" -}}
{{- $ := index . 0 }}
{{- $labels := deepCopy (index . 1) }}
{{- $obj := index . 2 }}
{{- if $obj.labels }}
{{- $labels = merge $labels $obj.labels }}
{{- end }}
{{- if $.Values.labels }}
{{- $labels = merge $labels $.Values.labels }}
{{- end }}
{{- tpl (toYaml $labels ) $ }}
{{- end }}

{{- define "core.obj.enricher" -}}
{{- $ := index . 0 }}
{{- $labels := index . 1 }}
{{- $obj := index . 2 }}
{{- if eq (hasKey $obj "namespace") false }}
{{- $namespace := $.Values.namespaceOverride | default $.Release.Namespace }}
{{- $obj = set $obj "namespace" $namespace }}
{{- end }}
{{- if eq (hasKey $obj "enabled") false }}
{{- $obj = set $obj "enabled" true }}
{{- end }}
{{- if and (hasKey $obj "annotations") (or (eq (toYaml $obj.annotations) "null") (eq (toYaml $obj.annotations) "null\n")) }}
{{- $_ := unset $obj "annotations" }}
{{- end }}
{{- toYaml $obj }} 
{{- end }}
