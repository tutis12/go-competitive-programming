package main

import (
	"main/fastio"
	"main/segment_tree"
)

// var solveX = solveC

func solveC(
	stdin *fastio.Reader,
	stdout *fastio.Writer,
) {
	t := stdin.Int()
	for range t {
		solveTestC(stdin, stdout)
	}
}

func solveTestC(
	stdin *fastio.Reader,
	stdout *fastio.Writer,
) {
	n := stdin.Int()
	a := stdin.Ints(n, 0)
	type stValue struct {
		minA  int
		minDP int
	}
	type lazy struct {
		addDP int
	}
	st := segment_tree.NewLazyST(
		func(i int) stValue {
			return stValue{
				minA:  a[i],
				minDP: 0,
			}
		},
		n,
		lazy{},
		func(a, b stValue) stValue {
			return stValue{
				minA:  min(a.minA, b.minA),
				minDP: min(a.minDP, b.minDP),
			}
		},
		func(lazy lazy, val *stValue) {
			val.minDP += lazy.addDP
		},
		func(top lazy, being_updated *lazy) {
			being_updated.addDP += top.addDP
		},
	)

	dp := make([]int, n)

	for i, ai := range a {
		dp[i] = i + 1
		for cost := 1; cost <= 4; cost++ {
			l, ok := st.BinarySearchOnLeftIndex(i, func(x stValue) bool {
				return x.minA*cost < ai
			})
			if !ok {
				l = 0
			} else {
				l++
			}
			var total int
			if l == 0 {
				total = cost
			} else {
				total = st.Get(l-1, i-1).minDP + cost
			}
			dp[i] = min(dp[i], total)
		}
		st.Update(i, i, lazy{addDP: dp[i]})
	}
	stdout.Int(dp[n-1], '\n')
}

/*input
4
5
3 1 4 1 5
10
9 2 6 5 3 5 8 9 7 9
8
1 2 3 4 5 6 7 8
2
1 1000000000000000000

*/

/*output
2
4
5
2

*/
