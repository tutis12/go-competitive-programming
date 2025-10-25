package main

import (
	"go/format"

	"golang.org/x/tools/imports"
)

func sanitizeCode(src []byte) ([]byte, error) {
	src, err := format.Source(src)
	if err != nil {
		return nil, err
	}

	return imports.Process("", src, nil)
}
