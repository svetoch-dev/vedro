{{- define "vedro.accessName" -}}
{{- $id := printf "%s-%s-%s" .bucket .principal .level | lower -}}
{{- printf "%s-%s" (trimSuffix "-" (trunc 54 $id)) (trunc 8 (sha256sum $id)) -}}
{{- end -}}

{{/*
Create chart name and version as used by the chart label.
*/}}
{{- define "vedro.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Common labels
*/}}
{{- define "vedro.labels" -}}
helm.sh/chart: {{ include "vedro.chart" . }}
app.kubernetes.io/name: {{ .Chart.Name | trunc 63 | trimSuffix "-" }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end }}

{{- define "vedro.principal.auth.validate" }}
{{- $auth := . }}
{{- $auth = fromYaml $auth }}
{{- if not $auth.method }}
{{- fail "principal.auth.method is required" }}
{{- end }}
{{- if eq $auth.method "WorkloadIdentity" }}
{{- if not $auth.workloadIdentity.serviceAccountRef }}
{{- fail "principal.auth.workloadIdentity.serviceAccountRef.name is required" }}
{{- end }}
{{- if not $auth.workloadIdentity.serviceAccountRef.name }}
{{- fail "principal.auth.workloadIdentity.serviceAccountRef.name is required" }}
{{- end }}
{{- if not $auth.workloadIdentity.deletionPolicy }}
{{- $_ := set $auth.workloadIdentity "deletionPolicy" "Delete" }}
{{- end }}
{{- else if eq $auth.method "StaticCredentials" }}
{{- if not $auth.staticCredentials.secretRef }} 
{{- fail "principal.auth.staticCredentials.secretRef.name is required" }}
{{- end }}
{{- if not $auth.staticCredentials.secretRef.name }}
{{- fail "principal.auth.staticCredentials.secretRef.name is required" }}
{{- end }}
{{- if not $auth.staticCredentials.deletionPolicy }}
{{- $_ := set $auth.staticCredentials "deletionPolicy" "Delete" }}
{{- end }}
{{- else }}
{{- fail "principal.auth.method must be WorkloadIdentity or StaticCredentials" }}
{{- end }}
{{- $auth | toYaml }}
{{- end }}


{{- define "vedro.principal.auth" -}}
{{- $ := index . 0 }}
{{- $auth := index . 1 }}
{{- if kindIs "string" $auth }}
{{- $auth = tpl $auth $ }}
{{- else if kindIs "map" $auth }}
{{- $auth = tpl (toYaml $auth) $ }}
{{- else }}
{{- fail "principal.auth must be either yaml str or a map" }}
{{- end }}
{{- include "vedro.principal.auth.validate" $auth }}
{{- end }}
