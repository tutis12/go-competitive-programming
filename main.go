package main

import (
	"fmt"
	"os"
	"slices"
	"sync"
	"sync/atomic"
	"time"
)

type input struct {
	N int
	K int
	S []int
}

func (o *input) Read() {
	o.N, o.K = stdin.NextInt2()
	o.S = stdin.NextInts(o.N)
}

type output struct {
	YES bool
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
	defer Recover()
	printSeconds()
	tests := stdin.NextUint()
	outputs := make([]output, tests)
	wgs := make([]sync.WaitGroup, tests)
	doneCounter := atomic.Uint64{}
	fmt.Fprintf(os.Stderr, "running %d tests\n", tests)
	for i := range tests {
		wgs[i].Add(1)
		input := input{}
		input.Read()
		go func() {
			defer Recover()
			start := time.Now()
			outputs[i] = solve(input)
			wgs[i].Done()
			doneCounter.Add(1)
			fmt.Fprintf(
				os.Stderr,
				"test %d (%d/%d) took %s\n",
				i+1,
				doneCounter.Load(),
				tests,
				time.Since(start),
			)
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

func solve(input input) output {
	slices.Sort(input.S)
	return output{
		YES: input.S[0]*max(1, input.N*2-3) <= input.K,
	}
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
