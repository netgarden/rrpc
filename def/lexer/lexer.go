package lexer

import (
	"fmt"
	"io"
	"unicode"
)

type Position struct {
	Line   int
	Column int
}

func (p Position) String() string {
	return fmt.Sprintf("line %d, column %d", p.Line, p.Column)
}

type Item struct {
	Position Position
	Token    Token
	Literal  string
}

func newItem(pos Position, token Token, lit string) *Item {
	return &Item{
		Position: pos,
		Token:    token,
		Literal:  lit,
	}
}

type Lexer struct {
	reader *RunesReader
}

func New(reader io.Reader) *Lexer {
	return &Lexer{
		reader: NewRunesReader(reader),
	}
}

// Scan scans the input for the next token. It returns the position of the token,
// the token's type, and the literal value.
func (l *Lexer) Scan() *Item {

	// keep looping until we return a token
	for {

		r, pos, err := l.reader.Read()
		if err != nil {
			if err == io.EOF {
				return newItem(pos, EOF, "")
			}

			// at this point there isn't much we can do, and the compiler
			// should just return the raw error to the user
			panic(err)
		}

		if unicode.IsSpace(r) || r == '#' {
			// A '#' comment runs to the end of the line and is absorbed into the
			// surrounding whitespace run (see readWhiteSpace), so a commented
			// line tokenizes exactly like the same line without the comment.
			l.backup()
			return l.readWhiteSpace()
		} else if r == '"' {
			startPos := pos
			l.backup()
			lit := l.readString()
			return newItem(startPos, STRING, lit)
		} else if r == '@' {
			startPos := pos
			lit := l.readAnnotationName()
			return newItem(startPos, ANNOTATION, lit)
		} else if r == '(' {
			return newItem(pos, BO, string(r))
		} else if r == ')' {
			return newItem(pos, BC, string(r))
		} else if r == '{' {
			return newItem(pos, BCO, string(r))
		} else if r == '}' {
			return newItem(pos, BCC, string(r))
		} else if r == ',' {
			return newItem(pos, COMMA, string(r))
		} else if r == '=' {
			return newItem(pos, EQ, string(r))
		} else {
			// backup and let readIdent rescan the beginning of the ident
			startPos := pos
			l.backup()
			lit := l.readIdent()

			if lit == "" {
				// A character that can't start any token. Consume it and
				// report it, instead of returning an empty IDENT without
				// advancing (which crashes the parser and can loop forever).
				bad, _, _ := l.reader.Read()
				return newItem(startPos, ILLEGAL, string(bad))
			}

			switch lit {
			case "include":
				return newItem(startPos, INCLUDE, lit)
			case "option":
				return newItem(startPos, OPTION, lit)
			case "service":
				return newItem(startPos, SERVICE, lit)
			case "struct":
				return newItem(startPos, STRUCT, lit)
			case "name":
				return newItem(startPos, NAME, lit)
			case "version":
				return newItem(startPos, VERSION, lit)
			case "error":
				return newItem(startPos, ERROR, lit)
			}

			return newItem(startPos, IDENT, lit)
		}
	}

}

// skipLine consumes input up to, but not including, the next newline (or EOF).
func (l *Lexer) skipLine() {
	for {
		r, _, err := l.reader.Read()
		if err != nil {
			return
		}
		if r == '\n' {
			l.backup()
			return
		}
	}
}

func (l *Lexer) backup() {

	if err := l.reader.Backup(); err != nil {
		panic(err)
	}

}

func (l *Lexer) readWhiteSpace() *Item {

	var startPos *Position

	var token Token = -1
	var lit string = ""

	for {

		r, pos, err := l.reader.Read()
		if startPos == nil {
			startPos = &pos
		}
		if err != nil {
			if err == io.EOF {
				break
			}
		}

		if r == '#' {
			l.skipLine() // the newline ending the comment is read next, as whitespace
			continue
		}

		if unicode.IsSpace(r) {
			if r == '\n' && token != NEWLINE {
				token = NEWLINE
			} else if token == -1 {
				token = WHITESPACE
			}
		} else {
			l.backup()
			break
		}

		lit += string(r)

	}

	if token == -1 { // a comment with nothing but EOF around it
		token = WHITESPACE
	}
	return newItem(*startPos, token, lit)
}

// readAnnotationName scans the input until the end of an annotation identifier and then returns the
// literal.
func (l *Lexer) readAnnotationName() string {
	var lit string
	for {

		r, _, err := l.reader.Read()
		if err != nil {
			if err == io.EOF {
				// at the end of the identifier
				return lit
			}
		}

		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' {
			lit = lit + string(r)
		} else {
			// scanned something not in the identifier
			l.backup()
			return lit
		}
	}
}

// readInt scans the input until the end of an integer and then returns the
// literal.
func (l *Lexer) readInt() string {
	var lit string
	for {
		r, _, err := l.reader.Read()
		if err != nil {
			if err == io.EOF {
				// at the end of the int
				return lit
			}
		}

		if unicode.IsDigit(r) {
			lit = lit + string(r)
		} else {
			// scanned something not in the integer
			l.backup()
			return lit
		}
	}
}

// readIdent scans the input until the end of an identifier and then returns the
// literal.
func (l *Lexer) readIdent() string {
	var lit string
	for {
		r, _, err := l.reader.Read()
		if err != nil {
			if err == io.EOF {
				// at the end of the identifier
				return lit
			}
		}

		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' || r == '[' || r == ']' {
			lit = lit + string(r)
		} else {
			// scanned something not in the identifier
			l.backup()
			return lit
		}
	}
}

// readString scans the input until the end of s string and then returns the
// literal.
func (l *Lexer) readString() string {

	var r rune
	var r0 rune
	var lit string

	for {

		if r != 0 {
			r0 = r
		}

		var err error
		r, _, err = l.reader.Read()
		if err != nil {
			panic(err)
		}

		if r0 == 0 && r == '"' {
			continue
		}

		if r0 != '\\' && r == '"' {
			return lit
		}

		lit = lit + string(r)
	}

}
