{{- define "core.clusterrole" -}}
{{- $ := index . 0 }}
{{- $labels      := index . 1 }}
{{- $obj := include "core.obj.enricher" (list $ $labels (index . 2)) | fromYaml }}
{{- if $obj.enabled }}
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  labels:
{{- include "core.labels.constructor" (list $ $labels $obj) | nindent 4 }}
  name: {{ tpl $obj.name $ }} 
rules:
{{- with $obj.rules }}
{{- tpl (toYaml .) $ | nindent 2 }}
{{- end }}
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  labels:
{{- include "core.labels.constructor" (list $ $labels $obj) | nindent 4 }}
  name: {{ tpl $obj.name $ }} 
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: {{ tpl $obj.name $ }} 
subjects:
- kind: ServiceAccount
  name: {{ tpl $obj.serviceAccount.name $ }}
  namespace: {{ tpl (default $obj.namespace $obj.serviceAccount.namespace) $ }}
{{- end -}}
{{- end -}}
