package main

import (
	"main/debug"
	"main/fastio"
	"os"
)

const (
	fromFile  = false
	inputFile = "crash_course_input.txt"
)

func main() {
	var stdin = &fastio.Reader{
		File: os.Stdin,
	}
	var stdout = &fastio.Writer{
		File: os.Stdout,
	}

	if fromFile {
		outputFile, err := os.Create("io/output" + inputFile)
		if err != nil {
			panic(err.Error())
		}
		stdout.File = outputFile

		inputFile, err := os.Open("io/" + inputFile)
		if err != nil {
			panic(err.Error())
		}
		stdin.File = inputFile
	}
	defer stdout.WriteAll()
	defer debug.Recover()

	//hackercup.Hackercup(stdin, stdout)
	SolveX(stdin, stdout)
}

/*input
6
7
ABBAAAB
1
A
1
B
2
AB
6
AAAAAA
7
BBBBBBA

*/

/*output
Case #1: Alice
Case #2: Alice
Case #3: Bob
Case #4: Bob
Case #5: Alice
Case #6: Alice

*/
