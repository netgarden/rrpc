package {{ if index .Options "go_package" }}{{ .Options.go_package }}{{ else }}rpc{{ end }}

import (
    "bytes"
    "context"
    "encoding/json"
    "io"
    "net/http"
    "strconv"

    "github.com/netgarden/rrpc"
)

{{ range $_, $service := .Services -}}
{{ template "serviceHandler" $service }}
{{ template "serviceClientHandler" $service }}
{{ end }}