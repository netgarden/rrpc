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
		it := lexer.Scan()
		if it.Token == EOF {
			break
		}

		fmt.Printf("%d:%d\t%s\t%s\n", it.Position.Line, it.Position.Column, it.Token, it.Literal)
	}
}
