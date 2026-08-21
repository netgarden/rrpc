{{- $needsEscapePathValue := false -}}
{{- $needsStream := false -}}
{{- $needsInStream := false -}}
{{- $needsOutStream := false -}}
{{- range $_, $service := .Services -}}
    {{- range $_, $method := $service.Methods -}}
        {{- if methodPathHasPathParam $method.Path -}}{{- $needsEscapePathValue = true -}}{{- end -}}
        {{- if eq $method.StreamDirection "both" -}}{{- $needsStream = true -}}{{- end -}}
        {{- if eq $method.StreamDirection "in" -}}{{- $needsInStream = true -}}{{- end -}}
        {{- if eq $method.StreamDirection "out" -}}{{- $needsOutStream = true -}}{{- end -}}
    {{- end -}}
{{- end -}}
import { RRPCClient, RRPCResponse{{ if $needsEscapePathValue }}, escapePathValue{{ end }}{{ if $needsStream }}, RRPCStream{{ end }}{{ if $needsInStream }}, RRPCInStream{{ end }}{{ if $needsOutStream }}, RRPCOutStream{{ end }} } from "{{ rrpcImportPath }}";

{{ template "client" . }}

{{ range $_, $service := .Services }}
{{ template "serviceHandler" $service }}
{{ end }}

{{- range $_, $struct := .Structs }}
{{ template "struct" $struct }}
{{ end }}