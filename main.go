package main

import (
	"main/fastio"
	"os"
	"slices"
	"sync"
)

var stdout = &fastio.Writer{
	File: os.Stdout,
}

var stdin = &fastio.Reader{
	File: os.Stdin,
}

func init() {
	inputFile, err := os.Open("io/walk_the_line_input.txt")
	if err != nil {
		panic(err.Error())
	}
	stdin.File = inputFile
}

func init() {
	outputFile, err := os.Create("io/output.txt")
	if err != nil {
		panic(err.Error())
	}
	stdout.File = outputFile
}

/*input
6
4 17
1
2
5
10
4 4
1
2
5
10
2 22
22
22
3 1000000000
1000000000
1000000000
1000000000
1 10
12
1 100
12
*/

/*output
Case #1: YES
Case #2: NO
Case #3: YES
Case #4: NO
Case #5: NO
Case #6: YES
*/

type input struct {
	N int
	K int
	S []int
}

type output struct {
	YES bool
}

func (o *input) Read() {
	o.N, o.K = stdin.NextInt2()
	o.S = stdin.NextInts(o.N)
}

func (o *output) Print() {
	if o.YES {
		stdout.PutString("YES\n")
	} else {
		stdout.PutString("NO\n")
	}
}

func main() {
	defer stdout.WriteAll()

	tests := stdin.NextUint()
	outputs := make([]output, tests)
	wgs := make([]sync.WaitGroup, tests)
	for i := range tests {
		wgs[i].Add(1)
		input := input{}
		input.Read()
		go func() {
			outputs[i] = solve(i, input)
			wgs[i].Done()
		}()
	}
	for i := range tests {
		wgs[i].Wait()
		stdout.PutString("Case #")
		stdout.PutUint(i+1, ':')
		stdout.PutString(" ")
		outputs[i].Print()
	}
}

func solve(test uint, input input) output {
	slices.Sort(input.S)
	return output{
		YES: input.S[0]*max(1, input.N*2-3) <= input.K,
	}
}
