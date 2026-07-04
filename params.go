package rrpc

import "errors"

func NewParams() *Params {
	return &Params{
		data: make(map[string]interface{}),
	}
}

type Params struct {
	data map[string]interface{}
}

func (p *Params) GetString(key string) string {
	return p.data[key].(string)
}

func (p *Params) SetString(key string, value string) {
	p.data[key] = value
}

func processParam(params *Params, param PathParam, paramStrValue string) error {

	switch param.Type {
	case "path", "string":
		params.SetString(param.Name, paramStrValue)
		break
	default:
		return errors.New("Unsupported path param type: " + param.Type)
	}

	return nil
}
