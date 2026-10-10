package parser

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"unicode"

	def "github.com/netgarden/rrpc/def"
	"github.com/netgarden/rrpc/def/lexer"
)

func New() *Parser {
	return &Parser{}
}

// identLikeTokens are tokens whose literal text is valid as an identifier
// (struct/service/method/field/error/option name) even though the lexer
// classifies them as structural keywords when they appear at the start of a
// top-level statement (e.g. "name = ..."). The lexer is context-free, so it
// always emits e.g. NAME for the literal "name" regardless of position; the
// parser resolves the ambiguity by accepting these tokens anywhere an IDENT
// is expected in a naming position.
var identLikeTokens = []lexer.Token{
	lexer.IDENT,
	lexer.NAME,
	lexer.VERSION,
	lexer.OPTION,
	lexer.STRUCT,
	lexer.SERVICE,
	lexer.INCLUDE,
	lexer.ERROR,
}

func (p *Parser) readIdent(lex *lexer.Reader) (*lexer.Item, error) {
	return p.readLexItem(lex, identLikeTokens...)
}

type Parser struct {
	currentFile string
}

func (p *Parser) Parse(filePath string) (*def.Module, error) {

	toProcess := make(chan string, 500)
	toProcess <- filePath

	sources := make([]*def.SourceOrigin, 0)

	// closeIfDrained closes toProcess once nothing else is queued behind
	// the item just received — the signal this single-goroutine producer/
	// consumer loop uses to know it's done. Must run on *every* iteration
	// that doesn't itself push more work, not just the "freshly parsed"
	// path below: the same file can legitimately be "include"d from more
	// than one other source in the same module (e.g. two sibling files
	// both including one shared, common definitions file) — when the
	// *second* such include is the last item in the channel, it hits the
	// "already processed" branch, and if that branch doesn't also check
	// here, the loop blocks on `range toProcess` forever waiting for a
	// close that will never come. Confirmed by hitting this exact hang
	// with two citadel .rrpc files both including maf/datatables' shared
	// datatable.rrpc.
	closeIfDrained := func() {
		if len(toProcess) == 0 {
			close(toProcess)
		}
	}

	for sourcePath := range toProcess {

		processed := false
		for _, source := range sources {
			if source.SourcePath == sourcePath {
				processed = true
				break
			}
		}

		if processed {
			closeIfDrained()
			continue
		}

		p.currentFile = sourcePath
		source, err := p.parseFile(sourcePath)
		p.currentFile = ""
		if err != nil {
			return nil, err
		}

		baseDir := filepath.Dir(sourcePath)
		for _, include := range source.Includes {
			includePath := filepath.Join(baseDir, include)
			includePath = filepath.Clean(includePath)
			toProcess <- includePath
		}

		closeIfDrained()

		sources = append(sources, source)

	}

	processor := NewProcessor()
	definition, err := processor.Process(sources)
	if err != nil {
		return nil, err
	}

	validator := NewValidator()
	err = validator.Validate(definition)
	if err != nil {
		return nil, err
	}

	return definition, nil
}

