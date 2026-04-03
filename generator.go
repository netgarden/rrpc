package rrpc_gen_go

import (
	"io/fs"
	"os"
	"path/filepath"
	"text/template"

	def "github.com/netgarden/rrpc-def"
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

	tplFuncs := def.GetTemplateFuncMap()
	tplFuncs["createFieldDefaultValue"] = createFieldDefaultValue

	baseTpl, err := template.New("").Funcs(tplFuncs).ParseFS(g.fs, "*.tpl")
	if err != nil {
		return err
	}

	//baseTpl = baseTpl.Funcs(tplFuncs)

	if len(definition.Sources) > 0 {
		err = os.MkdirAll(targetDir, os.ModePerm)
		if err != nil {
			return err
		}
	}

	sourceTpl := baseTpl.Lookup("source.tpl")
	sourceHandlerTpl := baseTpl.Lookup("sourceHandler.tpl")

	for _, source := range definition.Sources {

		if len(source.Structs) == 0 && len(source.Services) == 0 && len(source.Errors) == 0 {
			continue
		}

		name := source.SourceName[:len(source.SourceName)-len(filepath.Ext(source.SourceName))]
		target := filepath.Join(targetDir, name+".gen.go")

		err = g.executeSourceTemplate(sourceTpl, target, source)
		if err != nil {
			return err
		}

		handlerTarget := filepath.Join(targetDir, name+"_handler.gen.go")

		err = g.executeSourceTemplate(sourceHandlerTpl, handlerTarget, source)
		if err != nil {
			return err
		}

	}

	clientTpl := baseTpl.Lookup("rrpcClient.tpl")
	err = g.executeDefinitionTemplate(clientTpl, filepath.Join(targetDir, "rrpc.client.gen.go"), definition)
	if err != nil {
		return err
	}

	serverTpl := baseTpl.Lookup("rrpcModule.tpl")
	err = g.executeDefinitionTemplate(serverTpl, filepath.Join(targetDir, "rrpc.module.gen.go"), definition)
	if err != nil {
		return err
	}

	errorsTpl := baseTpl.Lookup("rrpcErrors.tpl")
	err = g.executeDefinitionTemplate(errorsTpl, filepath.Join(targetDir, "rrpc.errors.gen.go"), definition)
	if err != nil {
		return err
	}

	return nil
}

func (g *Generator) executeSourceTemplate(tpl *template.Template, target string, source *def.Source) error {

	f, err := os.Create(target)
	if err != nil {
		return err
	}
	defer f.Close()

	return tpl.Execute(f, source)
}

func (g *Generator) executeDefinitionTemplate(tpl *template.Template, target string, definition *def.Module) error {

	f, err := os.Create(target)
	if err != nil {
		return err
	}
	defer f.Close()

	return tpl.Execute(f, definition)
}
