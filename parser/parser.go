package parser

import (
	"go/types"
	"io/ioutil"
	"path/filepath"

	"github.com/pkg/errors"
	"golang.org/x/tools/go/loader"
)

type SourceData struct {
	Package             *types.Package
	NamedType           *types.Named
	UnderlyingInterface *types.Interface
}

func ParseDirectory(path, interfaceName string) (*SourceData, error) {
	filenames, err := GetGoFilenames(path)
	if err != nil {
		return nil, errors.Wrap(err, "Couldn't get *.go filenames")
	}

	conf := &loader.Config{}
	conf.CreateFromFilenames("", filenames...)

	program, err := conf.Load()
	if err != nil {
		return nil, errors.Wrap(err, "Couldn't load program from go files")
	}

	if len(program.Created) == 0 {
		return nil, errors.Errorf("No package to parse")
	}
	pkg := program.Created[0].Pkg

	object := pkg.Scope().Lookup(interfaceName)
	if object == nil {
		return nil, errors.Errorf("Couldn't find object for interface called %v", interfaceName)
	}
	named, ok := object.Type().(*types.Named)
	if !ok {
		return nil, errors.Errorf("%v is not a valid type", interfaceName)
	}
	iface, ok := named.Underlying().(*types.Interface)
	if !ok {
		return nil, errors.Errorf("%v is not an interface", interfaceName)
	}

	return &SourceData{
		Package:             pkg,
		NamedType:           named,
		UnderlyingInterface: iface,
	}, nil
}

func GetGoFilenames(path string) ([]string, error) {
	files, err := ioutil.ReadDir(path)
	if err != nil {
		return nil, errors.Wrapf(err, "couldn't read content of directory with path %v", path)
	}

	filenames := make([]string, 0, len(files))

	for _, f := range files {
		if f.IsDir() == false && filepath.Ext(f.Name()) == ".go" {
			filenames = append(filenames, f.Name())
		}
	}

	return filenames, nil
}
