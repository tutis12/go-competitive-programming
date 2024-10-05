package main

import (
	"main/fastio"
	"os"
)

const inputFile = "substantial_losses_input.txt"
const outputFile = "output.txt"

var stdout = &fastio.Writer{
	File: os.Stdout,
}

var stdin = &fastio.Reader{
	File: os.Stdin,
}

func init() {
	inputFile, err := os.Open("io/" + inputFile)
	if err != nil {
		panic(err.Error())
	}
	stdin.File = inputFile
}

func init() {
	outputFile, err := os.Create("io/" + outputFile)
	if err != nil {
		panic(err.Error())
	}
	stdout.File = outputFile
}

// func init() {
// 	stdin.File = os.Stdin
// 	stdout.File = os.Stdout
// }
