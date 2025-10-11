package main

import (
	"main/fastio"
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

const log = 18

func solveTest(
	stdin *fastio.Reader,
	stdout *fastio.Writer,
) {
	n, z := stdin.Uint32_2()
	x := stdin.Uint32s(int(n))
	q := stdin.Int()
	jump := [log][]uint32{}
	for i := range log {
		jump[i] = make([]uint32, n+1)
	}
	jump[0][n] = n
	{
		j := uint32(1)
		for i := range n {
			for j < n && x[j]-x[i] <= z {
				j++
			}
			jump[0][i] = j
		}
	}

	for j := 1; j < log; j++ {
		for i := range n + 1 {
			jump[j][i] = jump[j-1][jump[j-1][i]]
		}
	}

	type big struct {
		pos   uint32
		jumps uint32
	}
	jumpBig := [log][]big{}
	for i := range log {
		jumpBig[i] = make([]big, n+1)
	}
	for pos := range n {
		i := pos
		j := i + 1
		cnt := uint32(1)
		for t := log - 1; t >= 0; t-- {
			i1 := jump[t][i]
			i2 := jump[t][j]
			i3 := jump[0][i1]
			if i1 != i2 && i2 != i3 {
				i, j = i1, i2
				cnt += 1 << (t + 1)
			}
		}

		{
			k := jump[0][i]

			if k != j {
				cnt++
				k1 := jump[0][j]
				_, j = j, k

				{
					k := k1
					if k != j {
						cnt++
						_, j = j, k

						{
							k := jump[1][i]
							if k != j {
								cnt++
								_, j = j, k
							}
						}

					}
				}

			}
		}

		jumpBig[0][pos] = big{j, cnt}
	}
	jumpBig[0][n] = big{n, 0}
	for j := 1; j < log; j++ {
		for i := range n + 1 {
			b1 := jumpBig[j-1][i]
			b2 := jumpBig[j-1][b1.pos]
			jumpBig[j][i] = big{b2.pos, b1.jumps + b2.jumps}
		}
	}
	for range q {
		l, r := stdin.Uint32_2()
		l--
		r--

		i := l

		cnt := uint32(1)
		for t := log - 1; t >= 0; t-- {
			b := jumpBig[t][i]
			if b.pos <= r {
				i = b.pos
				cnt += b.jumps
			}
		}

		j := i + 1
		for t := log - 1; t >= 0; t-- {
			i1 := jump[t][i]
			i2 := jump[t][j]
			i3 := jump[0][i1]
			if i1 != i2 && i2 != i3 && i2 <= r {
				i, j = i1, i2
				cnt += 1 << (t + 1)
			}
		}
		if j <= r {
			cnt++
			k := jump[0][i]
			j0 := j
			if k == j {
				_, j = j, j+1
			} else {
				_, j = j, k
			}

			if j <= r {
				cnt++
				k := jump[0][j0]
				if k == j {
					_, j = j, j+1
				} else {
					_, j = j, k
				}

				if j <= r {
					cnt++
				}
			}
		}
		stdout.Uint32(cnt, '\n')
	}
}
