{{- $needsMultipart := false -}}
{{- $needsPathParams := false -}}
{{- $needsBytes := false -}}
{{- range $serviceKey, $service := .Services -}}
    {{- range $methodKey, $method := $service.Methods -}}
        {{- if eq $method.ContentType "multipart/form-data" -}}{{- $needsMultipart = true -}}{{- $needsBytes = true -}}{{- end -}}
        {{- if gt (len (methodPathParams $method.Path)) 0 -}}{{- $needsPathParams = true -}}{{- end -}}
        {{- range $fieldKey, $input := $method.Inputs -}}
            {{- if $input.IsStruct -}}{{- $needsBytes = true -}}{{- end -}}
        {{- end -}}
    {{- end -}}
{{- end -}}
package {{ if index .Options "go_package" }}{{ .Options.go_package }}{{ else }}rpc{{ end }}

import (
    {{ if $needsBytes }}"bytes"{{ end -}}
    "context"
    "encoding/json"
    {{ if $needsPathParams }}"fmt"{{ end -}}
    "io"
    {{ if $needsMultipart }}"mime/multipart"{{ end -}}
    "net/http"
    {{ if $needsPathParams }}"net/url"{{ end -}}
    "strconv"

    "github.com/netgarden/rrpc"
)

{{ range $_, $service := .Services -}}
{{ template "serviceHandler" $service }}
{{ template "serviceClientHandler" $service }}
{{ end }}