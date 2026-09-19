{{ define "struct" -}}

class {{ firstLetterToUpper .Name }} {
    {{- range $_, $field := .Fields }}
    {{ firstLetterToLower $field.Name }}{{ if $field.Optional }}?{{ else }}!{{ end }}: {{ if $field.IsStruct }}{{ firstLetterToUpper $field.Type }}{{ else }}{{ rrpcDataTypeToTSType $field.Type }}{{ end }};
    {{- end }}
}

{{- end }}