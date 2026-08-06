{{/*
Expand the name of the chart.
*/}}
{{- define "driftdetect.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create a default fully qualified app name.
*/}}
{{- define "driftdetect.fullname" -}}
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

{{- define "driftdetect.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{- define "driftdetect.labels" -}}
helm.sh/chart: {{ include "driftdetect.chart" . }}
{{ include "driftdetect.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end }}

{{- define "driftdetect.selectorLabels" -}}
app.kubernetes.io/name: {{ include "driftdetect.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{- define "driftdetect.serviceAccountName" -}}
{{- if .Values.serviceAccount.create }}
{{- default (include "driftdetect.fullname" .) .Values.serviceAccount.name }}
{{- else }}
{{- default "default" .Values.serviceAccount.name }}
{{- end }}
{{- end }}

{{- define "driftdetect.image" -}}
{{- $tag := default .Chart.AppVersion .Values.image.tag }}
{{- printf "%s:%s" .Values.image.repository $tag }}
{{- end }}

{{- define "driftdetect.configMapName" -}}
{{- if .Values.config.existingConfigMap }}
{{- .Values.config.existingConfigMap }}
{{- else }}
{{- printf "%s-config" (include "driftdetect.fullname" .) }}
{{- end }}
{{- end }}

{{- define "driftdetect.databaseSecretName" -}}
{{- if .Values.database.existingSecret }}
{{- .Values.database.existingSecret }}
{{- else }}
{{- printf "%s-database" (include "driftdetect.fullname" .) }}
{{- end }}
{{- end }}

{{- define "driftdetect.apiKeysSecretName" -}}
{{- if .Values.apiKeys.existingSecret }}
{{- .Values.apiKeys.existingSecret }}
{{- else if .Values.config.server.api_keys }}
{{- printf "%s-api-keys" (include "driftdetect.fullname" .) }}
{{- else }}
{{- "" }}
{{- end }}
{{- end }}

{{- define "driftdetect.databaseHost" -}}
{{- printf "%s-postgresql" (include "driftdetect.fullname" .) }}
{{- end }}

{{- define "driftdetect.databaseURL" -}}
{{- if .Values.postgresql.enabled }}
{{- printf "postgres://%s:%s@%s:5432/%s?sslmode=disable" .Values.postgresql.auth.username .Values.postgresql.auth.password (include "driftdetect.databaseHost" .) .Values.postgresql.auth.database }}
{{- else }}
{{- .Values.database.url }}
{{- end }}
{{- end }}
