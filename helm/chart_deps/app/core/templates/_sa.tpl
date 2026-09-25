{{- define "core.serviceAccount" -}}
{{- $ := index . 0 }}
{{- $labels := index . 1 }}
{{- $obj := include "core.obj.enricher" (list $ $labels (index . 2)) | fromYaml }}
{{- if $obj.create }}
---
apiVersion: v1
kind: ServiceAccount
{{- if hasKey $obj "automountServiceAccountToken" }}
automountServiceAccountToken: {{ $obj.automountServiceAccountToken }}
{{- end }}
metadata:
  name: {{ tpl $obj.name $ }}
  labels:
{{- include "core.labels.constructor" (list $ $labels $obj) | nindent 4 }}
  namespace: "{{ $obj.namespace }}"
  {{- with $obj.annotations }}
  annotations:
    {{- tpl (toYaml .) $ | nindent 4 }}
  {{- end }}
{{- end }}
{{- end }}
