package rrpc_def

func NewModule() *Module {
	return &Module{
		Sources:  make(map[string]*Source),
		Options:  make(map[string]string),
		Structs:  make(map[string]*Struct),
		Services: make(map[string]*Service),
		Errors:   make(map[string]*Error),
		Includes: make([]string, 0),
	}
}

type Module struct {
	MainSource *Source
	Sources    map[string]*Source
	Name       string
	Version    string
	Options    map[string]string
	Structs    map[string]*Struct
	Services   map[string]*Service
	Errors     map[string]*Error
	Includes   []string
}

func NewSourceOrigin(path string, name string) *SourceOrigin {
	//name := path[len(filepath.Base(path))+1:]
	return &SourceOrigin{
		SourceName: name,
		SourcePath: path,
		Options:    make([]*Option, 0),
		Structs:    make([]*Struct, 0),
		Services:   make([]*Service, 0),
		Errors:     make([]*Error, 0),
		Includes:   make([]string, 0),
	}
}

type SourceOrigin struct {
	SourceName string
	SourcePath string
	Name       string
	Version    string
	Options    []*Option
	Structs    []*Struct
	Services   []*Service
	Errors     []*Error
	Includes   []string
}

func NewSource(module *Module, name string, filePath string) *Source {
	//name := filePath[len(filepath.Base(filePath))+1:]
	return &Source{
		Module:     module,
		SourceName: name,
		SourcePath: filePath,
		Options:    make(map[string]string),
		Structs:    make(map[string]*Struct),
		Services:   make(map[string]*Service),
		Errors:     make(map[string]*Error),
		Includes:   make([]string, 0),
	}
}

type Source struct {
	Module     *Module
	SourceName string
	SourcePath string
	Name       string
	Version    string
	Options    map[string]string
	Structs    map[string]*Struct
	Services   map[string]*Service
	Errors     map[string]*Error
	Includes   []string
}

func NewStruct(name string) *Struct {
	return &Struct{
		Name:   name,
		Fields: make([]*Field, 0),
	}
}

type Struct struct {
	Name   string
	Fields []*Field
	Source *Source
}

func NewField(name string, t string) *Field {
	return &Field{
		Name: name,
		Type: t,
	}
}

type Field struct {
	Name     string
	Type     string
	IsStruct bool
	// Optional is set by the @optional field annotation: the field may be
	// absent on the wire. Generators represent that as a Go pointer (or a
	// nil slice/map) with omitempty, and as `name?: T` in TypeScript.
	Optional bool
}

func NewAnnotation(name string) *Annotation {
	return &Annotation{
		Name: name,
	}
}

type Annotation struct {
	Name  string
	Value string
}

func NewService(name string) *Service {
	return &Service{
		Name:        name,
		Methods:     make([]*Method, 0),
		Annotations: []*Annotation{},
	}
}

type Service struct {
	Name        string
	Path        string
	Methods     []*Method
	Annotations []*Annotation
	Source      *Source
}

func NewMethod(name string) *Method {
	return &Method{
		Name:        name,
		Type:        "POST",
		Inputs:      make([]*Field, 0),
		Outputs:     make([]*Field, 0),
		Annotations: make([]*Annotation, 0),
	}
}

type Method struct {
	Name string
	Path string
	Type string
	// ContentType is the whole request's Content-Type: auto-inferred from
	// Inputs (see the processor), or overridden by @RequestContentType for a
	// method with a single bytes input (no accompanying struct - that's
	// multipart, see BytesPartContentType instead). For a multipart method
	// this always stays "multipart/form-data" (the boundary is added at
	// send time); @RequestContentType never touches it in that case.
	ContentType string
	// ResponseContentType is the default Content-Type the server sets on a
	// bytes response, auto-inferred as "application/octet-stream" for a
	// bytes output or overridden by @ResponseContentType. Baked into
	// generated code as a literal; not part of the runtime Method/
	// MethodHolder struct (see rrpc/server.go), since nothing needs to read
	// it after codegen.
	ResponseContentType string
	// BytesPartContentType is the multipart bytes field's own part-level
	// Content-Type (distinct from ContentType, the envelope) - auto-inferred
	// as "application/octet-stream" (matching multipart.Writer.
	// CreateFormFile's default) for a method combining a struct input with a
	// bytes input, or overridden by @RequestContentType on such a method.
	// Empty/unused for a non-multipart method.
	BytesPartContentType string
	Inputs               []*Field
	Outputs              []*Field
	Annotations          []*Annotation
	// Streaming and StreamDirection are set by the processor when a
	// method carries @Stream/@InputStream/@OutputStream. StreamDirection
	// is one of "both"/"in"/"out" and is only meaningful when Streaming
	// is true.
	Streaming       bool
	StreamDirection string
}

func NewOption(name string, value string) *Option {
	return &Option{
		Name:  name,
		Value: value,
	}
}

type Option struct {
	Name  string
	Value string
}

func NewError(name string) *Error {
	return &Error{Name: name}
}

type Error struct {
	Name     string
	Message  string
	Code     uint64
	HttpCode uint64
	Source   *Source
}
