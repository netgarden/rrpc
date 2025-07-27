package lexer

import (
	"bufio"
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
	pos    Position
	reader *bufio.Reader
}

func New(reader io.Reader) *Lexer {
	return &Lexer{
		pos:    Position{Line: 1, Column: 0},
		reader: bufio.NewReader(reader),
	}
}

// Scan scans the input for the next token. It returns the position of the token,
// the token's type, and the literal value.
func (l *Lexer) Scan() *Item {

	// keep looping until we return a token
	for {

		r, _, err := l.reader.ReadRune()
		if err != nil {
			if err == io.EOF {
				return newItem(l.pos, EOF, "")
			}

			// at this point there isn't much we can do, and the compiler
			// should just return the raw error to the user
			panic(err)
		}

		// update the Column to the position of the newly read in rune
		l.pos.Column++

		if r == '\n' {
			l.resetPosition()
			continue

		} else if unicode.IsSpace(r) {
			l.backup()
			pos, lit := l.readWhiteSpace()
			if lit == ":newline:" {
				return newItem(pos, NEWLINE, lit)
			} else {
				return newItem(pos, WHITESPACE, lit)
			}
		} else if r == '"' {
			startPos := l.pos
			l.backup()
			lit := l.readString()
			return newItem(startPos, STRING, lit)
		} else if r == '@' {
			startPos := l.pos
			lit := l.readAnnotationName()
			return newItem(startPos, ANNOTATION, lit)
		} else if r == '(' {
			return newItem(l.pos, BO, string(r))
		} else if r == ')' {
			return newItem(l.pos, BC, string(r))
		} else if r == '{' {
			return newItem(l.pos, BCO, string(r))
		} else if r == '}' {
			return newItem(l.pos, BCC, string(r))
		} else if r == ',' {
			return newItem(l.pos, COMMA, string(r))
		} else if r == '=' {
			return newItem(l.pos, EQ, string(r))
		} else {
			// backup and let readIdent rescan the beginning of the ident
			startPos := l.pos
			l.backup()
			lit := l.readIdent()

			switch lit {
			case "include":
				return newItem(startPos, INCLUDE, lit)
			case "option":
				return newItem(startPos, OPTION, lit)
			case "service":
				return newItem(startPos, SERVICE, lit)
			case "struct":
				return newItem(startPos, STRUCT, lit)
			case "version":
				return newItem(startPos, VERSION, lit)
			}

			return newItem(startPos, IDENT, lit)
		}
	}

}

func (l *Lexer) resetPosition() {
	l.pos.Line++
	l.pos.Column = 0
}

func (l *Lexer) backup() {

	if err := l.reader.UnreadRune(); err != nil {
		panic(err)
	}

	l.pos.Column--
}

func (l *Lexer) readWhiteSpace() (Position, string) {

	var lit string
	startPos := l.pos

	for {

		r, _, err := l.reader.ReadRune()
		if err != nil {
			if err == io.EOF {
				// at the end of the int
				return startPos, lit
			}
		}

		currentPos := l.pos

		l.pos.Column++

		if unicode.IsSpace(r) {
			if r == '\n' && lit != ":newline:" {
				startPos = currentPos
				lit = ":newline:"
			} else if lit == "" {
				lit = ":whitespace:"
			}
		} else {
			// scanned something not in the integer
			l.backup()
			return startPos, lit
		}

	}

}

// readAnnotationName scans the input until the end of an annotation identifier and then returns the
// literal.
func (l *Lexer) readAnnotationName() string {
	var lit string
	for {
		r, _, err := l.reader.ReadRune()
		if err != nil {
			if err == io.EOF {
				// at the end of the identifier
				return lit
			}
		}

		l.pos.Column++
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
		r, _, err := l.reader.ReadRune()
		if err != nil {
			if err == io.EOF {
				// at the end of the int
				return lit
			}
		}

		l.pos.Column++
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
		r, _, err := l.reader.ReadRune()
		if err != nil {
			if err == io.EOF {
				// at the end of the identifier
				return lit
			}
		}

		l.pos.Column++
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
		r, _, err = l.reader.ReadRune()
		if err != nil {
			panic(err)
		}

		l.pos.Column++

		if r0 == 0 && r == '"' {
			continue
		}

		if r0 != '\\' && r == '"' {
			return lit
		}

		lit = lit + string(r)
	}

}

func (l *Lexer) read() (rune, int, error) {

	r, n, err := l.reader.ReadRune()
	if err == nil && r == '\n' {
		l.resetPosition()
	}

	return r, n, err
}

func (l *Lexer) unread() error {

	err := l.reader.UnreadRune()
	if err != nil {
		return err
	}

	r, _, err := l.reader.ReadRune()
	if err == nil && r == '\n' {

	}

	return nil
}
