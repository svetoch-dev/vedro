{{- define "core.service" -}}
{{- $ := index . 0 }}
{{- $labels := index . 1 }}
{{- $obj := include "core.obj.enricher" (list $ $labels (index . 2)) | fromYaml }}
{{- if $obj.enabled }}
---
apiVersion: v1
kind: Service
metadata:
  name: {{ tpl $obj.name $ }}
  labels:
{{- include "core.labels.constructor" (list $ $labels $obj) | nindent 4 }}
  namespace: "{{ $obj.namespace }}"
  {{- if hasKey $obj "annotations" }}
  annotations:
  {{- tpl (toYaml $obj.annotations) $ | nindent 4 }}
  {{- end }}
spec:
  type: {{ $obj.type }}
  {{- with $obj.ports }}
  ports:
  {{- tpl (toYaml .) $ | nindent 4 }}
  {{- end }}
  {{- if $obj.externalTrafficPolicy }}
  externalTrafficPolicy: {{ $obj.externalTrafficPolicy }}
  {{- end }}
  selector:
    {{- tpl (toYaml $obj.selectorLabels) $ | nindent 4 }} 
{{- end }}
{{- end }}
