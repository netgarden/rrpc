package rrpc_gen_ts

import (
	"io/fs"
	"os"
	"path/filepath"
	"text/template"

	def "github.com/netgarden/rrpc/def"
)

// defaultRRPCImportPath is the import specifier generated code uses to
// reach the RRPC TS runtime (RRPCClient/RRPCResponse/RRPCError/
// escapePathValue) when no WithRRPCImportPath option overrides it —
// matching every consumer's historical assumption that the runtime lives
// at web/src/lib/rrpc.ts.
const defaultRRPCImportPath = "$lib/rrpc"

// Option configures a Generator; see WithRRPCImportPath.
type Option func(*Generator)

// WithRRPCImportPath overrides where generated code imports the RRPC TS
// runtime from — e.g. "$lib/rpc/rrpc" for a consumer that keeps its
// generated clients and the vendored runtime together under a "rpc/"
// subdirectory rather than at the root of its lib directory.
func WithRRPCImportPath(path string) Option {
	return func(g *Generator) {
		if path != "" {
			g.rrpcImportPath = path
		}
	}
}

func NewGenerator(opts ...Option) def.Generator {
	g := &Generator{
		fs:             getTemplatesFS(),
		rrpcImportPath: defaultRRPCImportPath,
	}
	for _, opt := range opts {
		opt(g)
	}
	return g
}

type Generator struct {
	fs             fs.FS
	rrpcImportPath string
}

func (g *Generator) Generate(module *def.Module, targetDir string) error {

	tplFuncs := getTemplatesFuncs()
	tplFuncs["rrpcImportPath"] = func() string { return g.rrpcImportPath }

	baseTpl, err := template.New("").Funcs(tplFuncs).ParseFS(g.fs, "*.tpl")
	if err != nil {
		return err
	}

	mainTpl := baseTpl.Lookup("main.tpl")

	f, err := os.Create(filepath.Join(targetDir, module.Name+".ts"))
	if err != nil {
		return err
	}
	defer f.Close()

	return mainTpl.Execute(f, module)
}
