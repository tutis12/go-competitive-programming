package main

import (
	"fmt"
	"os"
	"runtime"
)

const (
	fromFile  = false
	inputFile = "crash_course_input.txt"
)

func main() {
	var stdin = Reader{
		File: os.Stdin,
	}
	var stdout = Writer{
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
	defer ExitOnPanic()

	SolveX(&stdin, &stdout)
}

var SolveX = SolveA

var globalCost int
var globalAi int

func SolveA(
	stdin *Reader,
	stdout *Writer,
) {
	t := stdin.Int()
	for range t {
		solveATest(stdin, stdout)
	}
}

func solveATest(
	stdin *Reader,
	stdout *Writer,
) {
	n := stdin.Int()
	a := stdin.Ints(n, 0)
	const maxDP = 128
	dp := [maxDP + 1]int{}
	for i := range dp {
		dp[i] = -1
	}
	stack := make([]int, 0, n+1)
	stack = append(stack, -1)
	for i := range n {
		ai := a[i]
		dpI := i + 1
		dpJ := 0
		for len(stack) != 1 && a[stack[len(stack)-1]] >= ai {
			stack = stack[:len(stack)-1]
		}
		stack = append(stack, i)
		for cost := 3; cost >= 1; cost-- {
			globalCost = cost
			globalAi = ai
			var l int
			if cost != 1 {
				lo := 0
				hi := len(stack) - 1
				for lo < hi {
					mid := (lo + hi + 1) / 2
					if a[stack[mid]]*cost < ai {
						lo = mid
					} else {
						hi = mid - 1
					}
				}
				l = stack[lo] + 1
			} else {
				l = stack[len(stack)-2] + 1
			}
			var total int
			if l == 0 {
				total = cost
			} else {
				for dp[dpJ] < l-1 {
					dpJ++
				}
				total = dpJ + cost
			}
			dpI = min(dpI, total)
		}
		for j := dpI; j <= maxDP; j++ {
			dp[j] = i
		}
	}
	ans := 0
	for dp[ans] < n-1 {
		ans++
	}
	stdout.Int(ans, '\n')
}

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

func (r *Reader) Ints(n int, firstIndex int) []int {
	a := make([]int, n+firstIndex)
	for i := range n {
		a[i+firstIndex] = r.Int()
	}
	return a
}

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

func ExitOnPanic() {
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

