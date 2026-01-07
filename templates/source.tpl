{{ $goImports := mklist }}
{{- if gt (len .Services) 0 -}}
{{- $goImports = append $goImports "context" -}}
{{- end -}}
package {{ if index .Options "go_package" }}{{ .Options.go_package }}{{ else }}rpc{{ end }}

{{ if gt (len $goImports) 0 -}}
import (
{{- range $_, $goImport := $goImports }}
    "{{ $goImport }}"
    "io"
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

