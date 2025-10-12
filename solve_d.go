package main

import (
	"main/fastio"
)

const (
	mod = 1000000007
)

//var solveX = solveD

func solveD(
	stdin *fastio.Reader,
	stdout *fastio.Writer,
) {
	t := stdin.Int()
	for range t {
		solveTestD(stdin, stdout)
	}
}

func solveTestD(
	stdin *fastio.Reader,
	stdout *fastio.Writer,
) {
	n := stdin.Int()
	a := stdin.Ints(n+1, 0)
	answer := count(n, a)
	a[n] = 0
	answer -= count(n, a)
	answer += mod
	answer %= mod
	stdout.Int(answer, '\n')
}

func count(n int, a []int) int {
	for i := range n + 1 {
		if a[i] < -1 || a[i] > n {
			return 0
		}
	}
	cnt := make([]int, n+1)
	for i := range n + 1 {
		if a[i] != -1 {
			cnt[a[i]]++
		}
	}
	for i := 1; i <= n; i++ {
		if cnt[i] >= 2 {
			return 0
		}
	}
	for i := 1; i <= n; i++ {
		if cnt[i] != 0 && a[i] == 0 {
			return 0
		}
	}
	c := 0
	for i := 1; i <= n; i++ {
		if cnt[i] == 0 {
			c++
		}
	}
	ret := 1
	for i := 1; i <= c+1; i++ {
		ret *= i
		ret %= mod
	}
	return ret
}

/*input
6
1
-1 -1
2
-1 2 -1
2
-1 -1 -1
3
-1 -1 3 -1
3
-1 2 3 -1
5
-1 -1 -1 1 0 -1

*/

/*output
1
1
3
2
0
3

*/
