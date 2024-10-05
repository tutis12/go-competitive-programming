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
			outputs[i] = solve(&input)
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
	n int
	s []string
}

func (i *input) Read() {
	i.n = stdin.NextInt()
	i.s = make([]string, i.n)
	for index := range i.n {
		i.s[index] = stdin.NextString()
	}
}

type output struct {
	value int
}

func (o *output) Print() {
	stdout.PutInt(o.value, '\n')
}

const mod = 998244353

func solve(input *input) output {
	return output{
		value: (1 + solveMask((1<<input.n)-1, 0, input.s, &dp{
			dp: make([][]int, 1<<input.n),
		})) % mod,
	}
}

type dp struct {
	dp [][]int
}

func solveMask(mask uint, prefix int, strings []string, dp *dp) int {
	if mask == 0 {
		return 0
	}
	dp1 := dp.dp[mask]
	if prefix < len(dp1) && dp1[prefix] != -1 {
		return dp1[prefix]
	}
	allQ := true
	for i := 0; i < 25; i++ {
		if (mask & (1 << i)) == 0 {
			continue
		}
		if len(strings[i]) <= prefix || strings[i][prefix] != '?' {
			allQ = false
		}
	}
	if allQ {
		answer := (26 + 26*solveMask(mask, prefix+1, strings, dp)) % mod
		dp1 := dp.dp[mask]
		for prefix >= len(dp1) {
			dp1 = append(dp1, -1)
		}
		dp1[prefix] = answer
		dp.dp[mask] = dp1
		return answer
	}
	answer := 0
	for c := byte('A'); c <= byte('Z'); c++ {
		matchMask := uint(0)
		for i := 0; i < 25; i++ {
			if (mask & (1 << i)) == 0 {
				continue
			}
			if len(strings[i]) > prefix && (strings[i][prefix] == c || strings[i][prefix] == '?') {
				matchMask += 1 << i
			}
		}
		if matchMask == 0 {
			continue
		}
		answer++
		answer += solveMask(matchMask, prefix+1, strings, dp)
		answer %= mod
	}
	dp1 = dp.dp[mask]
	for prefix >= len(dp1) {
		dp1 = append(dp1, -1)
	}
	dp1[prefix] = answer
	dp.dp[mask] = dp1
	return answer
}

/*input
5
2
META
MATE
2
?B
AC
1
??
3
XXY
X?
?X
2
??M?E?T?A??
?M?E?T?A?

*/

/*output
Case #1: 8
Case #2: 54
Case #3: 703
Case #4: 79
Case #5: 392316013

*/
