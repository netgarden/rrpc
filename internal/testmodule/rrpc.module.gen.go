package testmodule

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
    return "testmodule"
}

func (s *Module) Services() map[string]rrpc.ServiceHandler {
    return s.services
}


func (s *Module) SetComputeService(computeService ComputeService) {
    s.services["Compute"] = newComputeServiceHandler(computeService)
}

func (s *Module) SetEchoService(echoService EchoService) {
    s.services["Echo"] = newEchoServiceHandler(echoService)
}

func (s *Module) SetPingService(pingService PingService) {
    s.services["Ping"] = newPingServiceHandler(pingService)
}

func (s *Module) SetStoreService(storeService StoreService) {
    s.services["Store"] = newStoreServiceHandler(storeService)
}

