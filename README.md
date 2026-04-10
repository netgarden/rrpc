# RRPC

RRPC is a REST RPC framework. Services are defined once in `.rrpc` schema files; type-safe server and client code is generated for Go and TypeScript. The generated code is backed by a lightweight HTTP runtime that handles routing, error serialisation, path/query parameter decoding, and multipart form uploads.

## Repository layout

| Directory     | Go module                        | Purpose                                       |
|---------------|----------------------------------|-----------------------------------------------|
| `rrpc/`       | `github.com/netgarden/rrpc`      | RRPC server runtime (this package)            |
| `rrpc-def/`   | `github.com/netgarden/rrpc-def`  | `.rrpc` file lexer, parser, and AST           |
| `rrpc-gen/`   | `github.com/netgarden/rrpc-gen`  | `rrpc-gen` CLI — orchestrates code generation |
| `rrpc-gen-go/`| `github.com/netgarden/rrpc-gen-go` | Go code generator                             |
| `rrpc-gen-ts/`| `github.com/netgarden/rrpc-gen-ts` | TypeScript code generator                     |
| `rrpc-test/`  | `github.com/netgarden/rrpc-test` | Example application                           |

---

## Definition file format (`.rrpc`)

Each module is described in a `.rrpc` file. Multiple files can be composed with `include`.

### Module header

```
name    = "files"
version = "0.0.1"

option go_package = "files"

include "common.rrpc"
```

`go_package` is required when generating Go code and sets the `package` declaration of the output files.

### Structs

Structs define the input and output shapes of methods. Fields are typed; arrays and optional modifiers are supported.

```
struct LoginRequest {
    username string
    password string
}

struct SearchResult {
    items    []string
    total    int
    hasMore  bool
}

struct Config {
    timeout  int
    tag?     string      # optional field
}
```

**Primitive types**

| Type       | Go type       | TypeScript type |
|------------|---------------|-----------------|
| `string`   | `string`      | `string`        |
| `bool`     | `bool`        | `boolean`       |
| `int`      | `int`         | `number`        |
| `int8/16/32/64` | `int8` … | `number`       |
| `uint` …   | `uint` …      | `number`        |
| `float32/64` | `float32/64`| `number`        |
| `byte`     | `byte`        | `number`        |
| `bytes`    | `io.Reader` / `io.Writer` | `string \| Uint8Array` |

Arrays use `[]` prefix: `[]string`, `[]int`. Maps use `map[KeyType]ValueType`.

### Services

A service groups related methods. Its HTTP base path defaults to the service name and can be overridden with `@Path`.

```
@Path("/auth")
service Auth {
    login(request LoginRequest) (LoginResponse)
    logout()
}
```

### Methods

Methods define the HTTP verb, path, inputs, and output. Defaults: `POST`, path = method name.

```
service Example {

    @method("PUT")
    @Path("/upload/{filename:string}")
    upload(data bytes)

    @method("GET")
    @Path("/download/{filename:string}")
    download() bytes

    @method("POST")
    @Path("/search")
    search(query SearchRequest) (SearchResponse)

    @method("GET")
    @Path("/echo")
    echo(params EchoRequest) (EchoResponse)

}
```

**Method constraints**

| HTTP verb      | Allowed inputs                        |
|----------------|---------------------------------------|
| `POST`, `PUT`  | Single struct, single `bytes`, or struct + `bytes` (multipart) |
| `GET`, `HEAD`, `DELETE` | Single struct (serialised as query params); `bytes` inputs are not allowed |

A single `bytes` output is streamed; the handler receives an `io.Writer` to write into.

**Path parameters** use `{name:type}` syntax:

| Type     | Matches                |
|----------|------------------------|
| `string` | One path segment (no `/`) |
| `path`   | Any characters including `/` |

### Errors

Module-level errors carry a numeric code and an HTTP status. They are returned from service implementations and serialised automatically.

```
error AuthenticationError {
    code      = 1
    http_code = 401
    message   = "authentication failed"
}

error NotFoundError {
    code      = 2
    http_code = 404
    message   = "not found"
}
```

