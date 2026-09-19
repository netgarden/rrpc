package rrpc_gen_go

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// clientMethod returns the source of one generated client method
// (from its signature to the next top-level func).
func clientMethod(t *testing.T, out, service, method string) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(out, "module_handler.gen.go"))
	if err != nil {
		t.Fatal(err)
	}
	src := string(b)
	sig := "func (h *" + service + "ServiceClientHandler) " + method + "("
	i := strings.Index(src, sig)
	if i < 0 {
		t.Fatalf("client method %s.%s not found", service, method)
	}
	rest := src[i:]
	if j := strings.Index(rest[1:], "\nfunc "); j >= 0 {
		rest = rest[:j+1]
	}
	return rest
}

// A GET's request struct must travel in the query string (what the server's
// DecodeQueryParams and the TS client use), never in a body the server ignores.
func TestGetClientSendsQueryStringNotBody(t *testing.T) {
	out := generate(t, header+`
struct In { q string }
struct Out { q string }

@Path("api/s")
service S {
    @method("GET")
    find(request In) (Out)

    @method("POST")
    put(request In) (Out)
}
`)

	get := clientMethod(t, out, "S", "Find")
	if !strings.Contains(get, "rrpc.EncodeQueryParams(data)") || !strings.Contains(get, `url += "?" + query.Encode()`) {
		t.Errorf("GET client must encode a query string:\n%s", get)
	}
	if strings.Contains(get, "json.Marshal") || strings.Contains(get, "bytes.NewBuffer") {
		t.Errorf("GET client must not build a JSON body:\n%s", get)
	}
	if !strings.Contains(get, `"application/json",nil)`) && !strings.Contains(get, `"application/json", nil)`) {
		t.Errorf("GET client must pass a nil body:\n%s", get)
	}

	// Other verbs are unchanged: POST still sends JSON.
	post := clientMethod(t, out, "S", "Put")
	if !strings.Contains(post, "json.Marshal(data)") || !strings.Contains(post, "bytes.NewBuffer(jsonBody)") {
		t.Errorf("POST client must still send a JSON body:\n%s", post)
	}
	if strings.Contains(post, "EncodeQueryParams") {
		t.Errorf("POST client must not use query params:\n%s", post)
	}
}

// With only GET methods nothing uses bytes.NewBuffer, so importing "bytes"
// would not compile ("imported and not used").
func TestNoBytesImportForGetOnlyService(t *testing.T) {
	out := generate(t, header+`
struct In { q string }
struct Out { q string }

@Path("api/s")
service S {
    @method("GET")
    find(request In) (Out)
}
`)
	if imports(t, filepath.Join(out, "module_handler.gen.go"))[`"bytes"`] {
		t.Error(`GET-only service must not import "bytes"`)
	}
}
