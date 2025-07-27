package lexer

import (
	"bufio"
	"errors"
	"io"
)

type RuneData struct {
	Rune     rune
	Position Position
}

func NewRunesReader(reader io.Reader) *RunesReader {
	return &RunesReader{
		reader:  bufio.NewReader(reader),
		data:    make([]RuneData, 0),
		runePos: 0,
	}
}

type RunesReader struct {
	reader   *bufio.Reader
	data     []RuneData
	runePos  int
	runeRead int
}

func (r *RunesReader) Read() (rune, Position, error) {

	var runeData RuneData

	if r.runePos == len(r.data) {

		readRune, _, err := r.reader.ReadRune()
		if err != nil {
			return 0, Position{}, err
		}

		lastRuneData := r.data[r.runePos-1]
		pos := lastRuneData.Position

		if readRune == '\n' {
			pos.Line++
			pos.Column = 1
		} else {
			pos.Column++
		}

		runeData = RuneData{
			Rune:     readRune,
			Position: pos,
		}

		r.data = append(r.data, runeData)

	} else {

		runeData = r.data[r.runePos]

	}

	r.runePos++

	return runeData.Rune, runeData.Position, nil
}

func (r *RunesReader) Backup() error {

	if r.runePos == 0 {
		return errors.New("no runes available to backup")
	}

	r.runePos--

	return nil
}
