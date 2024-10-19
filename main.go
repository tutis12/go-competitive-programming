package main

import (
	"main/debug"
	"main/fastio"
	"main/hackercup"
	"os"
)

const (
	fromFile   = false
	inputFile  = "substitution_cipher_input.txt"
	outputFile = "output.txt"
)

/*input
7
-1
1
2
3
4
5
6
*/

/*output
Case #1: -1
Case #2: 1
Case #3: 2
Case #4: 3
Case #5: 4
Case #6: 5
Case #7: 6

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
