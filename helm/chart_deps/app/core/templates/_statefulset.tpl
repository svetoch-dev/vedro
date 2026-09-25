{{- define "core.statefulset" -}}
{{- $ := index . 0 }}
{{- $labels := index . 1 }}
{{- $obj := include "core.obj.enricher" (list $ $labels (index . 2)) | fromYaml }}
{{- if $obj.enabled }}
---
apiVersion: apps/v1
kind: StatefulSet
metadata:
  name: {{ $obj.name }}
  labels:
{{- include "core.labels.constructor" (list $ $labels $obj) | nindent 4 }}
  namespace: "{{ $obj.namespace }}"
spec:
  replicas: {{ $obj.replicaCount }}
  {{- with $obj.persistentVolumeClaimRetentionPolicy }}
  persistentVolumeClaimRetentionPolicy:
  {{- tpl (toYaml .) $ | nindent 4 }}
  {{- end }}
  {{- if $obj.volumeClaimTemplates }}
  volumeClaimTemplates:
  {{- range $name, $pvcTemplate :=  $obj.volumeClaimTemplates }}
  {{- $pvcTemplate = set $pvcTemplate "name" $name }}
  {{- $pvcTemplate = set $pvcTemplate "volumeClaimTampleate" true }}
  - {{ (include "core.pvc" (list $ $labels $pvcTemplate )) | nindent 4 | trim }} 
  {{- end }}
  {{- end }}
  {{- if $obj.revisionHistoryLimit  }}
  revisionHistoryLimit: {{ $obj.revisionHistoryLimit }}
  {{- end }}
  {{- with $obj.selectorLabels }}
  selector:
    matchLabels:
      {{- tpl (toYaml .) $ | nindent 8 }}
  {{- end }}
  {{ include "core.podtemplate" (list $ $obj) | nindent 2 | trim }}
{{- end }}
{{- end }}
