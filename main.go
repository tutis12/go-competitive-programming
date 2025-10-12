package main

import (
	"main/debug"
	"main/fastio"
	"os"
)

const (
	task       = "A"
	fromFile   = false
	inputFile  = "input.txt"
	outputFile = "output.txt"
)

func solveTask(
	stdin *fastio.Reader,
	stdout *fastio.Writer,
) {
	switch task {
	case "A":
		solveA(stdin, stdout)
	case "B":
		solveB(stdin, stdout)
	case "C":
		solveC(stdin, stdout)
	case "D":
		solveD(stdin, stdout)
	case "E":
		solveE(stdin, stdout)
	case "F":
		solveF(stdin, stdout)
	case "G":
		solveG(stdin, stdout)
	default:
		panic("unknown task")
	}
}

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

	solveTask(stdin, stdout)
}
