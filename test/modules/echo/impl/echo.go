package impl

import (
	"fmt"

	"github.com/netgarden/rrpc"
	"github.com/netgarden/rrpc/test/modules/echo"
)

func NewEchoService() echo.EchoService {
	return &EchoService{}
}

type EchoService struct{}

func (s *EchoService) Echo(ctx *rrpc.Context, request *echo.EchoRequest) (*echo.EchoResponse, error) {
	result := fmt.Sprintf("message=%q count=%d flag=%v", request.Message, request.Count, request.Flag)
	return &echo.EchoResponse{Result: result}, nil
}