Generated Go: `var ErrAuthenticationError = rrpc.RRPCError{Code: 1, ...}`

---

## Code generation

### CLI — `rrpc-gen`

```bash
# Build
cd rrpc-gen && go build ./cmd/rrpc-gen/

# Generate Go code
rrpc-gen --source path/to/module.rrpc --target ./output/dir --type go

# Generate TypeScript code
rrpc-gen --source path/to/module.rrpc --target ./output/dir --type ts
```

`RRPC_DEVEL=true` makes the generators load templates from the filesystem instead of the embedded FS — useful when editing templates.

### Go output files

For a source file `module.rrpc` the generator writes four files into `--target`:

| File | Contents |
|------|----------|
| `module.gen.go` | Struct types, service interfaces, error variables |
| `module_handler.gen.go` | HTTP server handler and generated Go client |
| `rrpc.module.gen.go` | Module registration, `SetXxxService` methods |
| `rrpc.client.gen.go` | `Client` struct, `NewClient`, `SizedReader/Closer` helpers |

Never edit `*.gen.go` files directly — regenerate them with `rrpc-gen`.

### TypeScript output

One `.gen.ts` file is produced per source. It exports a typed client class. Import it from `$lib/<module>` in SvelteKit.

---

## Go — implementing a service

1. Generate code from the `.rrpc` file.
2. Create an `impl/` subdirectory and implement the generated interface.
3. Register the module in your server setup.

```go
// modules/auth/impl/auth.go
package impl

import (
    "github.com/netgarden/rrpc"
    "github.com/netgarden/rrpc-test/modules/auth"
)

type AuthService struct{}

func (s *AuthService) Login(ctx *rrpc.Context, req *auth.LoginRequest) (*auth.LoginResponse, error) {
    if req.Username != req.Password {
        return nil, auth.ErrAuthenticationError
    }
    return &auth.LoginResponse{Ok: true}, nil
}

func (s *AuthService) Logout(ctx *rrpc.Context) error {
    return nil
}
```

Reading path params:

```go
func (s *FilesService) Upload(ctx *rrpc.Context, r io.Reader) error {
    filename := ctx.Params().GetString("filename")
    // write r to disk as filename…
}
```

Bytes output — write to the provided `io.Writer`:

```go
func (s *FilesService) Download(ctx *rrpc.Context, w io.Writer) error {
    filename := ctx.Params().GetString("filename")
    f, err := os.Open(filepath.Join(s.dir, filename))
    if err != nil { return err }
    defer f.Close()
    _, err = io.Copy(w, f)
    return err
}
```

Multipart (struct + bytes):

```go
func (s *FilesService) Upload2(ctx *rrpc.Context, info *files.UploadDataInfo, r io.Reader) error {
    f, err := os.Create(filepath.Join(s.dir, info.Filename))
    if err != nil { return err }
    defer f.Close()
    _, err = io.Copy(f, r)
    return err
}
```

### Server setup

```go
authModule := auth.NewModule()
authModule.SetAuthService(impl.NewAuthService())

filesModule := files.NewModule()
filesModule.SetFilesService(impl.NewFilesService("uploads/"))

srv := rrpc.NewServer()
srv.RegisterModule(authModule)
srv.RegisterModule(filesModule)

// Optional: authentication / logging hook
srv.SetOnRequest(func(w http.ResponseWriter, r *http.Request) error {
    if r.Header.Get("Authorization") == "" {
        return rrpc.RRPCError{Code: 401, Name: "Unauthorized", Message: "missing token", HTTPStatus: 401}
    }
    return nil
})

http.ListenAndServe(":8080", srv)
```

### Go client usage

```go
client := auth.NewClient("http://localhost:8080")

resp, err := client.Auth.Login(ctx, &auth.LoginRequest{
    Username: "alice",
    Password: "secret",
})

// Files — path param passed as first argument after ctx
err = client.Files.Upload(ctx, "hello.txt", rrpc.NewSizedReader(fileReader, fileSize))

// Download — returns *SizedReadCloser
rc, err := client.Files.Download(ctx, "hello.txt")
defer rc.Close()
io.Copy(dst, rc)

// Multipart
err = client.Files.Upload2(ctx, &files.UploadDataInfo{Filename: "doc.pdf"}, rrpc.NewSizedReader(reader, size))

// GET with query params (struct fields become URL params automatically)
resp, err := client.Echo.Echo(ctx, &echo.EchoRequest{Message: "hi", Count: 3, Flag: true})
```

