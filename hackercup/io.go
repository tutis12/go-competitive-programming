package hackercup

import (
	"main/fastio"
)

type input struct {
	N int
	S string
}

func (input *input) Read(stdin *fastio.Reader) {
	input.N = stdin.Int()
	input.S = stdin.String()
}

type output struct {
	alice bool
}

func (output *output) Print(stdout *fastio.Writer) {
	if output.alice {
		stdout.String("Alice\n")
	} else {
		stdout.String("Bob\n")
	}
}
