package hackercup

import "main/fastio"

type input struct {
	n int
}

func (i *input) Read(stdin *fastio.Reader) {
	i.n = stdin.Int()
}

type output struct {
	n int
}

func (o *output) Print(stdout *fastio.Writer) {
	stdout.Int(o.n, '\n')
}
