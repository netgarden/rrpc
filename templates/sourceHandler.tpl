package {{ if index .Options "go_package" }}{{ .Options.go_package }}{{ else }}rpc{{ end }}

import (
    "bytes"
    "context"
    "encoding/json"
    "errors"
    "io"
    "net/http"
    "strconv"
)

{{ range $_, $service := .Services -}}
{{ template "serviceHandler" $service }}
{{ template "serviceClientHandler" $service }}
{{ end }}