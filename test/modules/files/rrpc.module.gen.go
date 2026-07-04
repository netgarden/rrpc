package files

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
    return "files"
}

func (s *Module) Services() map[string]rrpc.ServiceHandler {
    return s.services
}


func (s *Module) SetFilesService(filesService FilesService) {
    s.services["Files"] = newFilesServiceHandler(filesService)
}

