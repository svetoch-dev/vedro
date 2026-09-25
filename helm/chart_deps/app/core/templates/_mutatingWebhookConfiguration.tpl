{{- define "core.mutatingWebhookConfiguration" -}}
{{- $ := index . 0 }}
{{- $labels := index . 1 }}
{{- $obj := include "core.obj.enricher" (list $ $labels (index . 2)) | fromYaml }}
{{- if $obj.enabled }}
---
apiVersion: admissionregistration.k8s.io/v1
kind: MutatingWebhookConfiguration
metadata:
  name: {{ tpl $obj.name $ }}
  labels:
{{- include "core.labels.constructor" (list $ $labels $obj) | nindent 4 }}
  {{- if hasKey $obj "annotations" }}
  annotations:
  {{- tpl (toYaml $obj.annotations) $ | nindent 4 }}
  {{- end }}
webhooks:
{{- tpl (toYaml $obj.webhooks) $ | nindent 2 }}
{{- end }}
{{- end }}
