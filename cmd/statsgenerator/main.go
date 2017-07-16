package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"path/filepath"

	"go/types"

	"golang.org/x/tools/go/loader"
	"bytes"
	"strings"
	"context"
)

type MyInterface interface {
	HelloWorld(context.Context, LocalStruct) (*privateStruct, *LocalStruct, string, error)
}

type LocalStruct struct {

}

type privateStruct struct {

}

func main() {
	conf := &loader.Config{}

	files, err := ioutil.ReadDir(".")
	if err != nil {
		log.Fatal(err)
	}

	filenames := []string{}

	for _, f := range files {
		if f.IsDir() == false && filepath.Ext(f.Name()) == ".go" {
			filenames = append(filenames, f.Name())
		}
	}

	conf.CreateFromFilenames("", filenames...)

	prog, err := conf.Load()
	if err != nil {
		log.Fatal(err)
	}

	if len(prog.Created) > 1 {
		log.Fatal("Why do I have more than 1 initial package? This shouldn't happen.")
	}

	pkg := prog.Created[0].Pkg

	name := "MyInterface"

	object := pkg.Scope().Lookup(name)
	if object == nil {
		log.Fatalf("Interface %v not found", name)
	}
	named, ok := object.Type().(*types.Named)
	if !ok {
		log.Fatal("Interface is not an interface")
	}
	i, ok := named.Underlying().(*types.Interface)
	if !ok {
		log.Fatal("Interface is not an interface")
	}

	// *******

	newPkg := types.NewPackage("stats", "statspkg")

	wrapped := types.NewVar(0, newPkg, "wrapped", object.Type())

	newStruct := types.NewStruct([]*types.Var{wrapped}, []string{})
	NewTypeName := types.NewTypeName(0, newPkg, "StatsStruct", newStruct)
	NewNamed := types.NewNamed(NewTypeName, NewTypeName.Type(), nil)

	newPkg.SetImports(append(newPkg.Imports(), pkg.Imports()...))
	newPkg.SetImports(append(newPkg.Imports(), pkg))

	fmt.Println(newPkg.Imports())

	fmt.Println(NewTypeName)

	fmt.Println(GetMethodString(
		i.Method(0).Type().(*types.Signature),
		newPkg,
		NewNamed,
	))

}



func GetMethodString(originalSignature *types.Signature, curPkg *types.Package, receiver *types.Named) string {
	funcName := "MyFunc"

	functionParams := originalSignature.Params()

	vars := []*types.Var{}
	for i := 0; i < functionParams.Len(); i++ {
		param := functionParams.At(i)
		vars = append(vars, types.NewVar(0, curPkg, fmt.Sprintf("input%d", i), param.Type()))
	}
	newParams := types.NewTuple(vars...)

	FunctionReceiver := types.NewVar(0, curPkg, "statsStruct", receiver)
	Signature := types.NewSignature(FunctionReceiver, newParams, originalSignature.Results(), false)
	NewFunc := types.NewFunc(0, curPkg, funcName, Signature)

	functionTemplate := `
func (%s %s) %s%s {
	t := stats.NewTimer("%s.ok")
	defer t.End()
	%s
}
		`

	receiverName := Signature.Recv().Name()

	receiverTypeBuffer := bytes.NewBuffer(nil)
	types.WriteType(receiverTypeBuffer, Signature.Recv().Type(), types.RelativeTo(curPkg))

	functionName := NewFunc.Name()

	signatureBuffer := bytes.NewBuffer(nil)
	types.WriteSignature(signatureBuffer, Signature, types.RelativeTo(curPkg))

	statName := strings.ToLower(strings.Join([]string{originalSignature.Recv().Type().String(), functionName}, "."))

	returned := []string{}
	errorPresent := false
	for i := 0; i < Signature.Results().Len(); i++ {
		currentVar := Signature.Results().At(i)
		if currentVar.Type().String() != "error" { // supply error type
			returned = append(returned, fmt.Sprintf("var%d", i))
			continue
		}
		errorPresent = true
		returned = append(returned, "err")
	}
	prettyReturned := strings.Join(returned, ", ")

	internalFunctionCallTemplate := "%s.wrapped.%s(%s)"

	arguments := []string{}
	for i := 0; i < Signature.Params().Len(); i++ {
		arguments = append(arguments, Signature.Params().At(i).Name())
	}

	wrappedFunctionCall := fmt.Sprintf(
		internalFunctionCallTemplate,
		receiverName,
		funcName,
		strings.Join(arguments, ", "),
	)

	errorDependantPart := ""
	if errorPresent == false {
		errorAbsentTemplate := "return %s"

		errorDependantPart = fmt.Sprintf(errorAbsentTemplate, wrappedFunctionCall)
	} else {
		errorPresentTemplate := `%s := %s
	if err != nil {
		t.Set("%s.err")
		return %s
	}
	return %s`

		errorDependantPart = fmt.Sprintf(
			errorPresentTemplate,
			prettyReturned,
			wrappedFunctionCall,
			statName,
			prettyReturned,
			prettyReturned,
		)
	}

	return fmt.Sprintf(
		functionTemplate,
		receiverName,
		receiverTypeBuffer,
		functionName,
		signatureBuffer,
		statName,
		errorDependantPart,
	)
}
