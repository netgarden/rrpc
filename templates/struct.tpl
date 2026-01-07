{{- define "struct" }}
type {{ .Name }} struct {
    {{- range $fieldKey, $field := .Fields }}
    {{ firstLetterToUpper $field.Name }} {{ $field.Type }} `json:"{{ $field.Name }}"`
    {{- end }}
}
{{- end }}