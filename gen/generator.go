package rrpc

import (
	"fmt"
	def "github.com/netgarden/rrpc/def"
	rrpcGenGo "github.com/netgarden/rrpc/gen/go"
	rrpcGenTs "github.com/netgarden/rrpc/gen/ts"
)

// tsRRPCImportPath, if non-empty, overrides where "ts"-generated code
// imports the RRPC TS runtime from (default "$lib/rrpc") — see
// rrpc_gen_ts.WithRRPCImportPath. Ignored for "go".
func NewGenerator(genType string, tsRRPCImportPath string) (def.Generator, error) {

	switch genType {
	case "go":
		return rrpcGenGo.NewGenerator(), nil
	case "ts":
		return rrpcGenTs.NewGenerator(rrpcGenTs.WithRRPCImportPath(tsRRPCImportPath)), nil
	default:
		return nil, fmt.Errorf("unsupported generator type: %s", genType)
	}

}
