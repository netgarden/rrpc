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

	// The embed root contains the "templates" directory itself; the
	// generators glob "*.tpl" at the FS root, so hand them the subtree.
	sub, err := fs.Sub(templates, "templates")
	if err != nil {
		panic(err) // the //go:embed pattern above guarantees the directory exists
	}
	return sub
}
