package printer

import (
	"bufio"
	"bytes"
	"io"

	"github.com/pkg/errors"
	"golang.org/x/tools/imports"
)

type PrinterConfig struct {
}

func Print(out io.Writer, data []byte, config *PrinterConfig) error {
	data, err := imports.Process("output.go", data, nil)
	if err != nil {
		return errors.Wrap(err, "couldn't prettify file")
	}

	_, err = io.Copy(out, bufio.NewReader(bytes.NewReader(data)))
	if err != nil {
		return errors.Wrap(err, "couldn't write data")
	}

	return nil
}
