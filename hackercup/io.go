package hackercup

import "main/fastio"

type input struct {
	table [6]string
}

func (input *input) Read(stdin *fastio.Reader) {
	input.table = [6]string(stdin.Strings(6))
}

type output struct {
	result string
}

func (output *output) Print(stdout *fastio.Writer) {
	stdout.String(output.result)
	stdout.String("\n")
}
