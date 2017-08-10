package main

import (
	"log"

	"github.com/cube2222/StatsGenerator/app"
	"gopkg.in/alecthomas/kingpin.v2"
)

var (
	InterfaceName  = kingpin.Flag("interface", "Interface to wrap.").Short('i').Required().String()
	TemplatePath   = kingpin.Flag("template", "Path of wrapper template to use.").Short('t').Required().String()
	OutputFilePath = kingpin.Flag("output", "Optional output file.").Short('o').String()
)

func main() {
	kingpin.Version("0.0.1")
	kingpin.Parse()

	conf := &app.Config{
		InterfaceName:  *InterfaceName,
		TemplatePath:   *TemplatePath,
		OutputFilePath: *OutputFilePath,
	}

	app, err := app.NewApp(conf)
	if err != nil {
		log.Fatal(err)
	}

	err = app.Run()
	if err != nil {
		log.Fatal(err)
	}
}
