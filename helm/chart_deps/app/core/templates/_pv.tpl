{{- define "core.pv" -}}
{{- $ := index . 0 }}
{{- $labels := index . 1 }}
{{- $obj := include "core.obj.enricher" (list $ $labels (index . 2)) | fromYaml }}
{{- if $obj.enabled }}
---
apiVersion: v1
kind: PersistentVolume
metadata:
  labels:
{{- include "core.labels.constructor" (list $ $labels $obj) | nindent 4 }}
  name: {{ tpl $obj.name $ }}
spec:
  {{- if $obj.accessModes }}
  accessModes:
  {{- toYaml $obj.accessModes | nindent 4}}
  {{- else }}
  accessModes:
  - ReadWriteOnce
  {{- end }}
  capacity:
    storage: {{ $obj.storage }}
  {{- if $obj.storageClass }}
  storageClassName: {{ $obj.storageClass }}
  {{- end }}
  {{- with $obj.mountOptions }}
  mountOptions:
  {{- toYaml . | nindent 4 }}
  {{- end }}
  {{- with $obj.csi }}
  csi:
  {{- tpl (toYaml .) $ | nindent 4 }}
  {{- end }}
  {{- with $obj.claimRef }}
  claimRef:
  {{- tpl (toYaml .) $ | nindent 4 }}
  {{- end }}
{{- end }}
{{- end }}
