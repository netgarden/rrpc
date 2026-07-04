package rrpc

import (
	"net/http"
	"strings"
)

type Context struct {
	request  *http.Request
	response http.ResponseWriter
	params   *Params
	values   map[any]any
}

func NewContext(request *http.Request, response http.ResponseWriter, params *Params) *Context {
	return &Context{
		request:  request,
		response: response,
		params:   params,
	}
}

func (c *Context) Set(key, value any) {
	if c.values == nil {
		c.values = make(map[any]any)
	}
	c.values[key] = value
}

func (c *Context) Get(key any) (any, bool) {
	v, ok := c.values[key]
	return v, ok
}

func (c *Context) Request() *http.Request {
	return c.request
}

func (c *Context) Response() http.ResponseWriter {
	return c.response
}

func (c *Context) Params() *Params {
	return c.params
}

func (c *Context) IsHTTPS() bool {
	if c.request.TLS != nil {
		return true
	}
	return strings.EqualFold(c.request.Header.Get("X-Forwarded-Proto"), "https")
}
