package rrpc

import (
	"fmt"
	def "github.com/netgarden/rrpc/def"
	rrpcGenGo "github.com/netgarden/rrpc/gen/go"
	rrpcGenTs "github.com/netgarden/rrpc/gen/ts"
)

func NewGenerator(genType string) (def.Generator, error) {

	switch genType {
	case "go":
		return rrpcGenGo.NewGenerator(), nil
	case "ts":
		return rrpcGenTs.NewGenerator(), nil
	default:
		return nil, fmt.Errorf("unsupported generator type: %s", genType)
	}

}
