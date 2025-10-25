package main

import (
	"fmt"
	"os"
	"runtime"
	"sync"
	"sync/atomic"
	"time"
)

func main() {
	var stdin = &Reader{
		File: os.Stdin,
	}
	var stdout = &Writer{
		File: os.Stdout,
	}

	if fromFile {
		outputFile, err := os.Create("io/output" + inputFile)
		if err != nil {
			panic(err.Error())
		}
		stdout.File = outputFile

		inputFile, err := os.Open("io/" + inputFile)
		if err != nil {
			panic(err.Error())
		}
		stdin.File = inputFile
	}
	defer stdout.WriteAll()
	defer Recover()

	Hackercup(stdin, stdout)
}

type (
	Reader struct {
		File  *os.File
		bytes [buffSize]byte
		from  int
		to    int
	}

	Writer struct {
		File      *os.File
		buffer    [buffSize]byte
		intBuffer [maxIntSize]byte
		used      int
	}
)

const (
	fromFile  = false
	inputFile = "crash_course_input.txt"
)

func (w *Writer) WriteAll() {
	n, err := w.File.Write(w.buffer[:w.used])
	if n != w.used {
		panic("failed to write: " + err.Error())
	}
	w.used = 0
}

func Recover() {
	err := recover()
	if err == nil {
		return
	}
	defer os.Exit(13)

	buf := make([]byte, 10000)
	n := runtime.Stack(buf, false)
	buf = buf[:n]
	fmt.Fprintf(os.Stderr, "panic: %v\nstacktrace:\n%s", err, string(buf))
}

