package analyzer

import (
	"fmt"
	"go/types"

	"github.com/cube2222/StatsGenerator/parser"
)

type WrapperTypeData struct {
	Pkg       *types.Package
	NamedType *types.Named
}

func GetWrapperTypeData(sourceData *parser.SourceData) *WrapperTypeData {
	wrapperPkg := types.NewPackage("stats", "stats")

	addImports(wrapperPkg, sourceData)

	// Umożliwić dodawanie nowych pól i tak samo wtedy zczytywać i dostosować konstruktor
	wrapped := types.NewVar(0, wrapperPkg, "wrapped", sourceData.NamedType)

	wrapperName := fmt.Sprintf("%s%s", sourceData.NamedType.Obj().Name(), "Stats")

	newStruct := types.NewStruct([]*types.Var{wrapped}, []string{})

	wrapperTypeName := types.NewTypeName(0, wrapperPkg, wrapperName, newStruct)
	wrapperNamedType := types.NewNamed(wrapperTypeName, wrapperTypeName.Type(), nil)

	return &WrapperTypeData{
		Pkg:       wrapperPkg,
		NamedType: wrapperNamedType,
	}
}

func addImports(wrapperPkg *types.Package, sourceData *parser.SourceData) {
	wrapperPkg.SetImports(append(wrapperPkg.Imports(), sourceData.Package.Imports()...))
	wrapperPkg.SetImports(append(wrapperPkg.Imports(), sourceData.Package))
}
