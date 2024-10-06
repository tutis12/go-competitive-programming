package hackercup

import (
	"fmt"
	"main/debug"
	"main/fastio"
	"os"
	"sync"
	"sync/atomic"
	"time"
)

func Hackercup(stdin *fastio.Reader, stdout *fastio.Writer) {
	defer stdout.WriteAll()
	defer debug.Recover()
	debug.PrintSeconds()
	tests := stdin.Uint()
	outputs := make([]output, tests)
	wgs := make([]sync.WaitGroup, tests)
	doneCounter := atomic.Uint64{}
	fmt.Fprintf(os.Stderr, "running %d tests\n", tests)
	for i := range tests {
		wgs[i].Add(1)
		input := input{}
		input.Read(stdin)
		go func() {
			defer debug.Recover()
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
		stdout.String("Case #")
		stdout.Uint(i+1, ':')
		stdout.String(" ")
		outputs[i].Print(stdout)
	}
}

type input struct {
	s string
	k int
}

func (i *input) Read(stdin *fastio.Reader) {
	i.s = stdin.String()
	i.k = stdin.Int()
}

type output struct {
	s     string
	value int
}

func (o *output) Print(stdout *fastio.Writer) {
	stdout.String(o.s)
	stdout.String(" ")
	stdout.Int(o.value, '\n')
}

const mod = 998244353

func solve(input *input) output {
	s1 := make([]uint8, len(input.s))
	for i, c := range input.s {
		if c == '?' {
			s1[i] = 1
		} else {
			s1[i] = uint8(c - '0')
		}
	}
	dp1 := make([]bool, len(s1)+1)
	dp2 := make([]bool, len(s1)+1)
	dp1[0] = true
	dp2[len(s1)] = true
	for i := 1; i <= len(s1); i++ {
		if s1[i-1] >= 1 {
			dp1[i] = dp1[i] || dp1[i-1]
		}
		if i >= 2 && s1[i-2] >= 1 && s1[i-2]*10+s1[i-1] <= 26 {
			dp1[i] = dp1[i] || dp1[i-2]
		}
	}
	for i := len(s1) - 1; i >= 0; i-- {
		if s1[i] >= 1 {
			dp2[i] = dp2[i] || dp2[i+1]
		}
		if i <= len(s1)-2 && s1[i] >= 1 && s1[i]*10+s1[i+1] <= 26 {
			dp2[i] = dp2[i] || dp2[i+2]
		}
	}
	good := make([][2]bool, len(s1))
	for i := 0; i < len(s1); i++ {
		if s1[i] >= 1 && s1[i] <= 9 && dp1[i] && dp2[i+1] {
			good[i][0] = true
		}
		if i <= len(s1)-2 && s1[i] >= 1 && s1[i]*10+s1[i+1] <= 26 && dp1[i] && dp2[i+2] {
			good[i][1] = true
		}
	}
	s := input.s
	dp3 := make([][26]int, len(s1))
	for i := len(s) - 1; i >= 0; i-- {
		for c := range 10 {
			if input.s[i] != '0'+byte(c) && input.s[i] != '?' {
				continue
			}
			if good[i][0] {
				if c == 0 {
					continue
				}
			}
			if i == len(s)-1 {
				dp3[i][c] = 1
				continue
			}
			ways := 0
			for c1 := range 10 {
				if good[i][1] {
					if c == 0 || c*10+c1 > 26 {
						continue
					}
				}
				ways += dp3[i+1][c1]
			}
			ways = min(ways, input.k)
			dp3[i][c] = ways
		}
	}
	sBytes := []byte(s)
	for i, c := range s {
		if c == '?' {
			for c := 25; c >= 0; c-- {
				if input.k > dp3[i][c] {
					input.k -= dp3[i][c]
					continue
				}
				sBytes[i] = '0' + byte(c)
				break
			}
		}
	}
	dp4 := make([]int, len(s1)+1)
	dp4[len(s1)] = 1
	for i := len(s1) - 1; i >= 0; i-- {
		if good[i][0] {
			dp4[i] += dp4[i+1]
		}
		if good[i][1] {
			dp4[i] += dp4[i+2]
		}
		if dp4[i] >= mod {
			dp4[i] -= mod
		}
	}
	return output{
		s:     string(sBytes),
		value: dp4[0],
	}
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
