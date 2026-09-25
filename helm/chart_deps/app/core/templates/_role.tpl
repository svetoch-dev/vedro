{{- define "core.role" -}}
{{- $ := index . 0 }}
{{- $labels := index . 1 }}
{{- $obj := include "core.obj.enricher" (list $ $labels (index . 2)) | fromYaml }}
{{- if $obj.enabled }}
---
apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  labels:
{{- include "core.labels.constructor" (list $ $labels $obj) | nindent 4 }}
  name: {{ tpl $obj.name $ }} 
  namespace: "{{ $obj.namespace }}"
rules:
{{- with $obj.rules }}
{{- tpl (toYaml .) $ | nindent 2 }}
{{- end }}
---
apiVersion: rbac.authorization.k8s.io/v1
kind: RoleBinding
metadata:
  labels:
{{- include "core.labels.constructor" (list $ $labels $obj) | nindent 4 }}
  name: {{ tpl $obj.name $ }} 
  namespace: "{{ $obj.namespace }}"
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: Role
  name: {{ tpl $obj.name $ }} 
subjects:
- kind: ServiceAccount
  name: {{ tpl $obj.serviceAccount.name $ }}
  namespace: {{ tpl $obj.serviceAccount.namespace $ }}
{{- end -}}
{{- end -}}
