package impl

import (
	"github.com/netgarden/rrpc"
	"github.com/netgarden/rrpc-test/modules/auth"
)

func NewAuthService() auth.AuthService {
	return &AuthService{}
}

type AuthService struct{}

func (s *AuthService) Login(ctx *rrpc.Context, request *auth.LoginRequest) (*auth.LoginResponse, error) {
	return &auth.LoginResponse{
		Ok: true,
	}, nil
}

func (s *AuthService) Logout(ctx *rrpc.Context) error {
	return nil
}