func (p *Parser) parseFile(filePath string) (*def.SourceOrigin, error) {

	f, err := os.OpenFile(filePath, os.O_RDONLY, 0644)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	// SourceName is the file's own basename, not a path relative to the
	// entry file's directory — an `include` can point anywhere on disk
	// (e.g. a sibling repo via "../../../other/def/x.rrpc"), and
	// gen/go/generator.go joins this name directly into its output path
	// to derive each source's own "<name>.gen.go" file. A relative path
	// containing ".." segments would silently resolve outside targetDir
	// once joined — confirmed by an actual cross-repo include landing its
	// generated file inside the *included* file's own directory instead
	// of the consuming module's target directory.
	source := def.NewSourceOrigin(filePath, filepath.Base(filePath))

	lex := lexer.New(f)
	r := lexer.NewReader(lex)

eof:
	for {

		annotations, err := p.readAnnotations(r)
		if err != nil {
			return nil, err
		}

		item := r.Read()
		if item.Token == lexer.EOF {
			break
		}

		if len(annotations) > 0 && item.Token != lexer.SERVICE {
			return nil, fmt.Errorf("unexpected token '%v' found after annotations, %v", item.Token, item.Position)
		}

		switch item.Token {
		case lexer.NEWLINE:
			continue
		case lexer.NAME:
			err = p.parseName(r, source)
			break
		case lexer.VERSION:
			err = p.parseVersion(r, source)
			break
		case lexer.OPTION:
			err = p.parseOption(r, source)
			break
		case lexer.STRUCT:
			err = p.parseStruct(r, source)
			break
		case lexer.SERVICE:
			err = p.parseService(r, source, annotations)
			break
		case lexer.ERROR:
			err = p.parseError(r, source)
			break
		case lexer.INCLUDE:
			err = p.parseInclude(r, source)
			break
		case lexer.EOF:
			break eof
		default:
			return nil, fmt.Errorf("unexpected token %q, %v", item.Token.String(), item.Position)
		}
		if err != nil {
			return nil, err
		}

	}

	return source, nil
}

func (p *Parser) parseName(lex *lexer.Reader, source *def.SourceOrigin) error {

	ret, err := p.parseSetting(lex, source)
	if err != nil {
		return err
	}

	source.Name = ret

	return nil
}

func (p *Parser) parseVersion(lex *lexer.Reader, source *def.SourceOrigin) error {

	ret, err := p.parseSetting(lex, source)
	if err != nil {
		return err
	}

	source.Version = ret

	return nil
}

func (p *Parser) parseSetting(lex *lexer.Reader, source *def.SourceOrigin) (string, error) {

	var item *lexer.Item
	var err error

	_, err = p.readLexItem(lex, lexer.WHITESPACE)
	if err != nil {
		return "", err
	}

	_, err = p.readLexItem(lex, lexer.EQ)
	if err != nil {
		return "", err
	}

	_, err = p.readLexItem(lex, lexer.WHITESPACE)
	if err != nil {
		return "", err
	}

	item, err = p.readLexItem(lex, lexer.STRING)
	if err != nil {
		return "", err
	}

	return item.Literal, nil
}

func (p *Parser) parseOption(lex *lexer.Reader, source *def.SourceOrigin) error {

	var item *lexer.Item
	var err error

	_, err = p.readLexItem(lex, lexer.WHITESPACE)
	if err != nil {
		return err
	}

	item, err = p.readIdent(lex)
	if err != nil {
		return err
	}

	optionName := item.Literal

	p.skipLexItem(lex, lexer.WHITESPACE)

	_, err = p.readLexItem(lex, lexer.EQ)
	if err != nil {
		return err
	}

	p.skipLexItem(lex, lexer.WHITESPACE)

	item, err = p.readLexItem(lex, lexer.STRING)
	if err != nil {
		return err
	}

	optionValue := item.Literal

	option := def.NewOption(optionName, optionValue)

	source.Options = append(source.Options, option)

	return nil
}

func (p *Parser) parseStruct(lex *lexer.Reader, source *def.SourceOrigin) error {

	var item *lexer.Item
	var err error

	_, err = p.readLexItem(lex, lexer.WHITESPACE)
	if err != nil {
		return err
	}

	item, err = p.readIdent(lex)
	if err != nil {
		return err
	}

	structName := item.Literal
	if !unicode.IsLetter(rune(item.Literal[0])) || !unicode.IsUpper(rune(item.Literal[0])) {
		return fmt.Errorf("struct name must start with uppercase letter, got %q", item.Literal)
	}

	p.skipLexItem(lex, lexer.WHITESPACE, lexer.NEWLINE)

	_, err = p.readLexItem(lex, lexer.BCO)
	if err != nil {
		return err
	}

	structDef := def.NewStruct(structName)
	fieldNames := make([]string, 0)

	for {

		if p.readBCC(lex) {
			break
		}

		if len(structDef.Fields) > 0 {
			_, err = p.readLexItem(lex, lexer.NEWLINE)
			if err != nil {
				return err
			}
		}

		// Field annotations (@optional) may sit on the same line as the
		// field or on the lines above it.
		p.skipLexItem(lex, lexer.WHITESPACE, lexer.NEWLINE)
		fieldAnnotations, err := p.readAnnotations(lex)
		if err != nil {
			return err
		}

		var field *def.Field

		field, err = p.parseField(lex, true)
		if err != nil {
			return err
		}

		if field == nil {
			break
		}

		if err = applyFieldAnnotations(field, fieldAnnotations); err != nil {
			return err
		}

		if slices.Contains(fieldNames, field.Name) {
			return fmt.Errorf("field '%s' already defined", field.Name)
		}

		structDef.Fields = append(structDef.Fields, field)
		fieldNames = append(fieldNames, field.Name)

	}

	source.Structs = append(source.Structs, structDef)

	return nil
}