func Hackercup(stdin *Reader, stdout *Writer) {
	start := time.Now()
	defer Recover()
	defer func() {
		stdout.WriteAll()
		time.Sleep(time.Millisecond)
		fmt.Fprintf(os.Stderr, "done! took %s\n", time.Since(start))
	}()

	PrintSeconds()
	tests := stdin.Uint()
	outputs := make([]output, tests)
	testWGs := make([]sync.WaitGroup, tests)

	for i := range tests {
		testWGs[i].Add(1)
	}

	doneCounter := atomic.Uint64{}
	fmt.Fprintf(os.Stderr, "running %d tests\n", tests)
	Go(func() {
		for i := range tests {
			input := input{}
			input.Read(stdin)
			Go(func() {
				start := time.Now()
				outputs[i] = solve(int(i), &input)
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
	})

	for i := range tests {
		testWGs[i].Wait()
		stdout.Printf("Case #%d: ", i+1)
		outputs[i].Print(stdout)
	}
}

func PrintSeconds() {
	start := time.Now()
	ticker := time.NewTicker(time.Second * 10)
	go func() {
		defer Recover()
		for range ticker.C {
			fmt.Fprintf(os.Stderr, "%ds passed\n", (time.Since(start)+time.Second/2)/time.Second)
		}
	}()
}

func (r *Reader) Uint() uint {
	n := uint(0)
	for {
		c, ok := r.peek()
		if !ok {
			return 0
		}
		r.seek()
		if '0' <= c && c <= '9' {
			n = uint(c - '0')
			break
		}
	}

	for {
		c, ok := r.peek()
		if !ok {
			break
		}
		r.seek()
		if c < '0' || c > '9' {
			break
		}
		n = n*10 + uint(c-'0')
	}
	return n
}

func (w *Writer) Uint(n uint, c byte) {
	i := maxIntSize - 1
	w.intBuffer[i] = c
	i--
	if n == 0 {
		w.intBuffer[i] = '0'
		i--
	}
	for n != 0 {
		w.intBuffer[i] = '0' + byte(n%10)
		n /= 10
		i--
	}
	w.bytes(w.intBuffer[i+1:])
}

func Go(fn func()) {
	go func() {
		defer Recover()
		fn()
	}()
}

type input struct {
	N int
	S string
}

func (input *input) Read(stdin *Reader) {
	input.N = stdin.Int()
	input.S = stdin.String()
}

func solve(test int, input *input) output {
	s := []byte("X" + input.S + "X")
	N := input.N
	al := make([]int, N+2)
	br := make([]int, N+2)
	queue := make([][]PairG1intG2int, N+4)
	for i := 0; i < N+2; i++ {
		al[i] = 0
		br[i] = N + 1
		queue[i+1] = append(queue[i+1], PairG1intG2int{X: 0, Y: i})
		queue[N+2-i] = append(queue[N+2-i], PairG1intG2int{X: 1, Y: i})
	}
	for i := 1; i <= N; i++ {
		if s[i] == 'A' {
			al[i] = i
			queue[1] = append(queue[1], PairG1intG2int{X: 0, Y: i})
		} else {
			br[i] = i
			queue[1] = append(queue[1], PairG1intG2int{X: 1, Y: i})
		}
	}
	for dist := 1; dist <= N; dist++ {
		for len(queue[dist]) > 0 {
			el := queue[dist][0]
			queue[dist] = queue[dist][1:]
			if el.X == 0 {
				r := el.Y
				l := al[r]
				if r-l+1 != dist {
					continue
				}
				if r == N+1 {
					continue
				}
				if s[r+1] == 'A' {
					if al[r+1] < l {
						al[r+1] = l
						queue[dist+1] = append(queue[dist+1], PairG1intG2int{X: 0, Y: r + 1})
					}
				} else if l != r {
					if br[l+1] > r+1 {
						br[l+1] = r + 1
						queue[dist] = append(queue[dist], PairG1intG2int{X: 1, Y: l + 1})
					}
				}
			} else {
				l := el.Y
				r := br[l]
				if r-l+1 != dist {
					continue
				}
				if l == 0 {
					continue
				}
				if s[l-1] == 'B' {
					if br[l-1] > r {
						br[l-1] = r
						queue[dist+1] = append(queue[dist+1], PairG1intG2int{X: 1, Y: l - 1})
					}
				} else if l != r {
					if al[r-1] < l-1 {
						al[r-1] = l - 1
						queue[dist] = append(queue[dist], PairG1intG2int{X: 0, Y: r - 1})
					}
				}
			}
		}
	}

	return output{
		alice: al[N] >= 1,
	}
}

func (w *Writer) Printf(format string, a ...any) {
	w.String(fmt.Sprintf(format, a...))
}

type output struct {
	alice bool
}

func (output *output) Print(stdout *Writer) {
	if output.alice {
		stdout.String("Alice\n")
	} else {
		stdout.String("Bob\n")
	}
}

func (r *Reader) String() string {
	res := []byte{}
	afterWhite := false
	if r.from == r.to {
		if !r.read() {
			return ""
		}
	}
	for {
		fr := r.from
		for i := r.from; i < r.to; i++ {
			r.from = i + 1
			if isWhite(r.bytes[i]) {
				if afterWhite {
					res = append(res, r.bytes[fr:i]...)
					return string(res)
				} else {
					fr = i + 1
				}
			} else {
				afterWhite = true
			}
		}
		res = append(res, r.bytes[fr:r.to]...)
		if !r.read() {
			break
		}
	}
	return string(res)
}

func (w *Writer) String(s string) {
	fr := 0
	to := len(s)
	for fr < to {
		left := buffSize - w.used
		toCopy := min(left, to-fr)
		copy(w.buffer[w.used:w.used+toCopy], s[fr:fr+toCopy])
		fr += toCopy
		w.used += toCopy
		if w.used >= buffSize-maxIntSize {
			w.WriteAll()
		}
	}
}

func (r *Reader) read() bool {
	n, _ := r.File.Read(r.bytes[:])
	if n == 0 {
		return false
	} else {
		r.from = 0
		r.to = n
		return true
	}
}

func isWhite(c byte) bool {
	return c == ' ' || c == '\n' || c == '\r' || c == '\t'
}

const (
	buffSize = 100000

	maxIntSize = 128
)

type PairG1intG2int struct {
	X int

	Y int
}

func (r *Reader) Int() int {
	pos := true
	n := 0
	for {
		c, ok := r.peek()
		if !ok {
			return 0
		}
		r.seek()
		if '0' <= c && c <= '9' {
			n = int(c - '0')
			break
		}
		if c == '-' {
			pos = false
			break
		}
	}

	for {
		c, ok := r.peek()
		if !ok {
			break
		}
		r.seek()
		if c < '0' || c > '9' {
			break
		}
		n = n*10 + int(c-'0')
	}
	if !pos {
		n = -n
	}
	return n
}

func (w *Writer) Int(value int, c byte) {
	pos := true
	var n uint
	if value < 0 {
		pos = false
		n = uint(-value)
	} else {
		n = uint(value)
	}
	i := maxIntSize - 1
	w.intBuffer[i] = c
	i--
	if n == 0 {
		w.intBuffer[i] = '0'
		i--
	}
	for n != 0 {
		w.intBuffer[i] = '0' + byte(n%10)
		n /= 10
		i--
	}
	if !pos {
		w.intBuffer[i] = '-'
		i--
	}
	w.bytes(w.intBuffer[i+1:])
}

func (r *Reader) peek() (byte, bool) {
	if r.from == r.to {
		if !r.read() {
			return 0, false
		}
	}
	return r.bytes[r.from], true
}

func (r *Reader) seek() {
	r.from++
}

func (w *Writer) bytes(c []byte) {
	if len(c) >= maxIntSize {
		panic("bytes too long")
	}
	copy(w.buffer[w.used:], c)
	w.used += len(c)
	if w.used >= buffSize-maxIntSize {
		w.WriteAll()
	}
}

