{{- define "prometheus.podMonitor" -}}
{{- $ := index . 0 }}
{{- $labels := index . 1 }}
{{- $obj := include "prometheus.obj.enricher" (list $ $labels (index . 2)) | fromYaml }}
{{- if $obj.enabled }}
---
apiVersion: monitoring.coreos.com/v1
kind: PodMonitor
metadata:
  labels:
  {{- if $obj.prometheusSelector }}
  {{ tpl (toYaml $obj.prometheusSelector) . | nindent 4}}
  {{- end }}
  {{- include "prometheus.labels.constructor" (list $ $labels $obj) | nindent 4 }}
  name: {{ tpl $obj.name $ }}
  {{- if $obj.namespace }}
  namespace: {{ $obj.namespace }}
  {{- end }}
spec:
  {{- with $obj.podMetricsEndpoints }}
  podMetricsEndpoints:
  {{- tpl (toYaml .) $ | nindent 4 }}
  {{- end }}
  {{- with $obj.jobLabel }}
  jobLabel: {{ tpl . $ }}
  {{- end }}
  namespaceSelector:
  {{- if eq $obj.namespaceSelector "any" }}
    any: true
  {{- else }}
    matchNames:
    - {{ $obj.namespace }}
  {{- end }}
  {{- with $obj.selectorLabels }}
  selector:
    matchLabels:
    {{- tpl (toYaml .) $ | nindent 6 }}
  {{- end }}
  {{- with $obj.podTargetLabels }}
  podTargetLabels:
  {{- tpl (toYaml .) $ | nindent 4 }}
  {{- end }}
{{- end }}
{{- end }}