// applyFieldAnnotations sets field flags from its annotations. Unknown
// annotations are rejected rather than silently ignored, so a typo such as
// "@optonal" can't quietly turn into a required field.
func applyFieldAnnotations(field *def.Field, annotations []*def.Annotation) error {
	for _, a := range annotations {
		switch a.Name {
		case "optional":
			if a.Value != "" {
				return fmt.Errorf("field '%s': @optional takes no arguments", field.Name)
			}
			if field.Optional {
				return fmt.Errorf("field '%s': @optional given more than once", field.Name)
			}
			field.Optional = true
		default:
			return fmt.Errorf("field '%s': unknown field annotation @%s (supported: @optional)", field.Name, a.Name)
		}
	}
	return nil
}

func (p *Parser) parseService(lex *lexer.Reader, source *def.SourceOrigin, annotations []*def.Annotation) error {

	var item *lexer.Item
	var err error

	_, err = p.readLexItem(lex, lexer.WHITESPACE)
	if err != nil {
		return err
	}

	item, err = p.readIdent(lex)
	if err != nil {
		return err
	}

	serviceName := item.Literal
	if !unicode.IsLetter(rune(item.Literal[0])) {
		return fmt.Errorf("service name must start with a letter, got %q", item.Literal)
	}

	p.skipLexItem(lex, lexer.WHITESPACE, lexer.NEWLINE)

	_, err = p.readLexItem(lex, lexer.BCO)
	if err != nil {
		return err
	}

	service := def.NewService(serviceName)
	service.Annotations = annotations

	methodNames := make([]string, 0)

	for {

		if p.readBCC(lex) {
			break
		}

		if len(service.Methods) > 0 {
			_, err = p.readLexItem(lex, lexer.NEWLINE)
			if err != nil {
				return err
			}
		} else {
			p.skipLexItem(lex, lexer.WHITESPACE, lexer.NEWLINE)
		}

		var method *def.Method

		method, err = p.parseServiceMethod(lex)
		if err != nil {
			return err
		}

		if method == nil {
			break
		}

		if slices.Contains(methodNames, method.Name) {
			return fmt.Errorf("method '%s' already defined", method.Name)
		}

		service.Methods = append(service.Methods, method)
		methodNames = append(methodNames, method.Name)

	}

	source.Services = append(source.Services, service)

	return nil
}

func (p *Parser) parseServiceMethod(lex *lexer.Reader) (*def.Method, error) {

	var item *lexer.Item
	var err error

	annotations, err := p.readAnnotations(lex)

	item, err = p.readIdent(lex)
	if err != nil {
		return nil, err
	}

	methodName := item.Literal
	if !unicode.IsLetter(rune(methodName[0])) {
		return nil, fmt.Errorf("method name must start with a letter, got %q", methodName)
	}

	method := def.NewMethod(methodName)
	method.Annotations = annotations

	p.skipLexItem(lex, lexer.WHITESPACE)

	_, err = p.readLexItem(lex, lexer.BO)
	if err != nil {
		return nil, err
	}

	inputs, err := p.parseServiceMethodParams(lex, true)
	if err != nil {
		return nil, err
	}
	method.Inputs = inputs

	p.skipLexItem(lex, lexer.WHITESPACE)

	item = lex.Read()
	if item.Token == lexer.IDENT {
		method.Outputs = []*def.Field{
			def.NewField("", item.Literal),
		}
		return method, nil
	} else if item.Token != lexer.BO {
		lex.Unread()
		return method, nil
	}

	outputs, err := p.parseServiceMethodParams(lex, false)
	if err != nil {
		return nil, err
	}
	method.Outputs = outputs

	return method, nil
}

