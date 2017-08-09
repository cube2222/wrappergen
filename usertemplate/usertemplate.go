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

	importsIndex := bytes.Index(data, []byte("Imports:"))
	fieldsIndex := bytes.Index(data, []byte("Fields:"))
	methodIndex := bytes.Index(data, []byte("Method:"))
	importsPart := bytes.TrimPrefix(data[importsIndex:fieldsIndex], []byte("Imports:"))
	fieldsPart := bytes.TrimPrefix(data[fieldsIndex:methodIndex], []byte("Fields:"))
	methodPart := bytes.TrimPrefix(data[methodIndex:], []byte("Method:"))

	importsPart = bytes.Replace(importsPart, []byte("\r"), []byte{}, -1)
	importsPart = bytes.Trim(importsPart, "\r\n ")
	imports := strings.Split(string(importsPart), "\n")

	fieldsPart = bytes.Replace(fieldsPart, []byte("\r"), []byte{}, -1)
	fieldsPart = bytes.Trim(fieldsPart, "\r\n ")
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

	methodPart = bytes.Replace(methodPart, []byte("\r"), []byte{}, -1)
	methodPart = bytes.Trim(methodPart, "\r\n ")
	tmpl, err := template.New("method").Parse(string(methodPart))
	if err != nil {
		return nil, errors.Wrap(err, "Couldn't parse template")
	}

	return &TemplateData{
		Imports: imports,
		Fields:  fields,
		Method:  tmpl,
	}, nil
}
