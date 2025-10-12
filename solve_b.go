package main

import (
	"main/fastio"
)

// var solveX = solveB

func solveB(
	stdin *fastio.Reader,
	stdout *fastio.Writer,
) {
	t := stdin.Int()
	for range t {
		solveTestB(stdin, stdout)
	}
}

const inf = 1_000_000

func solveTestB(
	stdin *fastio.Reader,
	stdout *fastio.Writer,
) {
	n, m := stdin.Int2()
	G := make([][]int, n)
	for i := 0; i < n; i++ {
		G[i] = make([]int, m)
		s := stdin.String()
		for j := 0; j < m; j++ {
			G[i][j] = CastBool(s[j] == '1')
		}
	}
	flip := false
	if n > m {
		n, m = m, n
		flip = true
		Transpose(&G)
	}
	answer := make([][]int, n)
	for i := 0; i < n; i++ {
		answer[i] = make([]int, m)
		for j := 0; j < m; j++ {
			answer[i][j] = inf
		}
	}

	answerI1 := make([][]int, n)
	for i := 0; i < n; i++ {
		answerI1[i] = make([]int, m)
	}
	for i1 := 0; i1 < n; i1++ {
		for i := 0; i < n; i++ {
			for j := 0; j < m; j++ {
				answerI1[i][j] = inf
			}
		}
		for i2 := i1 + 1; i2 < n; i2++ {
			diffI := i2 - i1 + 1
			lastj := -1
			for j := 0; j < m; j++ {
				if G[i1][j] == 1 && G[i2][j] == 1 {
					if lastj != -1 {
						diffJ := j - lastj + 1
						area := diffI * diffJ
						for jj := lastj; jj <= j; jj++ {
							answerI1[i2][jj] = min(answerI1[i2][jj], area)
						}
					}
					lastj = j
				}
			}
		}

		//stdout.Fprintf("%d:\n %v\n", i1, answerI1)
		for j := 0; j < m; j++ {
			ans := inf
			for i := n - 1; i >= i1; i-- {
				ans = min(ans, answerI1[i][j])
				answer[i][j] = min(answer[i][j], ans)
			}
		}
	}

	if flip {
		Transpose(&answer)
	}
	for _, v := range answer {
		for _, v := range v {
			if v >= inf {
				v = 0
			}
			stdout.Int(v, ' ')
		}
		stdout.Char('\n')
	}
}

/*input
3
3 5
10101
10100
00101
4 6
011101
010001
100010
101110
5 5
11100
10110
11111
01101
00111
*/

/*output
6 6 6 9 9
6 6 6 9 9
0 0 9 9 9
0 10 8 8 10 10
0 10 8 8 10 10
10 10 8 8 10 0
10 10 8 8 10 0
6 6 6 0 0
6 6 4 4 0
6 4 4 4 6
0 4 4 6 6
0 0 6 6 6

*/
