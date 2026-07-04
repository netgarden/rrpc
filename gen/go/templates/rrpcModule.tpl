package {{ if index .Options "go_package" }}{{ .Options.go_package }}{{ else }}rpc{{ end }}

import (
    "github.com/netgarden/rrpc"
)

func NewModule() *Module {
    return &Module{
        services: make(map[string]rrpc.ServiceHandler),
    }
}

type Module struct {
    services map[string]rrpc.ServiceHandler
}

func (s *Module) Name() string {
    return "{{ .Name }}"
}

func (s *Module) Services() map[string]rrpc.ServiceHandler {
    return s.services
}

{{ range $_, $service := .Services }}
func (s *Module) Set{{ $service.Name }}Service({{ firstLetterToLower $service.Name }}Service {{ $service.Name }}Service) {
    s.services["{{ $service.Name }}"] = new{{ firstLetterToUpper $service.Name }}ServiceHandler({{ firstLetterToLower $service.Name }}Service)
}
{{ end }}
