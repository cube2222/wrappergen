package main

import (
	"context"
	"log"

	"os"

	"github.com/cube2222/StatsGenerator/app"
)

type MyInterface interface {
	HelloWorld(context.Context, LocalStruct) (*privateStruct, *LocalStruct, string, error)
	GoodbyeWorld(context.Context, int) error
}

type LocalStruct struct {
}

type privateStruct struct {
}

func main() {
	conf := &app.Config{
		InterfaceName:  os.Args[1], //"MyInterface",
		TemplatePath:   "stats.tmpl",
		OutputFilePath: os.Args[2],
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
