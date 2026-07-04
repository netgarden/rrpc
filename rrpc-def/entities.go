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
	Name        string
	Path        string
	Type        string
	ContentType string
	Inputs      []*Field
	Outputs     []*Field
	Annotations []*Annotation
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
