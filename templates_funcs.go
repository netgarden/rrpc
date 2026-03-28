package rrpc_gen_ts

import (
	def "github.com/netgarden/rrpc-def"
	"text/template"
)

func getTemplatesFuncs() template.FuncMap {

	funcs := def.GetTemplateFuncMap()
	funcs["rrpcDataTypeToTSType"] = rrpcDataTypeToTSType

	return funcs
}
