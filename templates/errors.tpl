{{- define "errors" }}
var (
{{- range $_, $error := . }}
    Err{{ $error.Name }}  = RRPCError{Code: {{ $error.Code }}, Name: "{{ $error.Name }}", Message: "{{ $error.Message }}", HTTPStatus: {{ $error.HttpCode }}}
{{ end -}}
)
{{ end -}}