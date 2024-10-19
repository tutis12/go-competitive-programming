package hackercup

import (
	"fmt"
	"main/debug"
	"main/fastio"
	"os"
	"sync"
	"sync/atomic"
	"time"
)

func Hackercup(stdin *fastio.Reader, stdout *fastio.Writer) {
	defer stdout.WriteAll()
	defer debug.Recover()
	debug.PrintSeconds()
	tests := stdin.Uint()
	outputs := make([]output, tests)
	wgs := make([]sync.WaitGroup, tests)
	doneCounter := atomic.Uint64{}
	fmt.Fprintf(os.Stderr, "running %d tests\n", tests)
	for i := range tests {
		wgs[i].Add(1)
		input := input{}
		input.Read(stdin)
		go func() {
			defer debug.Recover()
			start := time.Now()
			outputs[i] = solve(&input)
			wgs[i].Done()
			doneCnt := doneCounter.Add(1)
			fmt.Fprintf(
				os.Stderr,
				"test %d (%d/%d) took %s\n",
				i+1,
				doneCnt,
				tests,
				time.Since(start),
			)
		}()
	}
	for i := range tests {
		wgs[i].Wait()
		stdout.String("Case #")
		stdout.Uint(i+1, ':')
		stdout.String(" ")
		outputs[i].Print(stdout)
	}
}
