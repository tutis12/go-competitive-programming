package main

import (
	"main/fastio"
	"slices"
)

var solveX = solveF

func solveF(
	stdin *fastio.Reader,
	stdout *fastio.Writer,
) {
	t := stdin.Int()
	for range t {
		solveTestF(stdin, stdout)
	}
}

func solveTestF(
	stdin *fastio.Reader,
	stdout *fastio.Writer,
) {
	n, m := stdin.Int2()
	queryMap = make([][]int, n+1)
	maxT := 2*n - 1
	for i := range queryMap {
		queryMap[i] = make([]int, maxT+1)
	}
	g := make([][]int, n)
	for i := range n {
		g[i] = stdin.Ints(n, 0)
	}
	{
		max1 := 0
		for j := range n {
			max1 = max(max1, g[0][j])
			queryMap[j+1][1] = max1
		}
	}
	{
		max2 := g[1][0]
		for j := range n {
			queryMap[j+1][2] = max2
			max2 = max(max2, g[0][j])
		}
	}
	time := make([]int, n*n+1)
	for i := range n {
		for j := range n {
			if i == 0 {
				time[g[i][j]] = 1 - i
			} else {
				time[g[i][j]] = 1 + i + j
			}
		}
	}
	toCheck := make([][]queryKey, n*n+1)
	inserted := make([][]bool, n+1)
	for i := range inserted {
		inserted[i] = make([]bool, maxT+1)
	}
	insertToCheck := func(length, t int) {
		if t < 1 || t > maxT {
			return
		}
		if inserted[length][t] {
			return
		}
		inserted[length][t] = true
		val := query(stdin, stdout, length, t)
		toCheck[val] = append(toCheck[val], queryKey{length, t})
	}
	for length := 1; length <= n; length++ {
		findValley := func(left, right int) {
			if left <= 0 {
				return
			}
			if right > maxT {
				return
			}
			if left > right {
				return
			}
			for right-left+1 >= 3 {
				mid := (left + right) / 2
				ans := query(stdin, stdout, length, mid)
				insertToCheck(length, mid)
				time := time[ans]
				t1 := time - 1
				if t1 >= left && t1 < mid && query(stdin, stdout, length, t1) < ans {
					right = mid - 1
				} else {
					left = mid + 1
				}
			}
			for t := left; t <= right; t++ {
				insertToCheck(length, t)
			}
		}
		peaks := make(map[int]struct{}, 0)
		peaks[1] = struct{}{}
		peaks[maxT] = struct{}{}
		for t := 1; t <= maxT+length-1; t += length {
			t1 := min(t, maxT)
			ans1 := time[query(stdin, stdout, length, t1)]
			peaks[ans1] = struct{}{}
			peaks[ans1+length-1] = struct{}{}
		}
		peaksArr := CollectMap(peaks)
		slices.Sort(peaksArr)
		for i := 1; i < len(peaksArr); i++ {
			findValley(peaksArr[i-1], peaksArr[i])
		}
		for i := 0; i < len(peaksArr); i++ {
			insertToCheck(length, peaksArr[i])
		}
	}
	answer := make([]int, m)
	checked := make(map[queryKey]struct{}, 0)
	minVal := 0
	for i := range m {
		for {
			if len(toCheck[minVal]) > 0 {
				break
			} else {
				minVal++
			}
		}
		key := toCheck[minVal][0]
		toCheck[minVal] = toCheck[minVal][1:]
		if len(toCheck) == 0 {
			panic("ran out of toCheck")
		}
		answer[i] = minVal
		checked[key] = struct{}{}
		for _, dt := range [2]int{-1, 1} {
			neighbor := queryKey{key.L, key.T + dt}
			if neighbor.T < 1 || neighbor.T > maxT {
				continue
			}
			_, ok := checked[neighbor]
			if !ok {
				insertToCheck(neighbor.L, neighbor.T)
			}
		}
	}
	stdout.String("! ")
	stdout.Ints(answer, ' ', '\n')
	stdout.WriteAll()
}

type queryKey struct {
	L int
	T int
}

var queryMap [][]int

func query(
	stdin *fastio.Reader,
	stdout *fastio.Writer,
	L int,
	T int,
) int {
	res := queryMap[L][T]
	if res != 0 {
		return res
	}
	stdout.Fprintf("? %d %d\n", L, T)
	stdout.WriteAll()
	res = stdin.Int()
	queryMap[L][T] = res
	return res
}

/*input
1
3 15
4 2 5
1 9 3
7 6 8

4

1

9

6

8

4

4

7

7

8

5

4

9

9

9


*/

/*output





? 1 1

? 1 2

? 1 3

? 1 4

? 1 5

? 2 1

? 2 2

? 2 3

? 2 4

? 2 5

? 3 1

? 3 2

? 3 3

? 3 4

? 3 5

! 1 4 4 4 4 5 6 7 7 8 8 9 9 9 9

*/
