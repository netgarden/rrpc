package rrpc_gen_ts

import (
	def "github.com/netgarden/rrpc-def"
	"io/fs"
	"os"
	"path/filepath"
	"text/template"
)

func NewGenerator() def.Generator {
	return &Generator{
		fs: getTemplatesFS(),
	}
}

type Generator struct {
	fs fs.FS
}

func (g *Generator) Generate(definition *def.Module, targetDir string) error {

	baseTpl, err := template.New("").Funcs(getTemplatesFuncs()).ParseFS(g.fs, "*.tpl")
	if err != nil {
		return err
	}

	mainTpl := baseTpl.Lookup("main.tpl")

	f, err := os.Create(filepath.Join(targetDir, "rrpc.ts"))
	if err != nil {
		return err
	}
	defer f.Close()

	return mainTpl.Execute(f, definition)
}
