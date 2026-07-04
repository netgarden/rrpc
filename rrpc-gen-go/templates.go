package rrpc_gen_go

import (
	"embed"
	"io/fs"
	"os"
)

//go:embed templates/*
var templates embed.FS

func getTemplatesFS() fs.FS {

	devel := os.Getenv("RRPC_DEVEL")
	if devel == "true" {
		return os.DirFS("../rrpc-gen-go/templates")
	}

	return templates
}
