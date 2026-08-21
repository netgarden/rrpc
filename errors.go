package rrpc

import (
	"errors"
	"fmt"
)

type RRPCError struct {
	Name       string `json:"error"`
	Code       int    `json:"code"`
	Message    string `json:"msg"`
	Cause      string `json:"cause,omitempty"`
	HTTPStatus int    `json:"-"`
	cause      error
}

var _ error = RRPCError{}

func (e RRPCError) Error() string {
	if e.cause != nil {
		return fmt.Sprintf("%s %d: %s: %v", e.Name, e.Code, e.Message, e.cause)
	}
	return fmt.Sprintf("%s %d: %s", e.Name, e.Code, e.Message)
}

func (e RRPCError) Is(target error) bool {
	if target == nil {
		return false
	}
	if rpcErr, ok := target.(RRPCError); ok {
		return rpcErr.Code == e.Code
	}
	return errors.Is(e.cause, target)
}

func (e RRPCError) Unwrap() error {
	return e.cause
}

func (e RRPCError) WithCause(cause error) RRPCError {
	err := e
	err.cause = cause
	err.Cause = cause.Error()
	return err
}

func (e RRPCError) WithCausef(format string, args ...interface{}) RRPCError {
	cause := fmt.Errorf(format, args...)
	err := e
	err.cause = cause
	err.Cause = cause.Error()
	return err
}

// WithCauseString sets the cause from the Cause string field.
// Used after JSON unmarshaling to restore the error chain.
func (e RRPCError) WithCauseString() RRPCError {
	if e.Cause != "" {
		e.cause = errors.New(e.Cause)
	}
	return e
}

// toRRPCError normalizes any error into an RRPCError: err as-is if it
// already is one, otherwise wrapped as ErrRrpcEndpoint's cause. Shared by
// the unary JSON-error path (sendErrorJSON) and the streaming
// error-frame path (Stream.SendError/OutStream.SendError in
// websocket.go) so both produce identical output for the same input
// error.
func toRRPCError(err error) RRPCError {
	rpcErr, ok := err.(RRPCError)
	if !ok {
		rpcErr = ErrRrpcEndpoint.WithCause(err)
	}
	return rpcErr
}

// RRPC errors
var (
	ErrRrpcEndpoint           = RRPCError{Code: 0, Name: "RrpcEndpoint", Message: "endpoint error", HTTPStatus: 400}
	ErrRrpcRequestFailed      = RRPCError{Code: -1, Name: "RrpcRequestFailed", Message: "request failed", HTTPStatus: 400}
	ErrRrpcBadRoute           = RRPCError{Code: -2, Name: "RrpcBadRoute", Message: "bad route", HTTPStatus: 404}
	ErrRrpcBadMethod          = RRPCError{Code: -3, Name: "RrpcBadMethod", Message: "bad method", HTTPStatus: 405}
	ErrRrpcBadRequest         = RRPCError{Code: -4, Name: "RrpcBadRequest", Message: "bad request", HTTPStatus: 400}
	ErrRrpcBadResponse        = RRPCError{Code: -5, Name: "RrpcBadResponse", Message: "bad response", HTTPStatus: 500}
	ErrRrpcServerPanic        = RRPCError{Code: -6, Name: "RrpcServerPanic", Message: "server panic", HTTPStatus: 500}
	ErrRrpcInternalError      = RRPCError{Code: -7, Name: "RrpcInternalError", Message: "internal error", HTTPStatus: 500}
	ErrRrpcClientDisconnected = RRPCError{Code: -8, Name: "RrpcClientDisconnected", Message: "client disconnected", HTTPStatus: 400}
)
