package lexer

import (
	"strings"
	"testing"
)

func scanAll(src string) []Item {
	l := New(strings.NewReader(src))
	var out []Item
	for {
		it := l.Scan()
		if it.Token == EOF {
			return out
		}
		out = append(out, *it)
	}
}

func TestComments(t *testing.T) {
	for name, src := range map[string]string{
		"own line":         "# a comment\nstruct A {\n}",
		"trailing":         "struct A { # trailing\n}",
		"inside struct":    "struct A {\n    # note\n    id string\n}",
		"no final newline": "struct A {\n}\n# last line",
	} {
		t.Run(name, func(t *testing.T) {
			for _, it := range scanAll(src) {
				if it.Token == ILLEGAL || strings.Contains(it.Literal, "#") || strings.Contains(it.Literal, "note") || strings.Contains(it.Literal, "comment") {
					t.Errorf("comment leaked into token stream: %+v", it)
				}
			}
		})
	}
}

func TestCommentKeepsNewline(t *testing.T) {
	var newlines int
	for _, it := range scanAll("a # c\nb\n") {
		if it.Token == NEWLINE || (it.Token == WHITESPACE && strings.Contains(it.Literal, "\n")) {
			newlines++
		}
	}
	if newlines == 0 {
		t.Error("the newline ending a comment line must still be scanned")
	}
}

func TestHashInsideStringIsNotAComment(t *testing.T) {
	items := scanAll(`option x = "a#b"`)
	last := items[len(items)-1]
	if last.Token != STRING || !strings.Contains(last.Literal, "a#b") {
		t.Errorf("last token = %+v, want STRING containing a#b", last)
	}
}

// An unknown character must be reported and consumed, not returned as an
// empty IDENT without advancing.
func TestIllegalCharacterAdvances(t *testing.T) {
	items := scanAll("tag? string")
	var illegal *Item
	for i := range items {
		if items[i].Token == ILLEGAL {
			illegal = &items[i]
		}
		if items[i].Token == IDENT && items[i].Literal == "" {
			t.Fatalf("empty IDENT emitted: %+v", items)
		}
	}
	if illegal == nil || illegal.Literal != "?" {
		t.Fatalf("expected ILLEGAL %q, got %+v", "?", items)
	}
}
