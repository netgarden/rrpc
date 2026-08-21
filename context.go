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
	hijacked bool
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

// Hijacked reports whether this request's response has already been
// handled by a WebSocket upgrade attempt (see Upgrade*/UpgradeRaw* in
// websocket.go) - true both when the upgrade succeeded and when it was
// attempted but failed, since the underlying WebSocket library writes its
// own response to Response() in either case. Once true, Response() is no
// longer usable for a normal HTTP response - the server will not attempt
// to write a JSON error body for whatever error the handler eventually
// returns.
func (c *Context) Hijacked() bool {
	return c.hijacked
}

// markHijacked records that a WebSocket upgrade was attempted - called
// from Upgrade*/UpgradeRaw* regardless of whether the attempt succeeded.
func (c *Context) markHijacked() {
	c.hijacked = true
}
