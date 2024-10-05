package main

import (
	"main/fastio"
	"os"
)

var stdout = &fastio.Writer{
	File: os.Stdout,
}

var stdin = &fastio.Reader{
	File: os.Stdin,
}

/*input
5
*/

/*output
5
*/

func main() {
	defer stdout.WriteAll()

	n := stdin.NextInt()
	stdout.PutInt(n, ' ')
}
