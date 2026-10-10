package parser

import (
	"path/filepath"
	"testing"
)

// @path is the annotation; @Path is its old spelling and must keep working, on a service and on a method.
func TestParse_PathAnnotationBothSpellings(t *testing.T) {
	dir := t.TempDir()
	mustWrite(t, filepath.Join(dir, "module.rrpc"), `name = "test"
version = "0.0.1"
option go_package = "test"

@path("/lower")
service Lower {
    @method("GET")
    @path("/a/{id:string}")
    a() (string)

    @method("GET")
    @Path("/b")
    b() (string)

    @method("GET")
    c() (string)
}

@Path("/upper")
service Upper {
    @method("GET")
    @path("/d")
    d() (string)
}
`)
	mod, err := New().Parse(filepath.Join(dir, "module.rrpc"))
	if err != nil {
		t.Fatal(err)
	}
	services := map[string]string{}
	methods := map[string]string{}
	for _, s := range mod.Services {
		services[s.Name] = s.Path
		for _, m := range s.Methods {
			methods[m.Name] = m.Path
		}
	}
	for name, want := range map[string]string{"Lower": "/lower", "Upper": "/upper"} {
		if services[name] != want {
			t.Errorf("service %s path = %q, want %q", name, services[name], want)
		}
	}
	for name, want := range map[string]string{"a": "/a/{id:string}", "b": "/b", "c": "c", "d": "/d"} {
		if methods[name] != want {
			t.Errorf("method %s path = %q, want %q", name, methods[name], want)
		}
	}
}
