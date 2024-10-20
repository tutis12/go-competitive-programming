package hackercup

import "fmt"

func solve(input *input) output {
	h := [7]int{}
	minMoves := [2]int{1000, 1000}
	maxMoves := [2]int{-1, -1}
	for h[0] = 0; h[0] <= 6; h[0]++ {
		for h[1] = 0; h[1] <= 6; h[1]++ {
			for h[2] = 0; h[2] <= 6; h[2]++ {
				for h[3] = 0; h[3] <= 6; h[3]++ {
					for h[4] = 0; h[4] <= 6; h[4]++ {
						for h[5] = 0; h[5] <= 6; h[5]++ {
							for h[6] = 0; h[6] <= 6; h[6]++ {
								for lastmove := 0; lastmove < 7; lastmove++ {
									moves, who, ok := check(h, input.table, lastmove)
									if !ok {
										continue
									}
									minMoves[who] = min(minMoves[who], moves)
									maxMoves[who] = max(maxMoves[who], moves)
								}
							}
						}
					}
				}
			}
		}
	}
	if maxMoves[0] == -1 && maxMoves[1] == -1 {
		return output{
			result: "0",
		}
	}
	if maxMoves[0] == -1 {
		return output{
			result: "F",
		}
	}
	if maxMoves[1] == -1 {
		return output{
			result: "C",
		}
	}
	if maxMoves[0] < minMoves[1] {
		return output{
			result: "C",
		}
	}
	if maxMoves[1] < minMoves[0] {
		return output{
			result: "F",
		}
	}
	return output{
		result: "?",
	}
}

func check(h [7]int, table [6]string, lastmove int) (int, int, bool) {
	if h[lastmove] == 0 {
		return 0, 0, false
	}
	moves := 0
	for _, v := range h {
		moves += v
	}
	if moves < 4 {
		return 0, 0, false
	}
	table1 := [6][7]byte{}
	for i, str := range table {
		for j, char := range []byte(str) {
			table1[i][j] = char
		}
	}
	for col := 0; col < 7; col++ {
		for j := 0; j < 6-h[col]; j++ {
			table1[j][col] = '?'
		}
	}

	totalC := 0
	totalF := 0
	for _, s := range table1 {
		for _, i := range s {
			if i == 'C' {
				totalC++
			} else if i == 'F' {
				totalF++
			}
		}
	}
	if totalC+totalF != moves {
		panic("a")
	}
	if totalC-totalF != 0 && totalC-totalF != 1 {
		return 0, 0, false
	}

	if moves%2 == 1 && table1[6-h[lastmove]][lastmove] != 'C' {
		return 0, 0, false
	}
	if moves%2 == 0 && table1[6-h[lastmove]][lastmove] != 'F' {
		return 0, 0, false
	}

	who, ok := check1(table1)
	if !ok {
		return 0, 0, false
	}

	table1[6-h[lastmove]][lastmove] = '?'

	_, ok = check1(table1)
	if ok {
		return 0, 0, false
	}
	h[lastmove]--
	for i, str := range table {
		for j, char := range []byte(str) {
			table1[i][j] = char
		}
	}
	if !check2(table1, h, moves-1) {
		return 0, 0, false
	}
	return moves, who, true
}

var who = []byte("CF")

func check2(table [6][7]byte, h [7]int, moves int) bool {
	w := 0
	g := [7]int{}
	for range moves {
		ok := false
		for i := range 7 {
			if g[i] < h[i] && table[5-g[i]][i] == who[w] {
				g[i]++
				ok = true
				break
			}
		}
		if !ok {
			return false
		}
		w = 1 - w
	}
	for range 42 - moves {
		ok := false
		for i := range 7 {
			if g[i] < 6 && table[5-g[i]][i] == who[w] {
				g[i]++
				ok = true
				break
			}
		}
		if !ok {
			fmt.Println(g, w)
			return false
		}
		w = 1 - w
	}
	return true
}

func check1(table [6][7]byte) (int, bool) {
	for x := 0; x < 6; x++ {
		for y := 0; y < 7; y++ {
			for i := range 2 {
				for dx := -1; dx <= 1; dx++ {
					for dy := -1; dy <= 1; dy++ {
						if dx == 0 && dy == 0 {
							continue
						}
						for t := 0; t < 4; t++ {
							a := x + dx*t
							b := y + dy*t
							if a < 0 || a >= 6 {
								continue
							}
							if b < 0 || b >= 7 {
								continue
							}
							if table[a][b] != who[i] {
								break
							}
							if t == 3 {
								return i, true
							}
						}
					}
				}
			}
		}
	}
	return 0, false
}
