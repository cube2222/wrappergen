package generator

import (
	"bytes"
	"fmt"
	"go/types"
	"strings"

	"io"

	"text/template"

	"github.com/cube2222/StatsGenerator/analyzer"
	"github.com/cube2222/StatsGenerator/parser"
	"github.com/cube2222/StatsGenerator/usertemplate"
	"github.com/pkg/errors"
)

func NewWrapperGenerator(sourceData *parser.SourceData, wrapperData *analyzer.WrapperTypeData, templateData *usertemplate.TemplateData) *WrapperGenerator {
	return &WrapperGenerator{
		sourceData:   sourceData,
		wrapperData:  wrapperData,
		templateData: templateData,
		out:          bytes.NewBuffer(nil),
	}
}

type WrapperGenerator struct {
	sourceData   *parser.SourceData
	wrapperData  *analyzer.WrapperTypeData
	templateData *usertemplate.TemplateData
	out          *bytes.Buffer
}

func (g *WrapperGenerator) Read(p []byte) (n int, err error) {
	return g.out.Read(p)
}

func (g *WrapperGenerator) GetBytes() []byte {
	return g.out.Bytes()
}

func (g *WrapperGenerator) Generate() error {
	writePackage(g.out, g.wrapperData.Pkg)
	writeImports(g.out, g.sourceData.Package.Imports())
	writeUserSuppliedImports(g.out, g.templateData.Imports)

	writeStructure(g.out, g.wrapperData, g.templateData.Fields)
	writeConstructor(g.out, g.sourceData.NamedType, g.wrapperData.Pkg, g.wrapperData.NamedType, g.templateData.Fields)

	for i := 0; i < g.sourceData.UnderlyingInterface.NumMethods(); i++ {
		curMethod := g.sourceData.UnderlyingInterface.Method(i)

		md := getMethodData(
			g.sourceData.NamedType,
			curMethod,
			g.wrapperData.Pkg,
			g.wrapperData.NamedType,
		)

		curSignature := curMethod.Type().(*types.Signature)

		writeMethod(g.out, md, curSignature, g.wrapperData, g.templateData.Method)
	}

	return nil
}

func writeStructure(w io.Writer, wrapperType *analyzer.WrapperTypeData, userSuppliedFields []usertemplate.UserSuppliedField) {
	tmpl := `type %s struct {
	%s
}
`

	fields := []string{}

	wrapperStructure := wrapperType.NamedType.Underlying().(*types.Struct)

	buf := bytes.NewBuffer(nil)
	types.WriteType(buf, wrapperStructure.Field(0).Type(), types.RelativeTo(wrapperType.Pkg))
	fields = append(fields, fmt.Sprintf("%s %s", wrapperStructure.Field(0).Name(), buf.String()))
	for _, field := range userSuppliedFields {
		fields = append(fields, field.String())
	}

	fmt.Fprintf(w, tmpl, wrapperType.NamedType.Obj().Name(), strings.Join(fields, "\n"))
}

func writeMethod(w io.Writer, md *MethodData, signature *types.Signature, wrapperTypeData *analyzer.WrapperTypeData, tmpl *template.Template) error {
	WriteSignature(w, md, signature, wrapperTypeData.Pkg, wrapperTypeData.NamedType)
	fmt.Fprint(w, " {\n")
	err := tmpl.Execute(w, md)
	if err != nil {
		return errors.Wrap(err, "couldn't execute method template")
	}
	fmt.Fprint(w, "}\n")

	return nil
}

func writePackage(w io.Writer, pkg *types.Package) {
	fmt.Fprintf(w, "package %s\n", pkg.Name())
}

func writeImports(w io.Writer, imports []*types.Package) {
	for _, i := range imports {
		fmt.Fprintf(w, "import \"%s\"\n", i.Name())
	}
}

func writeUserSuppliedImports(w io.Writer, imports []string) {
	for _, i := range imports {
		fmt.Fprintf(w, "import \"%s\"\n", i)
	}
}

func WriteSignature(w io.Writer, md *MethodData, originalSignature *types.Signature, curPkg *types.Package, created *types.Named) {
	receiverType := types.NewPointer(created)
	createdTypeBuffer := bytes.NewBuffer(nil)
	types.WriteType(createdTypeBuffer, receiverType, types.RelativeTo(curPkg))

	argumentVariables := []*types.Var{}
	for i := 0; i < originalSignature.Params().Len(); i++ {
		param := originalSignature.Params().At(i)
		argumentVariables = append(argumentVariables, types.NewVar(0, curPkg, md.Arguments[i], param.Type()))
	}
	arguments := types.NewTuple(argumentVariables...)

	receiver := types.NewVar(0, curPkg, md.ReceiverVar, receiverType)
	newSignature := types.NewSignature(receiver, arguments, originalSignature.Results(), false)

	signatureBuffer := bytes.NewBuffer(nil)
	types.WriteSignature(signatureBuffer, newSignature, types.RelativeTo(curPkg))

	fmt.Fprintf(
		w,
		"func (%s %s) %s%s",
		md.ReceiverVar,
		createdTypeBuffer,
		md.FunctionName,
		signatureBuffer,
	)
}

