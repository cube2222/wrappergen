package usertemplate

import (
	"io/ioutil"
	"strings"
	"text/template"

	"bytes"

	"github.com/pkg/errors"
)

type TemplateData struct {
	Fields  []UserSuppliedField
	Imports []string
	Method  *template.Template
	Package string
	Suffix  string
}

type UserSuppliedField struct {
	Varname, Typename string
}

func (f *UserSuppliedField) String() string {
	return strings.Join([]string{f.Varname, f.Typename}, " ")
}

type WrapperTemplateConfig struct {
	Path string
}

func GetWrapperTemplate(config *WrapperTemplateConfig) (*TemplateData, error) {
	data, err := ioutil.ReadFile(config.Path)
	if err != nil {
		return nil, errors.Wrap(err, "Couldn't read file")
	}

	packagePart := getField(data, []byte("Package"), []byte("Suffix"))

	suffixPart := getField(data, []byte("Suffix"), []byte("Imports"))

	importsPart := getField(data, []byte("Imports"), []byte("Fields"))
	imports := strings.Split(string(importsPart), "\n")

	fieldsPart := getField(data, []byte("Fields"), []byte("Method"))
	fieldsStrings := bytes.Split(fieldsPart, []byte("\n"))
	fields := []UserSuppliedField{}
	for _, fieldString := range fieldsStrings {
		parts := bytes.Split(fieldString, []byte(" "))
		if len(parts) < 2 {
			break
		}
		fields = append(fields, UserSuppliedField{
			Varname:  string(parts[0]),
			Typename: string(parts[1]),
		})
	}

	methodPart := getField(data, []byte("Method"), nil)
	tmpl, err := template.New("method").Parse(string(methodPart))
	if err != nil {
		return nil, errors.Wrap(err, "Couldn't parse template")
	}

	return &TemplateData{
		Imports: imports,
		Fields:  fields,
		Method:  tmpl,
		Package: string(packagePart),
		Suffix:  string(suffixPart),
	}, nil
}

func getField(data []byte, field []byte, nextField []byte) []byte {
	key := bytes.Join([][]byte{field, []byte(":")}, nil)
	valueIndex := bytes.Index(data, key)

	var value []byte
	if nextField != nil {
		endKey := bytes.Join([][]byte{nextField, []byte(":")}, nil)
		valueEndIndex := bytes.Index(data, endKey)
		value = bytes.TrimPrefix(data[valueIndex:valueEndIndex], key)
	} else {
		value = bytes.TrimPrefix(data[valueIndex:], key)
	}

	value = bytes.Replace(value, []byte("\r"), []byte{}, -1)
	value = bytes.Trim(value, "\r\n ")

	return value
}