func (p *Parser) parseServiceMethodParams(lex *lexer.Reader, input bool) ([]*def.Field, error) {

	var err error

	ret := make([]*def.Field, 0)
	fieldNames := make([]string, 0)

	for {

		if p.readBC(lex) {
			break
		}

		if len(ret) > 0 {

			p.skipLexItem(lex, lexer.WHITESPACE)

			_, err = p.readLexItem(lex, lexer.COMMA)
			if err != nil {
				return nil, err
			}

			_, err = p.readLexItem(lex, lexer.WHITESPACE)
			if err != nil {
				return nil, err
			}
		}

		var field *def.Field

		field, err = p.parseField(lex, input)
		if err != nil {
			return nil, err
		}

		if field == nil {
			break
		}

		if slices.Contains(fieldNames, field.Name) {
			return nil, fmt.Errorf("field '%s' already defined", field.Name)
		}

		ret = append(ret, field)
		fieldNames = append(fieldNames, field.Name)

	}

	return ret, nil
}

func (p *Parser) parseError(lex *lexer.Reader, source *def.SourceOrigin) error {

	var item *lexer.Item
	var err error

	_, err = p.readLexItem(lex, lexer.WHITESPACE)
	if err != nil {
		return err
	}

	item, err = p.readIdent(lex)
	if err != nil {
		return err
	}

	errorName := item.Literal

	p.skipLexItem(lex, lexer.WHITESPACE, lexer.NEWLINE)

	_, err = p.readLexItem(lex, lexer.BCO)
	if err != nil {
		return err
	}

	errorDef := def.NewError(errorName)

	for {

		if p.readBCC(lex) {
			break
		}

		p.skipLexItem(lex, lexer.WHITESPACE, lexer.NEWLINE)

		item, err = p.readIdent(lex)
		if err != nil {
			return err
		}

		fieldName := item.Literal

		p.skipLexItem(lex, lexer.WHITESPACE)

		_, err = p.readLexItem(lex, lexer.EQ)
		if err != nil {
			return err
		}

		p.skipLexItem(lex, lexer.WHITESPACE)

		item, err = p.readLexItem(lex, lexer.STRING, lexer.IDENT)
		if err != nil {
			return err
		}

		switch fieldName {
		case "message":
			errorDef.Message = item.Literal
		case "code":
			var v uint64
			_, err = fmt.Sscanf(item.Literal, "%d", &v)
			if err != nil {
				return fmt.Errorf("invalid code value %q: %w", item.Literal, err)
			}
			errorDef.Code = v
		case "http_code":
			var v uint64
			_, err = fmt.Sscanf(item.Literal, "%d", &v)
			if err != nil {
				return fmt.Errorf("invalid http_code value %q: %w", item.Literal, err)
			}
			errorDef.HttpCode = v
		}

	}

	source.Errors = append(source.Errors, errorDef)

	return nil
}

func (p *Parser) parseInclude(lex *lexer.Reader, source *def.SourceOrigin) error {

	var item *lexer.Item
	var err error

	_, err = p.readLexItem(lex, lexer.WHITESPACE, lexer.NEWLINE)
	if err != nil {
		return err
	}

	item, err = p.readLexItem(lex, lexer.STRING)
	if err != nil {
		return err
	}

	if !slices.Contains(source.Includes, item.Literal) {
		source.Includes = append(source.Includes, item.Literal)
	}

	return nil
}

