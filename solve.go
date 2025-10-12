package main

import (
	"main/fastio"
	"main/segment_tree"
)

func solve(
	stdin *fastio.Reader,
	stdout *fastio.Writer,
) {
	t := stdin.Int()
	for range t {
		solveTest(stdin, stdout)
	}
}

func solveTest(
	stdin *fastio.Reader,
	stdout *fastio.Writer,
) {
	n := stdin.Int()
	a := stdin.Ints(n, 1)
	adj := make([][]int, n+1)
	for range n - 1 {
		u, v := stdin.Int2()
		adj[u] = append(adj[u], v)
		adj[v] = append(adj[v], u)
	}

	from := make([]int, n+1)
	to := make([]int, n+1)
	ordered := make([]int, n*2)
	var rec func(v, p int)
	inc := 0
	rec = func(v, p int) {
		if a[v] == 0 {
			ordered[inc] = 0
		} else {
			ordered[inc] = 2
		}
		from[v] = inc
		inc++
		for _, u := range adj[v] {
			if u == p {
				continue
			}
			rec(u, v)
		}
		if a[v] == 0 {
			ordered[inc] = 1
		} else {
			ordered[inc] = 3
		}
		to[v] = inc
		inc++
	}
	rec(1, -1)

	type seqCount struct {
		max01      int32
		max10_diff int8
	}

	type stValue struct {
		counter01 seqCount
		counter23 seqCount
	}
	type stUpdate struct {
		swap bool
	}

	merge := func(a, b seqCount) seqCount {
		var max_01 int8
		var max_10 int8
		if a.max01%2 != 0 {
			max_01 = b.max10_diff
		}

		if (a.max10_diff%2 == 0) == (a.max01%2 == 0) {
			max_10 = int8(b.max10_diff)
		}

		return seqCount{
			max01:      int32(max_01) + a.max01 + b.max01,
			max10_diff: max_10 + a.max10_diff - max_01,
		}
	}

	st := segment_tree.NewLazyST(
		func(i int) stValue {
			return stValue{
				counter01: seqCount{
					max01:      int32(CastBool(ordered[i] == 0)),
					max10_diff: int8(CastBool(ordered[i] == 1) - CastBool(ordered[i] == 0)),
				},
				counter23: seqCount{
					max01:      int32(CastBool(ordered[i] == 2)),
					max10_diff: int8(CastBool(ordered[i] == 3) - CastBool(ordered[i] == 2)),
				},
			}
		},
		2*n,
		stUpdate{},
		func(a, b stValue) stValue {
			return stValue{
				counter01: merge(a.counter01, b.counter01),
				counter23: merge(a.counter23, b.counter23),
			}
		},
		func(update stUpdate, value *stValue) {
			if update.swap {
				value.counter01, value.counter23 = value.counter23, value.counter01
			}
		},
		func(top stUpdate, being_updated *stUpdate) {
			being_updated.swap = being_updated.swap != top.swap
		},
	)

	stdout.Int(int(st.Total().counter23.max01/2), '\n')

	q := stdin.Int()
	for range q {
		v := stdin.Int()
		st.Update(from[v], to[v], stUpdate{swap: true})
		stdout.Int(int(st.Total().counter23.max01/2), '\n')
	}
}

/*input
2
7
0 1 0 1 1 0 0
1 6
1 7
7 3
3 2
7 5
5 4
4
2
4
6
7
2
0 1
1 2
2
2
1

*/

/*output
2
1
1
2
3
1
0
1


*/

/*


(x(y(z))(t)(w))


*/
