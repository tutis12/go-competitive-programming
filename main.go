package main

import (
	"main/debug"
	"main/fastio"
	"main/hackercup"
	"os"
)

const (
	fromFile   = false
	inputFile  = "warm_up_input.txt"
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

	hackercup.Hackercup(stdin, stdout)
}

/*input
6
5
1 2 3 4 5
1 2 3 4 5
3
1 1 2
2 2 2
4
1 2 3 4
3 4 4 4
4
1 2 3 4
1 2 3 3
3
1 3 3
2 2 2
2
1 2
2 1

*/

/*output
Case #1: 0
Case #2: 2
3 1
3 2
Case #3: 3
3 1
4 2
4 3
Case #4: -1
Case #5: -1
Case #6: -1

*/
