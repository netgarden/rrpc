{{- define "service" }}
type {{ firstLetterToUpper .Name }}Service interface {
    {{- range $methodKey, $method := .Methods }}
    {{- if $method.Streaming }}
    {{ firstLetterToUpper $method.Name }}(ctx *rrpc.Context, stream *rrpc.{{ streamGoType $method }}) error
    {{- else }}
    {{ firstLetterToUpper $method.Name -}}
        (ctx *rrpc.Context
        {{- range $paramKey, $param := $method.Inputs -}}
        , {{ if $param.IsStruct }}{{ $param.Name }} *{{ $param.Type }}{{ else if eq $param.Type "bytes"}}r io.Reader{{ end }}
        {{- end }}
        {{- if eq (len $method.Outputs) 1 }}{{ if eq (index $method.Outputs 0).Type "bytes" }}, w io.Writer{{ end }}{{ end -}}
    )
    {{- if eq (len $method.Outputs) 0 }} error
    {{- else if eq (index $method.Outputs 0).Type "bytes" }} error
    {{- else if gt (len $method.Outputs) 0 }} (
        {{- range $_, $output := $method.Outputs }}{{ if $output.IsStruct }}*{{ $output.Type }}{{ end }}, {{ end -}}
        error)
    {{- end }}
    {{- end }}
    {{- end }}
}
{{- end }}