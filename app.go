package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	"github.com/codegangsta/cli"
)

func appMain(c *cli.Context) {
	i := c.Args().First()
	if len(c.Args().Tail()) != 0 {
		fmt.Println("ERROR: input file is needed")
	}

	s := c.StringSlice("struct")
	if len(s) == 0 {
		fmt.Println("ERROR: struct name is needed")
		return
	}

	o := c.String("output")
	if o == "" {
		o = outputFileName(i)
	}

	gen, err := NewGenerator(i, s)
	if err != nil {
		fmt.Println("ERROR: " + err.Error())
		return
	}
	err = gen.Run()
	if err != nil {
		fmt.Println("ERROR: " + err.Error())
		return
	}
	output_buf, err := gen.Output()
	if err != nil {
		fmt.Println("ERROR: " + err.Error())
		return
	}

	err = ioutil.WriteFile(o, output_buf, 0666)
	if err != nil {
		fmt.Println("ERROR: " + err.Error())
		return
	}

	return
}

func outputFileName(input string) string {
	// returns input_file.gen.go
	return strings.TrimSuffix(input, ".go") + ".gen.go"
}

func main() {
	app := cli.NewApp()
	app.Name = "go-entity-generate"
	app.Usage = "go-entity-generte generates useful methods and interface for database entity."
	app.Version = "0.0.1"

	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:  "output, o",
			Value: "",
		},
		cli.StringSliceFlag{
			Name:  "struct, s",
			Value: &cli.StringSlice{},
		},
	}
	app.Action = appMain
	app.Run(os.Args)
}
