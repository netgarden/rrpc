package echo

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
    return "echo"
}

func (s *Module) Services() map[string]rrpc.ServiceHandler {
    return s.services
}


func (s *Module) SetEchoService(echoService EchoService) {
    s.services["Echo"] = newEchoServiceHandler(echoService)
}

