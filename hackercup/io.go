package hackercup

import (
	"main/fastio"
)

type input struct {
	N int
}

func (input *input) Read(stdin *fastio.Reader) {
	input.N = stdin.Int()
}

type output struct {
	N int
}

func (output *output) Print(stdout *fastio.Writer) {
	stdout.Int(output.N, '\n')
}
