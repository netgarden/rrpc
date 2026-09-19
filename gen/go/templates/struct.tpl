{{- define "struct" }}
type {{ .Name }} struct {
    {{- range $fieldKey, $field := .Fields }}
    {{ firstLetterToUpper $field.Name }} {{ goFieldType $field }} `{{ goJSONTag $field }}`
    {{- end }}
}
{{- end }}