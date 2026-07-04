{{ define "struct" -}}

class {{ firstLetterToUpper .Name }} {
    {{- range $_, $field := .Fields }}
    {{ firstLetterToLower $field.Name }}!: {{ if $field.IsStruct }}{{ firstLetterToLower $field.Type }}{{ else }}{{ rrpcDataTypeToTSType $field.Type }}{{ end }};
    {{- end }}
}

{{- end }}