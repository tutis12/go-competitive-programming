package main

import (
	"main/fastio"
)

//var solveX = solveB

func solveB(
	stdin *fastio.Reader,
	stdout *fastio.Writer,
) {
	t := stdin.Int()
	for range t {
		solveTestB(stdin, stdout)
	}
}

func solveTestB(
	stdin *fastio.Reader,
	stdout *fastio.Writer,
) {

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