func (p *Parser) parseField(lex *lexer.Reader, withName bool) (*def.Field, error) {

	var item *lexer.Item
	var err error

	p.skipLexItem(lex, lexer.WHITESPACE, lexer.NEWLINE)

	fieldName := ""
	if withName {

		item, err = p.readIdent(lex)
		if err != nil {
			return nil, err
		}

		fieldName = item.Literal
		if !unicode.IsLetter(rune(fieldName[0])) {
			return nil, fmt.Errorf("field name must start with a letter, got %q", fieldName)
		}

		_, err = p.readLexItem(lex, lexer.WHITESPACE)
		if err != nil {
			return nil, err
		}

	}

	item, err = p.readLexItem(lex, lexer.IDENT)
	if err != nil {
		return nil, err
	}

	fieldType := item.Literal

	return def.NewField(fieldName, fieldType), nil
}

// canonicalAnnotationName maps a spelling found in a .rrpc file to the name the rest of the code uses.
// @Path is the old spelling of @path (annotations are lower case); both are accepted, so existing
// definitions keep working, and only "path" is seen after parsing.
func canonicalAnnotationName(name string) string {
	if name == "Path" {
		return "path"
	}
	return name
}

func (p *Parser) readAnnotations(lex *lexer.Reader) ([]*def.Annotation, error) {

	ret := make([]*def.Annotation, 0)

	for {

		annotation, err := p.readAnnotation(lex)
		if err != nil {
			return nil, err
		}

		if annotation == nil {
			break
		}

		ret = append(ret, annotation)

	}

	return ret, nil
}

func (p *Parser) readAnnotation(lex *lexer.Reader) (*def.Annotation, error) {

	var item *lexer.Item
	var err error

	item = lex.Read()
	if item.Token != lexer.ANNOTATION {
		lex.Unread()
		return nil, nil
	}

	annotation := def.NewAnnotation(canonicalAnnotationName(item.Literal))

	if p.skipLexItem(lex, lexer.BO) {

		p.skipLexItem(lex, lexer.WHITESPACE, lexer.NEWLINE)

		if !p.skipLexItem(lex, lexer.BC) {

			item, err = p.readLexItem(lex, lexer.STRING)
			if err != nil {
				return nil, err
			}

			value := item.Literal

			p.skipLexItem(lex, lexer.WHITESPACE, lexer.NEWLINE)

			_, err = p.readLexItem(lex, lexer.BC)
			if err != nil {
				return nil, err
			}

			annotation.Value = value

		}

	}

	p.skipLexItem(lex, lexer.WHITESPACE, lexer.NEWLINE)

	return annotation, nil
}

func (p *Parser) readBC(lex *lexer.Reader) bool {
	return p.readEnding(lex, lexer.BC)
}

func (p *Parser) readBCC(lex *lexer.Reader) bool {
	return p.readEnding(lex, lexer.BCC)
}

func (p *Parser) readEnding(lex *lexer.Reader, endingToken lexer.Token) bool {

	var item *lexer.Item

	skippedWS := p.skipLexItem(lex, lexer.WHITESPACE, lexer.NEWLINE)

	item = lex.Read()
	if item.Token == endingToken {
		return true
	}

	if skippedWS {
		lex.Unread()
	}
	lex.Unread()

	return false
}

func (p *Parser) readLexItem(lex *lexer.Reader, tokens ...lexer.Token) (*lexer.Item, error) {

	item := lex.Read()

	if len(tokens) >= 0 && !slices.Contains(tokens, item.Token) {

		var tokenStr string
		if len(tokens) == 1 {
			tokenStr = tokens[0].String()
		} else if len(tokens) > 1 {
			tokenStr = fmt.Sprintf("%v", tokens)
		}

		err := fmt.Errorf(
			"expected %s, got %s, file %s, %v",
			tokenStr,
			item.Token.String(),
			p.currentFile,
			item.Position,
		)

		return item, err
	}

	return item, nil
}

func (p *Parser) skipLexItem(lex *lexer.Reader, tokens ...lexer.Token) bool {
	item := lex.Read()
	if len(tokens) >= 0 && !slices.Contains(tokens, item.Token) {
		lex.Unread()
		return false
	}
	return true
}
