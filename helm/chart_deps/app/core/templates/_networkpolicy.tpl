{{- define "core.networkPolicy" -}}
{{- $ := index . 0 }}
{{- $labels := index . 1 }}
{{- $obj := include "core.obj.enricher" (list $ $labels (index . 2)) | fromYaml }}
{{- if $obj.enabled }}
---
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: {{ tpl $obj.name $ }}
  namespace: "{{ $obj.namespace }}"
  labels:
{{- include "core.labels.constructor" (list $ $labels $obj) | nindent 4 }}
spec:
  podSelector:
    matchLabels:
      {{- tpl (toYaml $obj.selectorLabels) $ | nindent 6 }}
  {{- with $obj.policyTypes }}
  policyTypes:
    {{- tpl (toYaml .) $ | nindent 4 }}
  {{- end }}
  {{- with $obj.ingress }}
  ingress:
    {{- tpl (toYaml .) $ | nindent 4 }}
  {{- end }}
  {{- with $obj.egress }}
  egress:
    {{- tpl (toYaml .) $ | nindent 4 }}
  {{- end }}
{{- end }}
{{- end -}}
