{{ $goImports := mklist }}
{{- $needsIo := false -}}
{{- if gt (len .Services) 0 -}}
    {{- range $serviceKey, $service := .Services -}}
        {{- range $methodKey, $method := $service.Methods -}}
            {{- range $fieldKey, $field := $method.Inputs -}}
                {{- if eq $field.Type "bytes" -}}{{- $needsIo = true -}}{{- end -}}
            {{- end -}}
            {{- range $fieldKey, $field := $method.Outputs -}}
                {{- if eq $field.Type "bytes" -}}{{- $needsIo = true -}}{{- end -}}
            {{- end -}}
        {{- end -}}
    {{- end -}}
    {{- $goImports = append $goImports "github.com/netgarden/rrpc" -}}
    {{- if $needsIo -}}{{- $goImports = append $goImports "io" -}}{{- end -}}
{{- end -}}
package {{ if index .Options "go_package" }}{{ .Options.go_package }}{{ else }}rpc{{ end }}

{{ if gt (len $goImports) 0 -}}
import (
{{- range $_, $goImport := $goImports }}
    "{{ $goImport }}"
{{ end -}}
)
{{- end }}

{{- if gt (len .Structs) 0 }}

// -- Structs ---------------------------------------------

{{- range $structKey, $struct := .Structs }}
{{ template "struct" $struct }}
{{- end }}

{{- end }}

{{- if gt (len .Services) 0 }}

// -- Services --------------------------------------------

{{- range $serviceKey, $service := .Services }}
{{ template "service" $service }}
{{- end }}

{{- end }}

{{- if gt (len .Errors) 0 }}

// -- Errors ----------------------------------------------

{{ template "errors" .Errors }}

{{- end }}