---

## TypeScript / SvelteKit client

Generated `.gen.ts` files expose typed client classes. Use them from a `+page.svelte` or any `.ts` file.

```typescript
import { FilesClient } from '$lib/files';
import { AuthClient } from '$lib/auth';
import { EchoClient } from '$lib/echo';

const auth  = new AuthClient('');      // empty = same origin
const files = new FilesClient('');
const echo  = new EchoClient('');

// POST with JSON body
const res = await auth.authService.login({ username: 'alice', password: 'secret' });

// PUT with raw bytes
const buf = await file.arrayBuffer();
await files.filesService.upload('hello.txt', new Uint8Array(buf));

// POST multipart (struct + bytes)
await files.filesService.upload2({ filename: 'doc.pdf' }, new Uint8Array(buf));

// GET with query params
const result = await echo.echoService.echo({ message: 'hi', count: 3, flag: true });

// GET with path param
const bytes = await files.filesService.download('hello.txt');
const text  = new TextDecoder().decode(bytes);
```

Errors thrown by the client are instances of `RRPCError`:

```typescript
import { RRPCError } from '$lib/rrpc';

try {
    await auth.authService.login({ username: 'x', password: 'y' });
} catch (e) {
    if (e instanceof RRPCError) {
        console.log(e.code, e.error, e.message, e.cause);
    }
}
```

---

## Error wire format

All errors are returned as `application/problem+json`:

```json
{
  "error": "AuthenticationError",
  "code": 1,
  "msg": "authentication failed",
  "cause": "passwords do not match"
}
```

The HTTP status code is set from the error's `http_code` (module errors) or the framework's built-in codes.

### Framework error codes

| Code | HTTP | Name                    | Meaning |
|------|------|-------------------------|---------|
|  0   | 400  | `RrpcEndpoint`          | Handler returned a non-`RRPCError` |
| -1   | 400  | `RrpcRequestFailed`     | HTTP request failed (client) |
| -2   | 404  | `RrpcBadRoute`          | No handler matched the path |
| -3   | 405  | `RrpcBadMethod`         | Wrong HTTP verb |
| -4   | 400  | `RrpcBadRequest`        | Malformed request body or params |
| -5   | 500  | `RrpcBadResponse`       | Failed to serialise the response |
| -6   | 500  | `RrpcServerPanic`       | Unhandled panic in a handler |
| -7   | 500  | `RrpcInternalError`     | Internal server error |
| -8   | 400  | `RrpcClientDisconnected`| Client disconnected mid-stream |

---

## Request body encoding by method type

| Inputs                  | HTTP verb    | Encoding |
|-------------------------|--------------|----------|
| None                    | any          | No body |
| Single struct           | POST / PUT   | JSON body (`application/json`) |
| Single `bytes`          | POST / PUT   | Raw binary body (`application/octet-stream`) |
| Struct + `bytes`        | POST / PUT   | Multipart form data (`multipart/form-data`); struct as named JSON field, bytes as file field |
| Single struct           | GET / HEAD / DELETE | URL query parameters; all struct fields must be primitive types |

---

## Static file serving

```go
// In service implementation
handler := rrpc.NewStaticFilesHandler("filepath", os.DirFS("."), "web/build")

func (s *StaticService) Serve(ctx *rrpc.Context, w io.Writer) error {
    return handler.ServeStatic(ctx, w)
}
```

`ServeStatic` sets `Content-Type` from the file extension and returns `404` for paths outside the base directory.

---

## Running tests

```bash
# rrpc runtime unit tests (uses generated testmodule)
cd rrpc && make test

# rrpc-def parser tests
cd rrpc-def && go test ./...

# rrpc-test integration (start the server)
cd rrpc-test && go run ./cmd/server/

# rrpc-test web frontend
cd rrpc-test/web && npm run dev
```