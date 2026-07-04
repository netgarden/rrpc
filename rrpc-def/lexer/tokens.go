package lexer

type Token int

const (
	EOF = iota
	ILLEGAL
	WHITESPACE
	NEWLINE
	IDENT
	TYPE
	STRING

	NAME
	VERSION
	OPTION
	STRUCT
	SERVICE
	INCLUDE
	ERROR

	ANNOTATION

	BO // (
	BC // )

	BCO // {
	BCC // }

	COMMA // ,
	EQ    // =
)

var tokens = map[Token]string{

	EOF:        "EOF",
	ILLEGAL:    "ILLEGAL",
	WHITESPACE: "WHITESPACE",
	NEWLINE:    "NEWLINE",
	IDENT:      "IDENT",
	TYPE:       "TYPE",
	STRING:     "STRING",

	NAME:    "NAME",
	VERSION: "VERSION",
	OPTION:  "OPTION",
	STRUCT:  "STRUCT",
	SERVICE: "SERVICE",
	INCLUDE: "INCLUDE",
	ERROR:   "ERROR",

	ANNOTATION: "ANNOTATION",

	BO:  "(",
	BC:  ")",
	BCO: "{",
	BCC: "}",

	COMMA: ",",
	EQ:    "=",
}

func (t Token) String() string {
	return tokens[t]
}
