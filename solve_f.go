package main

import (
	"main/fastio"
)

// var solveX = solveF

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
