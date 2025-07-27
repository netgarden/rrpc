package rrpc_def

type Generator interface {
	Generate(definition *Definition, targetDir string) error
}
