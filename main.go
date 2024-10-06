package main

import (
	"main/debug"
	"main/fastio"
	"main/hackercup"
	"os"
)

func main() {
	const fromFile = false

	var stdout = &fastio.Writer{
		File: os.Stdout,
	}

	var stdin = &fastio.Reader{
		File: os.Stdin,
	}
	if fromFile {
		inputFile, err := os.Open("io/wildcard_submissions_input.txt")
		if err != nil {
			panic(err.Error())
		}
		stdin.File = inputFile
		outputFile, err := os.Create("io/output.txt")
		if err != nil {
			panic(err.Error())
		}
		stdout.File = outputFile
	}

	defer stdout.WriteAll()
	defer debug.Recover()
	hackercup.Hackercup(stdin, stdout)
}

/*input
6
??2 3
135201 1
?35 2
1?0 2
1122 1
3???????????????????3 1337

*/

/*output
Case #1: 122 3
Case #2: 135201 2
Case #3: 135 2
Case #4: 110 1
Case #5: 1122 5
Case #6: 322222222121221112223 10946

*/
