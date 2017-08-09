package app

import (
	"io"
	"log"
	"os"

	"github.com/cube2222/StatsGenerator/analyzer"
	"github.com/cube2222/StatsGenerator/generator"
	"github.com/cube2222/StatsGenerator/parser"
	"github.com/cube2222/StatsGenerator/printer"
	"github.com/cube2222/StatsGenerator/usertemplate"
	"github.com/cube2222/StatsGenerator/utils"
	"github.com/pkg/errors"
)

type App struct {
	config *Config
	output io.WriteCloser
}

type Config struct {
	InterfaceName  string
	TemplatePath   string
	OutputFilePath string
}

func NewApp(config *Config) (*App, error) {
	a := &App{
		config: config,
	}

	if config.OutputFilePath != "" {
		file, err := os.Create(config.OutputFilePath)
		if err != nil {
			return nil, errors.Wrapf(err, "Couldn't create file: %v", config.OutputFilePath)
		}
		a.output = file
	} else {
		a.output = utils.NopCloser(os.Stdout)
	}

	return a, nil
}

func (a *App) Run() error {
	defer a.output.Close()

	sourceData, err := parser.ParseDirectory(".", a.config.InterfaceName)
	if err != nil {
		log.Fatal(err)
	}

	tmplConfig := &usertemplate.WrapperTemplateConfig{
		Path: a.config.TemplatePath,
	}
	templateData, err := usertemplate.GetWrapperTemplate(tmplConfig)
	if err != nil {
		log.Fatal(err)
	}

	wrapperTypeData := analyzer.GetWrapperTypeData(sourceData)

	g := generator.NewWrapperGenerator(sourceData, wrapperTypeData, templateData)
	g.Generate()

	err = printer.Print(a.output, g.GetBytes(), nil)
	if err != nil {
		log.Fatal(err)
	}

	return nil
}
