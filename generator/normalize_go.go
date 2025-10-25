package main

import (
	"bytes"
	"go/format"

	"golang.org/x/tools/imports"
)

func normalizeGo(src []byte) ([]byte, error) {
	out, err := format.Source(bytes.NewBuffer(src).Bytes())
	if err != nil {
		return nil, err
	}

	return imports.Process("", out, nil)
}
