package main

import (
	"main/debug"
	"main/fastio"
	"main/hackercup"
	"os"
)

const (
	fromFile   = false
	inputFile  = "four_in_a_burrow_input (1).txt"
	outputFile = "output.txt"
)

/*input
1
CFCCFFC
FCFFCCF
FFCFCFC
CCFFCFC
CFCFCFF
CFCCFFC
*/

/*output
Case #1: 2
Case #2: 3
Case #3: 1
Case #4: 2

*/

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
	hackercup.Hackercup(stdin, stdout)
}
