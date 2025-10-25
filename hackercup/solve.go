package hackercup

import (
	"main/utils"
)

func solve(test int, input *input) output {
	s := []byte("X" + input.S + "X")
	N := input.N
	al := make([]int, N+2)
	br := make([]int, N+2)
	queue := make([][]utils.Pair[int, int], N+4)
	for i := 0; i < N+2; i++ {
		al[i] = 0
		br[i] = N + 1
		queue[i+1] = append(queue[i+1], utils.Pair[int, int]{X: 0, Y: i})
		queue[N+2-i] = append(queue[N+2-i], utils.Pair[int, int]{X: 1, Y: i})
	}
	for i := 1; i <= N; i++ {
		if s[i] == 'A' {
			al[i] = i
			queue[1] = append(queue[1], utils.Pair[int, int]{X: 0, Y: i})
		} else {
			br[i] = i
			queue[1] = append(queue[1], utils.Pair[int, int]{X: 1, Y: i})
		}
	}
	for dist := 1; dist <= N; dist++ {
		for len(queue[dist]) > 0 {
			el := queue[dist][0]
			queue[dist] = queue[dist][1:]
			if el.X == 0 {
				r := el.Y
				l := al[r]
				if r-l+1 != dist {
					continue
				}
				if r == N+1 {
					continue
				}
				if s[r+1] == 'A' {
					if al[r+1] < l {
						al[r+1] = l
						queue[dist+1] = append(queue[dist+1], utils.Pair[int, int]{X: 0, Y: r + 1})
					}
				} else if l != r {
					if br[l+1] > r+1 {
						br[l+1] = r + 1
						queue[dist] = append(queue[dist], utils.Pair[int, int]{X: 1, Y: l + 1})
					}
				}
			} else {
				l := el.Y
				r := br[l]
				if r-l+1 != dist {
					continue
				}
				if l == 0 {
					continue
				}
				if s[l-1] == 'B' {
					if br[l-1] > r {
						br[l-1] = r
						queue[dist+1] = append(queue[dist+1], utils.Pair[int, int]{X: 1, Y: l - 1})
					}
				} else if l != r {
					if al[r-1] < l-1 {
						al[r-1] = l - 1
						queue[dist] = append(queue[dist], utils.Pair[int, int]{X: 0, Y: r - 1})
					}
				}
			}
		}
	}

	return output{
		alice: al[N] >= 1,
	}
}

func arit(a, b int) int {
	avg2 := (a + b)
	return (avg2 * (b - a + 1)) / 2
}

func matrixPower(m [][]int, p int) [][]int {
	n := len(m)
	ret := make([][]int, n)
	for i := range n {
		ret[i] = make([]int, n)
		ret[i][i] = 1
	}
	for p != 0 {
		if p%2 == 1 {
			ret = matrixMultiply(ret, m)
		}
		m = matrixMultiply(m, m)
		p /= 2
	}
	return ret
}

func matrixMultiply(a, b [][]int) [][]int {
	n := len(a)
	result := make([][]int, n)
	for i := range n {
		result[i] = make([]int, n)
	}
	for i := range n {
		for j := range n {
			if a[i][j] == 0 {
				continue
			}
			for k := range n {
				result[i][k] += a[i][j] * b[j][k]
				if result[i][k] >= mod {
					result[i][k] %= mod
				}
			}
		}
	}
	return result
}

const mod = 1_000_000_007
