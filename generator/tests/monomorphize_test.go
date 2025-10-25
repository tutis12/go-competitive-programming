package main

import (
	"fmt"
	"go/format"
	"io"
	"main/generator/monomorphize"
	"os"
	"testing"
)

func TestMonomorphize(t *testing.T) {
	// search for folders in the current directory
	files, err := os.ReadDir(".")
	if err != nil {
		panic(err.Error())
	}
	for _, file := range files {
		if !file.IsDir() {
			continue
		}
		testCase(file.Name())
	}
}

func testCase(
	name string,
) {
	fmt.Println("Testing:", name)
	srcFile, err := os.Open(fmt.Sprintf("%s/source/src.go", name))
	if err != nil {
		panic(err.Error())
	}
	defer srcFile.Close()
	src, err := io.ReadAll(srcFile)
	if err != nil {
		panic(err.Error())
	}
	expectedFile, err := os.Open(fmt.Sprintf("%s/expected/expected.go", name))
	if err != nil {
		panic(err.Error())
	}
	defer expectedFile.Close()
	expected, err := io.ReadAll(expectedFile)
	if err != nil {
		panic(err.Error())
	}
	output := monomorphize.Monomorphize(src)

	output, err = format.Source(output)
	if err != nil {
		panic(err.Error())
	}

	if string(output) != string(expected) {
		fmt.Println(string(output))
		panic("mismatch in " + name)
	}
}
