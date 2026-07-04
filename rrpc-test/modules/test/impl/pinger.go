package impl

import (
	"github.com/netgarden/rrpc"
	"github.com/netgarden/rrpc-test/modules/test"
)

func NewPingerService() test.PingerService {
	return &PingerService{}
}

type PingerService struct{}

func (s *PingerService) Ping(ctx *rrpc.Context) error {
	return nil
}
