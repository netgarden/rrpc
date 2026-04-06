package auth

import (
    "github.com/netgarden/rrpc"
)

// -- Structs ---------------------------------------------

type LoginRequest struct {
    Username string `json:"username"`
    Password string `json:"password"`
}

type LoginResponse struct {
    Ok bool `json:"ok"`
}

// -- Services --------------------------------------------

type AuthService interface {
    Login(ctx *rrpc.Context, request *LoginRequest) (*LoginResponse, error)
    Logout(ctx *rrpc.Context) error
}

// -- Errors ----------------------------------------------


var (
    ErrAuthenticationError  = rrpc.RRPCError{Code: 1, Name: "AuthenticationError", Message: "authentication failed", HTTPStatus: 401}
)


