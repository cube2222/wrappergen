package utils

import "io"

type nopCloser struct {
	io.Writer
}

func NopCloser(w io.Writer) io.WriteCloser {
	return &nopCloser{w}
}

func (*nopCloser) Close() error {
	return nil
}
