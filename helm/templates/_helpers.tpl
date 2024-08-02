{{/*
Expand the name of the chart.
*/}}
{{- define "signer.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create a default fully qualified app name.
We truncate at 63 chars because some Kubernetes name fields are limited to this (by the DNS naming spec).
If release name contains chart name it will be used as a full name.
*/}}
{{- define "signer.fullname" -}}
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
{{- define "signer.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Common labels
*/}}
{{- define "signer.labels" -}}
helm.sh/chart: {{ include "signer.chart" . }}
{{ include "signer.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end }}

{{/*
Selector labels
*/}}
{{- define "signer.selectorLabels" -}}
app.kubernetes.io/name: {{ include "signer.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{/*
Create the name of the cluster role to use
*/}}
{{- define "signer.roleName" -}}
{{- default (include "signer.fullname" .) .Values.role.name }}
{{- end }}

{{/*
Create the name of the cluster role binding to use
*/}}
{{- define "signer.roleBindingName" -}}
{{- default (include "signer.fullname" .) .Values.roleBinding.name }}
{{- end }}

{{/*
Create the name of the secret to use
*/}}
{{- define "signer.secretName" -}}
{{- default (include "signer.fullname" .) .Values.secret.name }}
{{- end }}

{{/*
Create the name of the service account to use
*/}}
{{- define "signer.serviceAccountName" -}}
{{- default (include "signer.fullname" .) .Values.serviceAccount.name }}
{{- end }}