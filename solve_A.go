package main

import (
	"main/fastio"
	"math"
)

var SolveX = SolveA

type intHash int

func (x intHash) Hash() uint64 {
	return uint64(x)
}

type stValue struct {
	minA int
}

type lazy struct {
}

type controller struct {
}

func (controller) Merge(a, b stValue) stValue {
	return stValue{minA: min(a.minA, b.minA)}
}

func (controller) ZeroValue() stValue {
	return stValue{
		minA: math.MaxInt64,
	}
}

func (controller) ZeroUpdate() lazy {
	return lazy{}
}
func (controller) ApplyUpdate(update lazy, val *stValue) {
}

func (controller) Push(update lazy, existing *lazy) {
}

var globalCost int
var globalAi int

func (controller) Predicate(x stValue) bool {
	return x.minA*globalCost >= globalAi
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
	const maxDP = 128
	dp := [maxDP + 1]int{}
	for i := range dp {
		dp[i] = -1
	}
	stack := make([]int, 0, n+1)
	stack = append(stack, -1)
	for i := range n {
		ai := a[i]
		dpI := i + 1
		dpJ := 0
		for len(stack) != 1 && a[stack[len(stack)-1]] >= ai {
			stack = stack[:len(stack)-1]
		}
		stack = append(stack, i)
		for cost := 3; cost >= 1; cost-- {
			globalCost = cost
			globalAi = ai
			var l int
			if cost != 1 {
				lo := 0
				hi := len(stack) - 1
				for lo < hi {
					mid := (lo + hi + 1) / 2
					if a[stack[mid]]*cost < ai {
						lo = mid
					} else {
						hi = mid - 1
					}
				}
				l = stack[lo] + 1
			} else {
				l = stack[len(stack)-2] + 1
			}
			var total int
			if l == 0 {
				total = cost
			} else {
				for dp[dpJ] < l-1 {
					dpJ++
				}
				total = dpJ + cost
			}
			dpI = min(dpI, total)
		}
		for j := dpI; j <= maxDP; j++ {
			dp[j] = i
		}
	}
	ans := 0
	for dp[ans] < n-1 {
		ans++
	}
	stdout.Int(ans, '\n')
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
