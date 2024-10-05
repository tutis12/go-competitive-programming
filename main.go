package main

import (
	"fmt"
	"os"
	"sync"
	"sync/atomic"
	"time"
)

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
			doneCnt := doneCounter.Add(1)
			fmt.Fprintf(
				os.Stderr,
				"test %d (%d/%d) took %s\n",
				i+1,
				doneCnt,
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

type input struct {
	s string
	k int
}

func (i *input) Read() {
	i.s = stdin.NextString()
	i.k = stdin.NextInt()
}

type output struct {
	value int
}

func (o *output) Print() {
	stdout.PutInt(o.value, '\n')
}

const mod = 998244353

type dp struct {
	nInt   int
	nFloat float64
	ways   int

	nInt1   int
	nFloat1 float64
	ways1   int
}

func make1() dp {
	return dp{1, 1, 1, 1, 1, 1}
}

func solve(input input) output {
	s := input.s
	dp := make([][10][2]dp, len(s)+2)
	can := func(i int, c int) bool {
		if i >= len(s) {
			return c == 0
		}
		if s[i] == '?' {
			return true
		}
		return int(s[i]-'0') == c
	}
	dp[len(s)][0][0] = make1()
	dp[len(s)+1][0][0] = make1()
	for i := len(s) - 1; i >= 0; i-- {
		for ci := range 10 {
			if !can(i, ci) {
				continue
			}
			for cj := range 10 {
				if !can(i+1, cj) {
					continue
				}

				for ck := range 10 {
					var nInt int
					var nFloat float64
					nWays := dp[i+1][cj][ck].ways
					if ci >= 1 {
						nInt += dp[i+1][cj][ck].nInt
						nFloat += dp[i+1][cj][ck].nFloat
					}
					if ci*10+cj >= 1 && ci*10+cj <= 26 {

					}
				}
			}
		}
	}
}

const epsilon = 1e-7

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
