package rrpc

import "net/http"

type Context struct {
	request  *http.Request
	response http.ResponseWriter
	params   *Params
}

func NewContext(request *http.Request, response http.ResponseWriter, params *Params) *Context {
	return &Context{
		request:  request,
		response: response,
		params:   params,
	}
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
