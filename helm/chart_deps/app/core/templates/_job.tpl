{{- define "core.job" -}}
{{- $ := index . 0 }}
{{- $labels := index . 1 }}
{{- $obj := include "core.obj.enricher" (list $ $labels (index . 2)) | fromYaml }}
{{- if $obj.enabled }}
---
apiVersion: batch/v1
kind: Job
metadata:
  labels:
{{- include "core.labels.constructor" (list $ $labels $obj) | nindent 4 }}
  name: {{ tpl $obj.name $ }}
  namespace: "{{ $obj.namespace }}"
  {{- with $obj.annotations }}
  annotations:
    {{- tpl (toYaml .) $ | nindent 4 }}
  {{- end }}
spec:
  {{ include "core.podtemplate" (list $ $obj) | nindent 2 | trim }}
{{- end }}
{{- end }}
