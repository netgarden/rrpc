package lexer

import (
	"fmt"
	"os"
	"testing"
)

func Test(t *testing.T) {
	file, err := os.Open("test.rrpc")
	if err != nil {
		panic(err)
	}

	lexer := New(file)
	for {
		pos, tok, lit := lexer.Scan()
		if tok == EOF {
			break
		}

		fmt.Printf("%d:%d\t%s\t%s\n", pos.Line, pos.Column, tok, lit)
	}
}
