{{- $needsMultipart := false -}}
{{- $needsPathParams := false -}}
{{- $needsUrlEscape := false -}}
{{- $needsBytes := false -}}
{{- range $serviceKey, $service := .Services -}}
    {{- range $methodKey, $method := $service.Methods -}}
        {{- if eq $method.ContentType "multipart/form-data" -}}{{- $needsMultipart = true -}}{{- $needsBytes = true -}}{{- end -}}
        {{- if gt (len (methodPathParams $method.Path)) 0 -}}{{- $needsPathParams = true -}}{{- end -}}
        {{- if methodPathHasStringParam $method.Path -}}{{- $needsUrlEscape = true -}}{{- end -}}
        {{- range $fieldKey, $input := $method.Inputs -}}
            {{- /* Streaming methods send their struct over a WebSocket, never via bytes.NewBuffer. */ -}}
            {{- /* A GET's struct goes in the query string, so it needs no bytes.NewBuffer either. */ -}}
            {{- if and $input.IsStruct (not $method.Streaming) (ne $method.Type "GET") -}}{{- $needsBytes = true -}}{{- end -}}
        {{- end -}}
    {{- end -}}
{{- end -}}
package {{ if index .Module.Options "go_package" }}{{ .Module.Options.go_package }}{{ else }}rpc{{ end }}

import (
    {{ if $needsBytes }}"bytes"
    {{ end -}}"context"
    "encoding/json"
    {{ if $needsPathParams }}"fmt"
    {{ end -}}"io"
    {{ if $needsMultipart }}"mime/multipart"
    {{ end -}}"net/http"
    {{ if $needsUrlEscape }}"net/url"
    {{ end -}}"strconv"

    "github.com/netgarden/rrpc"
)

{{ range $_, $service := .Services -}}
{{ template "serviceHandler" $service }}
{{ template "serviceClientHandler" $service }}
{{ end }}