{{- define "core.ingress" }}
{{- $ := index . 0 }}
{{- $labels := index . 1 }}
{{- $obj := include "core.obj.enricher" (list $ $labels (index . 2)) | fromYaml }}
{{- if $obj.enabled -}}
{{- $svcPort :=  $obj.service.port -}}
{{- $svcName := tpl $obj.service.name $ -}}
{{- if and $obj.className (not (semverCompare ">=1.18-0" $.Capabilities.KubeVersion.GitVersion)) }}
  {{- if not (hasKey $obj.annotations "kubernetes.io/obj.class") }}
  {{- $_ := set $obj.annotations "kubernetes.io/obj.class" $obj.className}}
  {{- end }}
{{- end }}
---
{{ if semverCompare ">=1.19-0" $.Capabilities.KubeVersion.GitVersion -}}
apiVersion: networking.k8s.io/v1
{{- else if semverCompare ">=1.14-0" $.Capabilities.KubeVersion.GitVersion -}}
apiVersion: networking.k8s.io/v1beta1
{{- else -}}
apiVersion: extensions/v1beta1
{{- end }}
kind: Ingress
metadata:
  name: {{ tpl $obj.name $ }}
  labels:
{{- include "core.labels.constructor" (list $ $labels $obj) | nindent 4 }}
  namespace: "{{ $obj.namespace }}"
  {{- with $obj.annotations }}
  annotations:
    {{- tpl (toYaml .) $ | nindent 4 }}
  {{- end }}
spec:
  {{- if and $obj.className (semverCompare ">=1.18-0" $.Capabilities.KubeVersion.GitVersion) }}
  ingressClassName: {{ tpl $obj.className $}}
  {{- end }}
  {{- if $obj.tls }}
  tls:
    {{- range $obj.tls }}
    - hosts:
        {{- range .hosts }}
        - {{ tpl . $ | quote }}
        {{- end }}
      secretName: {{ .secretName }}
    {{- end }}
  {{- end }}
  rules:
    {{- range $obj.hosts }}
    - host: {{ tpl .host $ | quote }}
      {{- $svcPort := or .servicePort $svcPort }}
      {{- $svcName := or .serviceName $svcName }}
      {{- $svcName := tpl $svcName $ }}
      http:
        paths:
          {{- range .paths }}
          - path: {{ .path }}
            {{- if and .pathType (semverCompare ">=1.18-0" $.Capabilities.KubeVersion.GitVersion) }}
            pathType: {{ .pathType }}
            {{- end }}
            backend:
              {{- if semverCompare ">=1.19-0" $.Capabilities.KubeVersion.GitVersion }}
              service:
                name: {{ $svcName }}
                port:
                  number: {{ $svcPort }}
              {{- else }}
              serviceName: {{ $svcName }}
              servicePort: {{ $svcPort }}
              {{- end }}
          {{- end }}
    {{- end }}
{{- end }}
{{- end }}
