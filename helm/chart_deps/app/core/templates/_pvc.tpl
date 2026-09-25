{{- define "core.pvc" -}}
{{- $ := index . 0 }}
{{- $labels := index . 1 }}
{{- $obj := include "core.obj.enricher" (list $ $labels (index . 2)) | fromYaml }}
{{- if $obj.enabled }}
{{- if not $obj.volumeClaimTampleate }}
---
{{- end }}
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  labels:
{{- include "core.labels.constructor" (list $ $labels $obj) | nindent 4 }}
  name: {{ tpl $obj.name $ }}
  namespace: "{{ $obj.namespace }}"
spec:
  {{- if $obj.accessModes }}
  accessModes:
  {{- toYaml $obj.accessModes | nindent 4}}
  {{- else }}
  accessModes:
  - ReadWriteOnce
  {{- end }}
  {{- if $obj.volumeName }}
  volumeName: {{ tpl $obj.volumeName $ }}
  {{- end }}
  resources:
    requests:
      storage: {{ $obj.storage }}
  {{- if $obj.storageClass }}
  storageClassName: {{ $obj.storageClass }}
  {{- end }}
{{- end }}
{{- end }}
