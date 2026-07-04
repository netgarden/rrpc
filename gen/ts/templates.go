package rrpc_gen_ts

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
		return os.DirFS("./ts/templates")
	}

	return templates
}
