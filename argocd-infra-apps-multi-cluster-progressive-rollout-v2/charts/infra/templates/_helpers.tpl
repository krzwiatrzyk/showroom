{{- define "infra.validate" -}}
{{- if not .Values.clusterName }}{{ fail "clusterName is required" }}{{- end }}
{{- end -}}