func writeConstructor(w io.Writer, originalInterfaceType *types.Named, curPkg *types.Package, created *types.Named, userSuppliedFields []usertemplate.UserSuppliedField) {
	constructorTemplate := `
func New%s(%s) %s {
	return &%s{
		%s
	}
}
`
	fieldStrings := []string{
		fmt.Sprintf("wrapped %s", originalInterfaceType),
	}
	for _, field := range userSuppliedFields {
		fieldStrings = append(fieldStrings, field.String())
	}

	initializers := []string{
		fmt.Sprintf("wrapped: wrapped,"),
	}
	for _, field := range userSuppliedFields {
		initializers = append(initializers, fmt.Sprintf("%s: %s,", field.Varname, field.Varname))
	}

	createdNameBuffer := bytes.NewBuffer(nil)
	types.WriteType(createdNameBuffer, created, types.RelativeTo(curPkg))

	fmt.Fprintf(
		w,
		constructorTemplate,
		createdNameBuffer,
		strings.Join(fieldStrings, ", "),
		originalInterfaceType,
		createdNameBuffer,
		strings.Join(initializers, "\n"),
	)
}

type MethodData struct {
	// The name of the function being wrapped
	// example: MyFunction
	FunctionName string
	// The name of the function being wrapped, but all lowercase
	// example: myfunction
	LowercaseFunctionName string
	// The name of the receiver variable. It's the type name without the initial letter capitalized
	// example: myInterfaceWrapper
	ReceiverVar string
	// The original interface name, with the package name prepended
	// example: pkg.MyInterface
	FullOriginalTypeName string
	// The original interface name, with the package name prepended, but all lowercase
	// example: pkg.myinterface
	LowercaseFullOriginalTypeName string
	// The original interface name only, without the package
	// example: MyInterface
	ShortOriginalTypeName string
	// A variable for each of the variables returned by this function
	// example: []string{var0, var1, var2}
	ReturnVars []string
	// A set of variables returned by this function, comma-seperated
	// example: var0, var1, var2
	ReturnVarsConnected string
	// This is set to true, if one of the return types of this function is an error
	ErrorPresent bool
	// An argument name for each of the arguments taken by this function
	// example: []string{input0, input1, input2}
	Arguments []string
	// The set of arguments taken by this function, comma-seperated
	// example: input0, input1, input2
	ArgumentsConnected string // strings.Join(arguments, ", ")
	// This contains the call to the wrapped function
	// example: myInterfaceWrapper.wrapped.MyFunction(input0, input1, input2)
	CallWrapped string
	// This contains the set of zero variable declarations corresponding to the return variables followed by a return
	// example:
	// var zero0 type0
	// var zero1 type1
	// var zero2 type2
	// return zero0, zero1, zero2
	ZeroValuesReturn string
	// This contains the set of zero variable declarations corresponding to the return variables, but excluding the error, if present.
	// Followed by a return
	// You can use this, and right after this you can continue with the error expression you want to return as the error.
	// example:
	// var zero0 type0
	// var zero1 type1
	// return zero0, zero1,
	ZeroValuesReturnWithoutError string
}

func getMethodData(originalInterfaceType *types.Named, originalFunction *types.Func, curPkg *types.Package, receiverType *types.Named) *MethodData {
	md := &MethodData{}

	md.FunctionName = originalFunction.Name()
	md.LowercaseFunctionName = strings.ToLower(md.FunctionName)

	originalSignature := getFunctionSignature(originalFunction)

	arguments := getArguments(originalSignature, curPkg)

	md.ReceiverVar = getReceiverVariableName(receiverType, curPkg)

	signature := makeSignature(curPkg, md.ReceiverVar, receiverType, arguments, originalSignature)

	md.FullOriginalTypeName = getFullOriginalTypename(originalInterfaceType, curPkg)
	md.LowercaseFullOriginalTypeName = strings.ToLower(md.FullOriginalTypeName)

	md.ShortOriginalTypeName = originalInterfaceType.Obj().Name()

	md.ReturnVars, md.ErrorPresent = getReturnVarsAndCheckErrorPresent(signature)
	md.ReturnVarsConnected = strings.Join(md.ReturnVars, ", ")

	md.Arguments = getArgumentNames(signature)
	md.ArgumentsConnected = strings.Join(md.Arguments, ", ")

	md.CallWrapped = fmt.Sprintf(
		"%s.wrapped.%s(%s)",
		md.ReceiverVar,
		originalFunction.Name(),
		md.ArgumentsConnected,
	)

	md.ZeroValuesReturn, md.ZeroValuesReturnWithoutError = zeroValuesReturn(signature, curPkg)

	return md
}

