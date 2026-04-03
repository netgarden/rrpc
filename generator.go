package rrpc_def

type Generator interface {
	Generate(definition *Module, targetDir string) error
}
