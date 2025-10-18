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
	defer debug.Recover()
	debug.PrintSeconds()
	tests := stdin.Uint()
	outputs := make([]output, tests)
	testWGs := make([]sync.WaitGroup, tests)

	for i := range tests {
		testWGs[i].Add(1)
	}

	doneCounter := atomic.Uint64{}
	fmt.Fprintf(os.Stderr, "running %d tests\n", tests)
	outputWG := sync.WaitGroup{}
	outputWG.Add(1)
	debug.Go(func() {
		defer stdout.WriteAll()
		defer outputWG.Done()
		for i := range tests {
			testWGs[i].Wait()
			stdout.Printf("Case #%d: ", i+1)
			outputs[i].Print(stdout)
		}
	})
	for i := range tests {
		input := input{}
		input.Read(stdin)
		debug.Go(func() {
			start := time.Now()
			outputs[i] = solve(&input)
			testWGs[i].Done()
			doneCnt := doneCounter.Add(1)
			fmt.Fprintf(
				os.Stderr,
				"test %d (%d/%d) took %s\n",
				i+1,
				doneCnt,
				tests,
				time.Since(start),
			)
		})
	}
	outputWG.Wait()
}
