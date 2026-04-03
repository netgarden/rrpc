package test

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
    return "test"
}

func (s *Module) Services() map[string]rrpc.ServiceHandler {
    return s.services
}

func (s *Module) SetPingerService(pingerService PingerService) {
    s.services["Pinger"] = newPingerServiceHandler(pingerService)
}func (s *Module) SetStaticService(staticService StaticService) {
    s.services["Static"] = newStaticServiceHandler(staticService)
}
