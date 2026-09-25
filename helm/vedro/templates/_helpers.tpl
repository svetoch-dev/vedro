{{- define "vedro.providerName" -}}
{{- $providers := .root.Values.providers | default dict -}}
{{- if .ref -}}
  {{- $provider := index $providers .ref -}}
  {{- if $provider -}}{{ default .ref $provider.name }}{{- else -}}{{ .ref }}{{- end -}}
{{- else if eq (len $providers) 1 -}}
  {{- range $key, $provider := $providers -}}{{ default $key $provider.name }}{{- end -}}
{{- else -}}
  {{- fail "set provider on each principal or bucket unless exactly one provider is configured" -}}
{{- end -}}
{{- end -}}

{{- define "vedro.accessName" -}}
{{- $id := printf "%s-%s-%s-%s" .bucket .namespace .principal .level | lower -}}
{{- printf "%s-%s" (trimSuffix "-" (trunc 54 $id)) (trunc 8 (sha256sum $id)) -}}
{{- end -}}
