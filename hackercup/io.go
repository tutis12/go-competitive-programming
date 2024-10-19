package hackercup

import "main/fastio"

type input struct {
	n int
}

func (input *input) Read(stdin *fastio.Reader) {
	input.n = stdin.Int()
}

type output struct {
	n int
}

func (output *output) Print(stdout *fastio.Writer) {
	stdout.Int(output.n, '\n')
}
