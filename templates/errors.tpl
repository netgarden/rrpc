{{ define "errors" -}}

{{ range $_, $error := .Errors -}}
export class {{ firstLetterToUpper $error.Name }} extends Error {
    code: number = {{ $error.Code }};
    status: number = {{ $error.HttpCode }};
    constructor() {
        super("{{ $error.Message }}")
    }
}
{{ end }}

{{- end }}