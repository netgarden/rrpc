{{- $needsMultipart := false -}}
{{- range $serviceKey, $service := .Services -}}
    {{- range $methodKey, $method := $service.Methods -}}
        {{- if eq $method.ContentType "multipart/form-data" -}}{{- $needsMultipart = true -}}{{- end -}}
    {{- end -}}
{{- end -}}
package {{ if index .Options "go_package" }}{{ .Options.go_package }}{{ else }}rpc{{ end }}

import (
    "bytes"
    "context"
    "encoding/json"
    "io"
    {{ if $needsMultipart }}"mime/multipart"
    {{ end }}"net/http"
    "strconv"

    "github.com/netgarden/rrpc"
)

{{ range $_, $service := .Services -}}
{{ template "serviceHandler" $service }}
{{ template "serviceClientHandler" $service }}
{{ end }}