{{- $needsEscapePathValue := false -}}
{{- range $_, $service := .Services -}}
    {{- range $_, $method := $service.Methods -}}
        {{- if methodPathHasPathParam $method.Path -}}{{- $needsEscapePathValue = true -}}{{- end -}}
    {{- end -}}
{{- end -}}
import { RRPCClient, RRPCResponse{{ if $needsEscapePathValue }}, escapePathValue{{ end }} } from "$lib/rrpc";

{{ template "client" . }}

{{ range $_, $service := .Services }}
{{ template "serviceHandler" $service }}
{{ end }}

{{- range $_, $struct := .Structs }}
{{ template "struct" $struct }}
{{ end }}