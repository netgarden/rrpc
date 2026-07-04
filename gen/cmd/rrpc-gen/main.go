package main

import (
	"os"

	"github.com/alecthomas/kingpin/v2"
	"github.com/netgarden/rrpc/def/parser"
	gen "github.com/netgarden/rrpc/gen"
)

type Cmd struct {
	genType string
	source  string
	target  string
}

func main() {

	cmd := &Cmd{}

	app := kingpin.New("rrpc-gen", "RRPC code generator")
	app.Flag("source", "Source file").Short('s').Required().StringVar(&cmd.source)
	app.Flag("target", "Target directory").Short('d').Required().StringVar(&cmd.target)
	app.Flag("type", "Generator type. Available: go, ts").Short('t').Default("go").StringVar(&cmd.genType)

	_, err := app.Parse(os.Args[1:])
	if err != nil {
		panic(err)
	}

	err = cmd.run()
	if err != nil {
		panic(err)
	}

}

func (c *Cmd) run() error {

	p := parser.New()
	definition, err := p.Parse(c.source)
	if err != nil {
		return err
	}

	generator, err := gen.NewGenerator(c.genType)
	if err != nil {
		return err
	}

	err = generator.Generate(definition, c.target)
	if err != nil {
		return err
	}

	return nil
}