// Contains the zero value returns with declaration. One with the error and one without, so the user can supply the error
func zeroValuesReturn(signature *types.Signature, curPkg *types.Package) (string, string) {
	zeroValueDeclarations := []string{}
	zeroValueVariables := []string{}
	zeroValueDeclarationsWithoutError := []string{}
	zeroValueVariablesWithoutError := []string{}
	for i := 0; i < signature.Results().Len(); i++ {
		currentType := signature.Results().At(i).Type()
		zeroVal := types.NewVar(0, curPkg, fmt.Sprintf("zero%d", i), currentType)

		zeroValueDeclarations = append(zeroValueDeclarations, zeroVal.String())
		zeroValueVariables = append(zeroValueVariables, zeroVal.Name())

		if currentType.String() != "error" {
			zeroValueDeclarationsWithoutError = append(zeroValueDeclarationsWithoutError, zeroVal.String())
			zeroValueVariablesWithoutError = append(zeroValueVariablesWithoutError, zeroVal.Name())
		}
	}
	zeroValuesReturn := strings.Join(
		append(
			zeroValueDeclarations,
			fmt.Sprintf("return %s", strings.Join(zeroValueVariables, ", ")),
		),
		"\n",
	)
	zeroValuesReturnWithoutError := strings.Join(
		append(
			zeroValueDeclarationsWithoutError,
			fmt.Sprintf("return %s", strings.Join(append(zeroValueVariablesWithoutError, ""), ", ")),
		),
		"\n",
	)
	return zeroValuesReturn, zeroValuesReturnWithoutError
}

func getArgumentNames(signature *types.Signature) []string {
	argumentNames := []string{}
	for i := 0; i < signature.Params().Len(); i++ {
		argumentNames = append(argumentNames, signature.Params().At(i).Name())
	}

	return argumentNames
}

func getReturnVarsAndCheckErrorPresent(signature *types.Signature) ([]string, bool) {
	returnVars := []string{}
	errorPresent := false
	for i := 0; i < signature.Results().Len(); i++ {
		currentVar := signature.Results().At(i)
		if currentVar.Type().String() != "error" { // supply error type
			returnVars = append(returnVars, fmt.Sprintf("var%d", i))
			continue
		}
		errorPresent = true
		returnVars = append(returnVars, "err")
	}
	return returnVars, errorPresent
}

func getFullOriginalTypename(originalInterfaceType *types.Named, curPkg *types.Package) string {
	originalTypeNameBuffer := bytes.NewBuffer(nil)
	types.WriteType(originalTypeNameBuffer, originalInterfaceType, types.RelativeTo(curPkg))
	fullOriginalTypeName := originalTypeNameBuffer.String()

	return fullOriginalTypeName
}

func makeSignature(curPkg *types.Package, receiverVariableName string, receiverType *types.Named, arguments *types.Tuple, originalSignature *types.Signature) *types.Signature {
	FunctionReceiver := types.NewVar(0, curPkg, receiverVariableName, receiverType)
	signature := types.NewSignature(FunctionReceiver, arguments, originalSignature.Results(), false)

	return signature
}

func getReceiverVariableName(receiverType *types.Named, curPkg *types.Package) string {
	receiverVarBuffer := bytes.NewBuffer(nil)
	types.WriteType(receiverVarBuffer, receiverType, types.RelativeTo(curPkg))

	// Make the first letter lowercase
	receiverName := strings.Join([]string{strings.ToLower(receiverVarBuffer.String()[0:1]), receiverVarBuffer.String()[1:]}, "")

	return receiverName
}

func getArguments(originalSignature *types.Signature, curPkg *types.Package) *types.Tuple {
	argumentVariables := []*types.Var{}
	for i := 0; i < originalSignature.Params().Len(); i++ {
		param := originalSignature.Params().At(i)
		argumentVariables = append(argumentVariables, types.NewVar(0, curPkg, fmt.Sprintf("input%d", i), param.Type()))
	}
	arguments := types.NewTuple(argumentVariables...)

	return arguments
}

func getFunctionSignature(originalFunction *types.Func) *types.Signature {
	return originalFunction.Type().(*types.Signature)
}
