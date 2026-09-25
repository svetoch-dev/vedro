{{- define "core.horizontalPodAutoscaler" -}}
{{- $ := index . 0 }}
{{- $labels := index . 1 }}
{{- $obj := include "core.obj.enricher" (list $ $labels (index . 2)) | fromYaml }}
{{- if $obj.enabled }}
---
apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
metadata:
  name: {{ tpl $obj.name $ }}
  namespace: "{{ $obj.namespace }}"
  labels:
{{- include "core.labels.constructor" (list $ $labels $obj) | nindent 4 }}
spec:
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: {{ tpl $obj.targetName $ }}
  minReplicas: {{ $obj.minReplicas }}
  maxReplicas: {{ $obj.maxReplicas }}
  {{- with $obj.metrics }}
  metrics:
  {{- tpl (toYaml .) $ | nindent 4 }}
  {{- end }}
  {{- with $obj.behavior }}
  behavior:
  {{- tpl (toYaml .) $ | nindent 4 }}
  {{- end }}
{{- end }}
{{- end -}}
