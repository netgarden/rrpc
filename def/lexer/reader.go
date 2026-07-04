package lexer

func NewReader(lex *Lexer) *Reader {
	return &Reader{
		lex:  lex,
		pos:  -1,
		data: make([]*Item, 0),
	}
}

type Reader struct {
	lex *Lexer

	data []*Item
	pos  int
}

func (r *Reader) Read() *Item {

	if r.pos >= 0 && r.data[r.pos].Token == EOF {
		return r.data[r.pos]
	}

	r.pos++

	if r.pos >= len(r.data) {
		r.data = append(r.data, r.lex.Scan())
	}

	return r.data[r.pos]
}

func (r *Reader) Unread() {
	if r.pos > -1 {
		r.pos--
	}
}
