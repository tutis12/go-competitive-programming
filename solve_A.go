package main

import (
	"main/fastio"
	"main/segment_tree_iter"
	"math"
)

var SolveX = SolveA

type intHash int

func (x intHash) Hash() uint64 {
	return uint64(x)
}

type stValue struct {
	minA  int
	minDP int
}

type lazy struct {
	addDP int
}

func (a stValue) Merge(b stValue) stValue {
	return stValue{minA: min(a.minA, b.minA), minDP: min(a.minDP, b.minDP)}
}

func (up lazy) ApplyUpdate(val *stValue) {
	val.minDP += up.addDP
}

func (top lazy) Push(existing *lazy) {
	existing.addDP += top.addDP
}

func SolveA(
	stdin *fastio.Reader,
	stdout *fastio.Writer,
) {
	t := stdin.Int()
	for range t {
		solveATest(stdin, stdout)
	}
}

func solveATest(
	stdin *fastio.Reader,
	stdout *fastio.Writer,
) {

	n := stdin.Int()
	a := stdin.Ints(n, 0)
	st := *segment_tree_iter.NewSegmentTree(
		func(i int) stValue {
			return stValue{
				minA:  a[i],
				minDP: 0,
			}
		},
		n,
		stValue{
			minA:  math.MaxInt,
			minDP: math.MaxInt,
		},
		lazy{},
	)

	dp := make([]int, n)
	for i := range n {
		ai := a[i]
		dp[i] = i + 1
		for cost := 1; cost <= 3; cost++ {
			l, _ := st.LongestRangeWherePredicate(i, func(x stValue) bool {
				return x.minA*cost >= ai
			})
			var total int
			if l == 0 {
				total = cost
			} else {
				total = st.Get(l-1, i-1).minDP + cost
			}
			dp[i] = min(dp[i], total)
		}
		st.SetValue(i, stValue{
			minA:  ai,
			minDP: dp[i],
		})
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
