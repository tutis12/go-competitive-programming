package main

import (
	"main/debug"
	"main/fastio"
	"os"
)

const (
	fromFile   = false
	inputFile  = "four_in_a_burrow_input (1).txt"
	outputFile = "output.txt"
)

/*input
3
6 10
1 5 7 8 11 12
6
1 6
1 5
2 6
1 4
2 5
3 6
6 1
1 1 1 3 3 3
2
3 3
1 6
12 15
4 5 15 24 27 32 36 39 40 46 48 48
20
1 12
1 11
6 10
1 8
8 12
11 12
2 9
3 8
7 8
7 10
4 8
9 12
9 10
2 12
1 5
3 12
4 8
3 7
7 12
10 11

*/

/*output
3
2
2
2
2
2
1
4
6
6
2
4
2
2
5
3
2
2
2
2
2
6
4
5
2
3
2
2

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
	solve(stdin, stdout)
}
