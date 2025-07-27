package parser

import (
	"fmt"
	def "github.com/netgarden/rrpc-def"
	"github.com/netgarden/rrpc-def/lexer"
	"os"
	"slices"
	"unicode"
)

func New() *Parser {
	return &Parser{}
}

type Parser struct {
	queue []*lexer.Item
}

func (p *Parser) Parse(filePath string) (*def.Definition, error) {

	toProcess := make(chan string, 500)
	toProcess <- filePath

	sources := make([]*def.SourceOrigin, 0)

	for sourcePath := range toProcess {

		processed := false
		for _, source := range sources {
			if source.Path == sourcePath {
				processed = true
				break
			}
		}

		if processed {
			continue
		}

		source, err := p.parseFile(filePath)
		if err != nil {
			return nil, err
		}

		for _, include := range source.Includes {
			toProcess <- include
		}

		sources = append(sources, source)

	}

	processor := NewProcessor()
	return processor.Process(sources)
}

func (p *Parser) parseFile(filePath string) (*def.SourceOrigin, error) {

	f, err := os.OpenFile(filePath, os.O_RDONLY, 0644)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	source := def.NewSourceOrigin(filePath)

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

func (p *Parser) parseVersion(lex *lexer.Reader, source *def.SourceOrigin) error {

	var item *lexer.Item
	var err error

	_, err = p.readLexItem(lex, lexer.WHITESPACE)
	if err != nil {
		return err
	}

	item, err = p.readLexItem(lex, lexer.STRING)
	if err != nil {
		return err
	}

	source.Version = item.Literal

	return nil
}

func (p *Parser) parseOption(lex *lexer.Reader, source *def.SourceOrigin) error {

	var item *lexer.Item
	var err error

	_, err = p.readLexItem(lex, lexer.WHITESPACE)
	if err != nil {
		return err
	}

	item, err = p.readLexItem(lex, lexer.IDENT)
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

	item, err = p.readLexItem(lex, lexer.IDENT)
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

		var field *def.Field

		field, err = p.parseField(lex)
		if err != nil {
			return err
		}

		if field == nil {
			break
		}

		if slices.Contains(fieldNames, field.Name) {
			return fmt.Errorf("field '%s' already defined", field.Name)
		}

		structDef.Fields = append(structDef.Fields, field)
		fieldNames = append(fieldNames, field.Name)

		p.skipLexItem(lex, lexer.NEWLINE)

	}

	source.Structs = append(source.Structs, structDef)

	return nil
}

func (p *Parser) parseService(lex *lexer.Reader, source *def.SourceOrigin, annotations []*def.Annotation) error {

	var item *lexer.Item
	var err error

	_, err = p.readLexItem(lex, lexer.WHITESPACE)
	if err != nil {
		return err
	}

	item, err = p.readLexItem(lex, lexer.IDENT)
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

	item, err = p.readLexItem(lex, lexer.IDENT)
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

	inputs, err := p.parseServiceMethodParams(lex)
	if err != nil {
		return nil, err
	}
	method.Inputs = inputs

	item = lex.Read()
	if item.Token != lexer.BO {
		lex.Unread()
		return method, nil
	}

	outputs, err := p.parseServiceMethodParams(lex)
	if err != nil {
		return nil, err
	}
	method.Outputs = outputs

	return method, nil
}

func (p *Parser) parseServiceMethodParams(lex *lexer.Reader) ([]*def.Field, error) {

	var err error

	ret := make([]*def.Field, 0)
	fieldNames := make([]string, 0)

	for {

		if p.readBC(lex) {
			break
		}

		if len(ret) > 0 {

			p.skipLexItem(lex, lexer.WHITESPACE, lexer.NEWLINE)

			_, err = p.readLexItem(lex, lexer.COMMA)
			if err != nil {
				return nil, err
			}

			_, err = p.readLexItem(lex, lexer.WHITESPACE, lexer.NEWLINE)
			if err != nil {
				return nil, err
			}
		}

		var field *def.Field

		field, err = p.parseField(lex)
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

func (p *Parser) parseField(lex *lexer.Reader) (*def.Field, error) {

	var item *lexer.Item
	var err error

	item, err = p.readLexItem(lex, lexer.IDENT)
	if err != nil {
		return nil, err
	}

	fieldName := item.Literal
	if !unicode.IsLetter(rune(fieldName[0])) {
		return nil, fmt.Errorf("field name must start with a letter, got %q", fieldName)
	}

	_, err = p.readLexItem(lex, lexer.WHITESPACE)
	if err != nil {
		return nil, err
	}

	item, err = p.readLexItem(lex, lexer.IDENT)
	if err != nil {
		return nil, err
	}

	fieldType := item.Literal

	return def.NewField(fieldName, fieldType), nil
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

	annotation := def.NewAnnotation(item.Literal)

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

		return item, fmt.Errorf("expected %s, got %s, %v", tokenStr, item.Token.String(), item.Position)
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
