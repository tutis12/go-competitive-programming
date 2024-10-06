package main

import (
	"fmt"
	"math"
	"os"
	"runtime"
	"strconv"
	"sync"
	"sync/atomic"
	"time"
)

//package main
//file ..//go

func main() {
	const fromFile = false

	var stdout = &Writer{
		File: os.Stdout,
	}

	var stdin = &Reader{
		File: os.Stdin,
	}
	if fromFile {
		inputFile, err := os.Open("io/wildcard_submissions_input.txt")
		if err != nil {
			panic(err.Error())
		}
		stdin.File = inputFile
		outputFile, err := os.Create("io/output.txt")
		if err != nil {
			panic(err.Error())
		}
		stdout.File = outputFile
	}

	defer stdout.WriteAll()
	defer Recover()
	Hackercup(stdin, stdout)
}

/*input
6
??2 3
135201 1
?35 2
1?0 2
1122 1
3???????????????????3 1337

*/

/*output
Case #1: 122 3
Case #2: 135201 2
Case #3: 135 2
Case #4: 110 1
Case #5: 1122 5
Case #6: 322222222121221112223 10946

*/
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
	for r.from == r.to {
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
		return math.NaN()
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
	n, _ := w.File.Write(w.buffer[:w.used])
	if n != w.used {
		panic("failed to write")
	}
	w.used = 0
}

func (w *Writer) bytes(c []byte) {
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
	pos := true
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
	w.String(str)
}

//package debug
//file ..//debug/go

func Recover() {
	err := recover()
	if err == nil {
		return
	}
	buf := make([]byte, 10000)
	n := runtime.Stack(buf, false)
	buf = buf[:n]
	fmt.Fprintf(os.Stderr, "panic: %v\nstacktrace:\n%s", err, string(buf))
	os.Exit(13)
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

type input struct {
	s string
	k int
}

func (i *input) Read(stdin *Reader) {
	i.s = stdin.String()
	i.k = stdin.Int()
}

type output struct {
	s     string
	value int
}

func (o *output) Print(stdout *Writer) {
	stdout.String(o.s)
	stdout.String(" ")
	stdout.Int(o.value, '\n')
}

const mod = 998244353

func solve(input *input) output {
	s1 := make([]uint8, len(input.s))
	for i, c := range input.s {
		if c == '?' {
			s1[i] = 1
		} else {
			s1[i] = uint8(c - '0')
		}
	}
	dp1 := make([]bool, len(s1)+1)
	dp2 := make([]bool, len(s1)+1)
	dp1[0] = true
	dp2[len(s1)] = true
	for i := 1; i <= len(s1); i++ {
		if s1[i-1] >= 1 {
			dp1[i] = dp1[i] || dp1[i-1]
		}
		if i >= 2 && s1[i-2] >= 1 && s1[i-2]*10+s1[i-1] <= 26 {
			dp1[i] = dp1[i] || dp1[i-2]
		}
	}
	for i := len(s1) - 1; i >= 0; i-- {
		if s1[i] >= 1 {
			dp2[i] = dp2[i] || dp2[i+1]
		}
		if i <= len(s1)-2 && s1[i] >= 1 && s1[i]*10+s1[i+1] <= 26 {
			dp2[i] = dp2[i] || dp2[i+2]
		}
	}
	good := make([][2]bool, len(s1))
	for i := 0; i < len(s1); i++ {
		if s1[i] >= 1 && s1[i] <= 9 && dp1[i] && dp2[i+1] {
			good[i][0] = true
		}
		if i <= len(s1)-2 && s1[i] >= 1 && s1[i]*10+s1[i+1] <= 26 && dp1[i] && dp2[i+2] {
			good[i][1] = true
		}
	}
	s := input.s
	dp3 := make([][26]int, len(s1))
	for i := len(s) - 1; i >= 0; i-- {
		for c := range 10 {
			if input.s[i] != '0'+byte(c) && input.s[i] != '?' {
				continue
			}
			if good[i][0] {
				if c == 0 {
					continue
				}
			}
			if i == len(s)-1 {
				dp3[i][c] = 1
				continue
			}
			ways := 0
			for c1 := range 10 {
				if good[i][1] {
					if c == 0 || c*10+c1 > 26 {
						continue
					}
				}
				ways += dp3[i+1][c1]
			}
			ways = min(ways, input.k)
			dp3[i][c] = ways
		}
	}
	sBytes := []byte(s)
	for i, c := range s {
		if c == '?' {
			for c := 25; c >= 0; c-- {
				if input.k > dp3[i][c] {
					input.k -= dp3[i][c]
					continue
				}
				sBytes[i] = '0' + byte(c)
				break
			}
		}
	}
	dp4 := make([]int, len(s1)+1)
	dp4[len(s1)] = 1
	for i := len(s1) - 1; i >= 0; i-- {
		if good[i][0] {
			dp4[i] += dp4[i+1]
		}
		if good[i][1] {
			dp4[i] += dp4[i+2]
		}
		if dp4[i] >= mod {
			dp4[i] -= mod
		}
	}
	return output{
		s:     string(sBytes),
		value: dp4[0],
	}
}

/*input
6
??2 3
135201 1
?35 2
1?0 2
1122 1
3???????????????????3 1337

*/

/*output
Case #1: 122 3
Case #2: 135201 2
Case #3: 135 2
Case #4: 110 1
Case #5: 1122 5
Case #6: 322222222121221112223 10946

*/

