{{- define "core.deployment" -}}
{{- $ := index . 0 }}
{{- $labels := index . 1 }}
{{- $obj := include "core.obj.enricher" (list $ $labels (index . 2)) | fromYaml }}
{{- if $obj.enabled }}
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: {{ tpl $obj.name $ }}
  {{- with $obj.annotations }}
  annotations:
  {{- tpl (toYaml .) $ | nindent 4 }}
  {{- end }}
  labels:
{{- include "core.labels.constructor" (list $ $labels $obj) | nindent 4 }}
  namespace: "{{ $obj.namespace }}"
spec:
  {{- if not $obj.autoscaling.enabled }}
  replicas: {{ $obj.replicaCount }}
  {{- end }}
  {{- if ne $obj.revisionHistoryLimit nil }}
  revisionHistoryLimit: {{ $obj.revisionHistoryLimit }}
  {{- end }}
  {{- with $obj.selectorLabels }}
  selector:
    matchLabels:
      {{- tpl (toYaml .) $ | nindent 8 }}
  {{- end }}
  {{- with $obj.strategy }}
  strategy:
    {{- tpl (toYaml .) $ | nindent 4 }}
  {{- end }}
  {{ include "core.podtemplate" (list $ $obj) | nindent 2 | trim }}
{{- end }}
{{- end }}
