{{- define "core.podDisruptionBudget" -}}
{{- $ := index . 0 }}
{{- $labels := index . 1 }}
{{- $obj := include "core.obj.enricher" (list $ $labels (index . 2)) | fromYaml }}
{{- if $obj.enabled }}
---
apiVersion: policy/v1
kind: PodDisruptionBudget
metadata:
  name: {{ tpl $obj.name $ }}
  namespace: "{{ $obj.namespace }}"
  labels:
{{- include "core.labels.constructor" (list $ $labels $obj) | nindent 4 }}
spec:
  {{- if hasKey $obj "minAvailable" }}
  minAvailable: {{ $obj.minAvailable }}
  {{- else if hasKey $obj "maxUnavailable" }}
  maxUnavailable: {{ $obj.maxUnavailable }}
  {{- end }}
  selector:
    matchLabels:
      {{- tpl (toYaml $obj.selectorLabels) $ | nindent 6 }}
{{- end }}
{{- end -}}
