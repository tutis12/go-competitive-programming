package main

import (
	"fmt"
	"os"
	"runtime"
	"strconv"
	"time"
)

//package main
//file ..//go

const (
	fromFile   = false
	inputFile  = "four_in_a_burrow_input (1).txt"
	outputFile = "output.txt"
)

/*input
3
6 10
1 5 7 8 11 12
6
1 6
1 5
2 6
1 4
2 5
3 6
6 1
1 1 1 3 3 3
2
3 3
1 6
12 15
4 5 15 24 27 32 36 39 40 46 48 48
20
1 12
1 11
6 10
1 8
8 12
11 12
2 9
3 8
7 8
7 10
4 8
9 12
9 10
2 12
1 5
3 12
4 8
3 7
7 12
10 11

*/

/*output
3
2
2
2
2
2
1
4
6
6
2
4
2
2
5
3
2
2
2
2
2
6
4
5
2
3
2
2

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
	solve(stdin, stdout)
}

//package main
//file ..//solve.go

func solve(
	stdin *Reader,
	stdout *Writer,
) {
	t := stdin.Int()
	for range t {
		solveTest(stdin, stdout)
	}
}

const log = 18

func solveTest(
	stdin *Reader,
	stdout *Writer,
) {
	n, z := stdin.Uint32_2()
	x := stdin.Uint32s(int(n))
	q := stdin.Int()
	jump := [log][]uint32{}
	for i := range log {
		jump[i] = make([]uint32, n+1)
	}
	jump[0][n] = n
	{
		j := uint32(1)
		for i := range n {
			for j < n && x[j]-x[i] <= z {
				j++
			}
			jump[0][i] = j
		}
	}

	for j := 1; j < log; j++ {
		for i := range n + 1 {
			jump[j][i] = jump[j-1][jump[j-1][i]]
		}
	}

	type big struct {
		pos   uint32
		jumps uint32
	}
	jumpBig := [log][]big{}
	for i := range log {
		jumpBig[i] = make([]big, n+1)
	}
	for pos := range n {
		i := pos
		j := i + 1
		cnt := uint32(1)
		for t := log - 1; t >= 0; t-- {
			i1 := jump[t][i]
			i2 := jump[t][j]
			i3 := jump[0][i1]
			if i1 != i2 && i2 != i3 {
				i, j = i1, i2
				cnt += 1 << (t + 1)
			}
		}

		{
			k := jump[0][i]

			if k != j {
				cnt++
				k1 := jump[0][j]
				_, j = j, k

				{
					k := k1
					if k != j {
						cnt++
						_, j = j, k

						{
							k := jump[1][i]
							if k != j {
								cnt++
								_, j = j, k
							}
						}

					}
				}

			}
		}

		jumpBig[0][pos] = big{j, cnt}
	}
	jumpBig[0][n] = big{n, 0}
	for j := 1; j < log; j++ {
		for i := range n + 1 {
			b1 := jumpBig[j-1][i]
			b2 := jumpBig[j-1][b1.pos]
			jumpBig[j][i] = big{b2.pos, b1.jumps + b2.jumps}
		}
	}
	for range q {
		l, r := stdin.Uint32_2()
		l--
		r--

		i := l

		cnt := uint32(1)
		for t := log - 1; t >= 0; t-- {
			b := jumpBig[t][i]
			if b.pos <= r {
				i = b.pos
				cnt += b.jumps
			}
		}

		j := i + 1
		for t := log - 1; t >= 0; t-- {
			i1 := jump[t][i]
			i2 := jump[t][j]
			i3 := jump[0][i1]
			if i1 != i2 && i2 != i3 && i2 <= r {
				i, j = i1, i2
				cnt += 1 << (t + 1)
			}
		}
		if j <= r {
			cnt++
			k := jump[0][i]
			j0 := j
			if k == j {
				_, j = j, j+1
			} else {
				_, j = j, k
			}

			if j <= r {
				cnt++
				k := jump[0][j0]
				if k == j {
					_, j = j, j+1
				} else {
					_, j = j, k
				}

				if j <= r {
					cnt++
				}
			}
		}
		stdout.Uint32(cnt, '\n')
	}
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

func (r *Reader) Uint32() uint32 {
	n := uint32(0)
	for {
		c, ok := r.peek()
		if !ok {
			return 0
		}
		r.seek()
		if '0' <= c && c <= '9' {
			n = uint32(c - '0')
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
		n = n*10 + uint32(c-'0')
	}
	return n
}

func (r *Reader) Int2() (int, int) {
	return r.Int(), r.Int()
}

func (r *Reader) UInt2() (uint, uint) {
	return r.Uint(), r.Uint()
}

func (r *Reader) Uint32_2() (uint32, uint32) {
	return r.Uint32(), r.Uint32()
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

func (r *Reader) Uints(n int) []uint {
	a := make([]uint, n)
	for i := range a {
		a[i] = r.Uint()
	}
	return a
}

func (r *Reader) Uint32s(n int) []uint32 {
	a := make([]uint32, n)
	for i := range a {
		a[i] = r.Uint32()
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

func (w *Writer) Uint32(n uint32, c byte) {
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

