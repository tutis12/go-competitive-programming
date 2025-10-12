package main

import (
	"main/debug"
	"main/fastio"
	"os"
)

const (
	fromFile   = false
	inputFile  = "input.txt"
	outputFile = "output.txt"
)

func main() {
	var stdin = &fastio.Reader{
		File: os.Stdin,
	}
	var stdout = &fastio.Writer{
		File: os.Stdout,
	}

	if fromFile {
		inputFile, err := os.Open("io/" + inputFile)
		if err != nil {
			panic(err.Error())
		}
		stdin.File = inputFile

		outputFile, err := os.Create("io/" + outputFile)
		if err != nil {
			panic(err.Error())
		}
		stdout.File = outputFile
	}
	defer stdout.WriteAll()
	defer debug.Recover()

	solveX(stdin, stdout)
}
