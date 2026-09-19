package rrpc_gen_go

import (
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"testing"

	rrpcparser "github.com/netgarden/rrpc/def/parser"
)

func generate(t *testing.T, rrpc string) string {
	t.Helper()
	dir := t.TempDir()
	src := filepath.Join(dir, "module.rrpc")
	if err := os.WriteFile(src, []byte(rrpc), 0o644); err != nil {
		t.Fatal(err)
	}
	mod, err := rrpcparser.New().Parse(src)
	if err != nil {
		t.Fatal(err)
	}
	out := filepath.Join(dir, "out")
	if err := NewGenerator().Generate(mod, out); err != nil {
		t.Fatal(err)
	}
	return out
}

func imports(t *testing.T, file string) map[string]bool {
	t.Helper()
	f, err := parser.ParseFile(token.NewFileSet(), file, nil, parser.ImportsOnly)
	if err != nil {
		t.Fatalf("generated file doesn't parse: %v", err)
	}
	got := map[string]bool{}
	for _, imp := range f.Imports {
		got[imp.Path.Value] = true
	}
	return got
}

const header = "name = \"m\"\nversion = \"0.0.1\"\noption go_package = \"m\"\n\n"

// A service made only of @Stream methods never calls bytes.NewBuffer, so
// importing "bytes" would fail to compile ("imported and not used").
func TestNoBytesImportForStreamOnlyService(t *testing.T) {
	out := generate(t, header+`
struct In { text string }
struct Out { text string }

@Path("api/chat")
service Chat {
    @Stream
    @Path("stream")
    stream(request In) (Out)
}
`)
	if imports(t, filepath.Join(out, "module_handler.gen.go"))[`"bytes"`] {
		t.Error(`stream-only service must not import "bytes"`)
	}
}

// A regular JSON POST does use bytes.NewBuffer for the request body.
func TestBytesImportForStructRequest(t *testing.T) {
	out := generate(t, header+`
struct In { text string }
struct Out { text string }

@Path("api/echo")
service Echo {
    @method("POST")
    say(request In) (Out)
}
`)
	if !imports(t, filepath.Join(out, "module_handler.gen.go"))[`"bytes"`] {
		t.Error(`struct request service must import "bytes"`)
	}
}

// Mixed: one stream method plus one regular method still needs "bytes".
func TestBytesImportForMixedService(t *testing.T) {
	out := generate(t, header+`
struct In { text string }
struct Out { text string }

@Path("api/mix")
service Mix {
    @method("POST")
    say(request In) (Out)

    @Stream
    live(request In) (Out)
}
`)
	if !imports(t, filepath.Join(out, "module_handler.gen.go"))[`"bytes"`] {
		t.Error(`mixed service must import "bytes"`)
	}
}
