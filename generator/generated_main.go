package main

import (
	"fmt"
	"os"
	"runtime"
	"strconv"
	"sync"
	"sync/atomic"
	"time"
)

//package main
//file ..//go

const (
	fromFile   = true
	inputFile  = "bunny_hopscotch_validation_input.txt"
	outputFile = "output.txt"
)

/*input
4
1 3 3
1 1 2
1 4 12
1 2 3 4
2 2 5
1 2
2 1
2 3 17
1 1 2
1 2 2

*/

/*output
Case #1: 2
Case #2: 3
Case #3: 1
Case #4: 2

*/

func main() {
	var stdin = &Reader{
		File: os.Stdin,
	}
	var stdout = &Writer{
		File: os.Stdout,
	}

	if fromFile {
		inputFile, err := os.Open("io/" + inputFile)
		if err != nil {
			panic(err.Error())
		}
		stdin.File = inputFile

		outputFile, err := os.Create("io/" + outputFile)
		if err != nil {
			panic(err.Error())
		}
		stdout.File = outputFile
	}

	defer stdout.WriteAll()
	defer Recover()
	Hackercup(stdin, stdout)
}

//package hackercup
//file ..//hackercup/go

func Hackercup(stdin *Reader, stdout *Writer) {
	defer stdout.WriteAll()
	defer Recover()
	PrintSeconds()
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
			defer Recover()
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

//package hackercup
//file ..//hackercup/io.go

type input struct {
	R, C, K int
	table   [][]int
}

func (input *input) Read(stdin *Reader) {
	input.R, input.C, input.K = stdin.Int3()
	input.table = make([][]int, input.R)
	for i := range input.R {
		input.table[i] = stdin.Ints(input.C)
	}
}

type output struct {
	result int
}

func (output *output) Print(stdout *Writer) {
	stdout.Int(output.result, '\n')
}

//package hackercup
//file ..//hackercup/solve.go

type coord struct {
	i, j int
}

func solve(input *input) output {
	N := max(input.R, input.C)
	ways := make([]atomic.Int64, N)
	for dx := -input.R + 1; dx <= input.R-1; dx++ {
		for dy := -input.C + 1; dy <= input.C-1; dy++ {
			ways[max(abs(dx), abs(dy))].Add(int64((input.R - abs(dx)) * (input.C - abs(dy))))
		}
	}
	RC := input.R * input.C
	V := make([][]coord, RC+1)
	for i := range input.R {
		for j := range input.C {
			V[input.table[i][j]] = append(V[input.table[i][j]], coord{i, j})
		}
	}
	wg := sync.WaitGroup{}
	wg.Add(RC + 1)
	for i := range RC + 1 {
		go func() {
			for j, vj := range V[i] {
				for _, vk := range V[i][:j] {
					ways[max(abs(vj.i-vk.i), max(vj.j, vk.j))].Add(-2)
				}
			}
			wg.Done()
		}()
	}

	wg.Wait()
	ways[0].Store(0)
	for i := range N {
		if int(ways[i].Load()) >= input.K {
			return output{
				result: i,
			}
		}
		input.K -= int(ways[i].Load())
	}
	panic("K too big")
}

func abs(a int) int {
	if a < 0 {
		return -a
	}
	return a
}

//package debug
//file ..//debug/go

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

//package fastio
//file ..//fastio/reader.go

type Reader struct {
	File  *os.File
	bytes [buffSize]byte
	from  int
	to    int
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

func (r *Reader) Int2() (int, int) {
	return r.Int(), r.Int()
}

func (r *Reader) Int3() (int, int, int) {
	return r.Int(), r.Int(), r.Int()
}

func (r *Reader) Ints(n int) []int {
	a := make([]int, n)
	for i := range a {
		a[i] = r.Int()
	}
	return a
}

func isWhite(c byte) bool {
	return c == ' ' || c == '\n' || c == '\r' || c == '\t'
}

func isNewLine(c byte) bool {
	return c == '\n' || c == '\r'
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

func (r *Reader) Strings(n int) []string {
	strings := make([]string, n)
	for i := range n {
		strings[i] = r.String()
	}
	return strings
}

func (r *Reader) Line() string {
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
			if isNewLine(r.bytes[i]) {
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

func (w *Reader) Float() float64 {
	str := w.String()
	flt, err := strconv.ParseFloat(str, 64)
	if err != nil {
		panic("invalid float str: " + str + " err:" + err.Error())
	}
	return flt
}

//package fastio
//file ..//fastio/writer.go

const buffSize = 100000

const maxIntSize = 128

type Writer struct {
	File      *os.File
	buffer    [buffSize]byte
	intBuffer [maxIntSize]byte
	used      int
}

func (w *Writer) WriteAll() {
	n, err := w.File.Write(w.buffer[:w.used])
	if n != w.used {
		panic("failed to write: " + err.Error())
	}
	w.used = 0
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

func (w *Writer) Ints(n []int, sep byte) {
	if len(n) == 0 {
		w.String("\n")
	}
	for i, v := range n {
		if i != len(n)-1 {
			w.Int(v, sep)
		} else {
			w.Int(v, '\n')
		}
	}
}

func (w *Writer) Float(f float64) {
	str := strconv.FormatFloat(f, 'f', -1, 64)
	w.bytes([]byte(str))
}

