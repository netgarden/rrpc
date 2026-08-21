package rrpc_gen_go

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"text/template"

	def "github.com/netgarden/rrpc/def"
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
	tplFuncs["methodPathParams"] = methodPathParams
	tplFuncs["methodPathToGo"] = methodPathToGo
	tplFuncs["methodPathHasStringParam"] = methodPathHasStringParam
	tplFuncs["streamGoType"] = streamGoType
	tplFuncs["streamUpgradeCall"] = streamUpgradeCall
	tplFuncs["streamClientGoType"] = streamClientGoType
	tplFuncs["streamDialCall"] = streamDialCall

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
		if name != filepath.Base(name) {
			// Defense in depth: SourceName is expected to be a bare
			// filename (see def/parser/parser.go), but if it ever again
			// carried a directory component (e.g. "../" from a path
			// relative to some other directory), joining it into target
			// below would silently write outside targetDir entirely —
			// exactly the bug that once sent a cross-repo include's
			// generated file into the *included* file's own directory
			// instead of this module's.
			return fmt.Errorf("rrpc-gen: source name %q must be a bare filename, not a path", source.SourceName)
		}
		target := filepath.Join(targetDir, name+".gen.go")

		err = g.executeSourceTemplate(sourceTpl, target, source)
		if err != nil {
			return err
		}

		// sourceHandler.tpl unconditionally imports context/json/io/http/
		// strconv/rrpc for its service-handling boilerplate — only
		// meaningful (and only compiles without "imported and not used")
		// when the source actually declares a service. A source with only
		// structs (e.g. one meant purely to be "include"d for its types,
		// like maf/datatables' own datatable.rrpc) has nothing for a
		// handler file to do.
		if len(source.Services) > 0 {
			handlerTarget := filepath.Join(targetDir, name+"_handler.gen.go")

			err = g.executeSourceTemplate(sourceHandlerTpl, handlerTarget, source)
			if err != nil {
				return err
			}
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
