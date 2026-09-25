{{/*
Expand the name of the chart.
*/}}
{{- define "common.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create a default fully qualified app name.
We truncate at 63 chars because some Kubernetes name fields are limited to this (by the DNS naming spec).
If release name contains chart name it will be used as a full name.
*/}}
{{- define "common.fullname" -}}
{{- if .Values.fullnameOverride }}
{{- .Values.fullnameOverride | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- $name := default .Chart.Name .Values.nameOverride }}
{{- if contains $name .Release.Name }}
{{- .Release.Name | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- printf "%s-%s" .Release.Name $name | trunc 63 | trimSuffix "-" }}
{{- end }}
{{- end }}
{{- end }}

{{/*
Create chart name and version as used by the chart label.
*/}}
{{- define "common.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Common labels
*/}}
{{- define "common.labels" -}}
helm.sh/chart: {{ include "common.chart" . }}
{{ include "common.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end }}

{{/*
Selector labels
*/}}
{{- define "common.selectorLabels" -}}
app.kubernetes.io/name: {{ include "common.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{/*
Create the name of the service account to use
*/}}
{{- define "common.serviceAccountName" -}}
{{- if .Values.serviceAccount.create }}
{{- default (include "common.fullname" .) .Values.serviceAccount.name }}
{{- else }}
{{- default "default" .Values.serviceAccount.name }}
{{- end }}
{{- end }}

{{- define "common.env" }}
env:
{{- if not .Values.disableGlobalEnvironmentFromSecrets }}
{{- with .Values.global.environmentFromSecrets }}
{{- tpl (toYaml .) $ | nindent 2 }}
{{- end }}
{{- end }}
{{- with .Values.environmentFromSecrets }}
{{- tpl (toYaml .) $ | nindent 2 }}
{{- end }}
{{- with .Values.environment }}
{{- tpl (toYaml .) $ | nindent 2 }}
{{- end }}
{{- if not .Values.disableGlobalEnvironment }}
{{- with .Values.global.environment }}
{{- tpl (toYaml .) $ | nindent 2 }}
{{- end }}
{{- end }}
{{- end }}

{{- define "common.volumeMounts" }}
{{- if or .Values.pvcs .Values.volumeMounts }}
volumeMounts:
  {{- range $name, $pvc := .Values.pvcs }}
  - name: {{ $name }}
    mountPath: {{ $pvc.mountPath }}
  {{- end }}
  {{- with .Values.volumeMounts }}
  {{- tpl (toYaml .) $ | nindent 2 }}
  {{- end }}
{{- end }}
{{- end }}

{{- define "common.initContainers" }}
initContainers:
{{- range $name, $container := .Values.initContainers }}
  - name: {{ $name }}
    {{- with $container.command }}
    command:
    {{- toYaml . | nindent 6 }}
    {{- end }}
    image: "{{ $container.image }}"
    {{- include "common.volumeMounts" $ | nindent 4 }}
    {{- include "common.env" $ | nindent 4 }}
{{- end }}
{{- end }}

{{- define "common.volumes" }}
{{- if or .Values.pvcs .Values.volumes }}
volumes:
  {{- range $name, $pvc := .Values.pvcs }}
  - name: {{ $name }}
    persistentVolumeClaim:
      claimName: {{ include "common.fullname" $ }}-{{ $name }}
  {{- end }}
  {{- with .Values.volumes }}
  {{- tpl (toYaml .) $ | nindent 2 }}
  {{- end }}
{{- end }}
{{- end }}
