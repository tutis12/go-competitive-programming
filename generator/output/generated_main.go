package main

import (
	"fmt"
	"math"
	"math/bits"
	"os"
	"runtime"
)

const (
	fromFile   = false
	inputFile  = "input.txt"
	outputFile = "output.txt"
)

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

	solveX(stdin, stdout)
}

var solveX = solveC

func solveC(
	stdin *Reader,
	stdout *Writer,
) {
	t := stdin.Int()
	for range t {
		solveTestC(stdin, stdout)
	}
}

type stValue struct {
	minA  int
	minDP int
}

type lazy struct {
	addDP int
}

func (a stValue) Merge(b stValue) stValue {
	return stValue{minA: min(a.minA, b.minA), minDP: min(a.minDP, b.minDP)}
}

func (up lazy) ApplyUpdate(val *stValue) {
	val.minDP += up.addDP
}

func (top lazy) Push(existing *lazy) {
	existing.addDP += top.addDP
}

func solveTestC(
	stdin *Reader,
	stdout *Writer,
) {
	n := stdin.Int()
	a := stdin.Ints(n, 0)
	st := NewSTG1(
		func(i int) stValue {
			return stValue{
				minA:  a[i],
				minDP: 0,
			}
		},
		n,
		stValue{
			minA:  math.MaxInt,
			minDP: math.MaxInt,
		},
		lazy{},
	)

	dp := make([]int, n)

	for i, ai := range a {
		dp[i] = i + 1
		for cost := 1; cost <= 3; cost++ {
			l, _ := st.LongestRangeWherePredicate(i, func(x stValue) bool {
				return x.minA*cost >= ai
			})
			var total int
			if l == 0 {
				total = cost
			} else {
				total = st.Get(l-1, i-1).minDP + cost
			}
			dp[i] = min(dp[i], total)
		}
		st.SetValue(i, stValue{
			minA:  ai,
			minDP: dp[i],
		})
	}
	stdout.Int(dp[n-1], '\n')
}

const (
	mod = 1000000007
)

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

func isWhite(c byte) bool {
	return c == ' ' || c == '\n' || c == '\r' || c == '\t'
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

func (w *Writer) Fprintf(format string, a ...any) {
	w.String(fmt.Sprintf(format, a...))
}

func (w *Writer) Char(c byte) {
	w.bytes([]byte{c})
}

func (w *Writer) Ints(n []int, sep byte, end byte) {
	if len(n) == 0 {
		w.Char(end)
	}
	for i, v := range n {
		if i != len(n)-1 {
			w.Int(v, sep)
		} else {
			w.Int(v, end)
		}
	}
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

type STG1 struct {
	log2n      int
	n          int
	arr        []nodeG1
	zeroValue  stValue
	zeroUpdate lazy
}
type nodeG1 struct {
	value  stValue
	update lazy
}

func (st STG1) SetValue(
	i int,
	val stValue,
) {
	if i < 0 || i >= st.n {
		panic("index out of bounds")
	}
	st.pushUpdates(i)
	st.arr[i+st.n].value = val
	st.arr[i+st.n].update = st.zeroUpdate
	st.rebuild(i)
}

func (st STG1) Get(
	l, r int,
) stValue {
	l = max(l, 0)
	r = min(r, st.n-1)
	if l > r {
		return st.zeroValue
	}
	st.pushUpdates(l)
	st.pushUpdates(r)
	summedL := st.zeroValue
	summedR := st.zeroValue
	l, r = l+st.n, r+st.n
	for l <= r {
		if l%2 == 1 {

			(st.arr[l].update).ApplyUpdate(&st.arr[l].value)
			summedL = summedL.Merge(st.arr[l].value)
			l = l/2 + 1
		} else {
			l = l / 2
		}
		if r%2 == 0 {
			(st.arr[r].update).ApplyUpdate(&st.arr[r].value)
			summedR = st.arr[r].value.Merge(summedR)
			r = r/2 - 1
		} else {
			r = r / 2
		}
	}
	return summedL.Merge(summedR)
}
func (st STG1) LongestRangeWherePredicate(
	rMax int,
	predicate func(stValue) bool,
) (int, bool) {
	if rMax < 0 || rMax >= st.n {
		panic("invalid rMax")
	}

	st.pushUpdates(rMax)
	i := rMax + st.n
	if !predicate(st.arr[i].value) {
		return -1, false
	}
	summedValue := st.zeroValue
	for i > 0 {
		arrVal := st.arr[i]
		(arrVal.update).ApplyUpdate(&arrVal.value)
		val := arrVal.value.Merge(summedValue)
		if predicate(val) {
			if IsPowerOf2G1(i) {
				return 0, true
			}
			if i%2 == 0 {
				i = i/2 - 1
				summedValue = val.Merge(summedValue)
			} else {
				i = i / 2
			}
		} else {
			break
		}
	}

	upd := st.arr[i].update
	for i < st.n {
		val := st.arr[2*i+1]
		(upd).Push(&val.update)
		(upd).ApplyUpdate(&val.value)
		combined := val.value.Merge(summedValue)
		if predicate(combined) {
			summedValue = combined
			i = 2 * i
		} else {
			upd = val.update
			i = 2*i + 1
		}
	}
	return i - st.n + 1, true
}
func (st STG1) pushUpdates(
	i int,
) {
	i += st.n
	for shift := st.log2n; shift > 0; shift-- {
		i := i >> shift
		update := st.arr[i].update

		st.arr[i].update = st.zeroUpdate
		(update).ApplyUpdate(&st.arr[i].value)

		(update).Push(&st.arr[2*i].update)
		(update).Push(&st.arr[2*i+1].update)
	}
	(st.arr[i].update).ApplyUpdate(&st.arr[i].value)
	st.arr[i].update = st.zeroUpdate
}
func (st STG1) rebuild(
	i int,
) {
	i += st.n
	i /= 2
	for i != 0 {
		left := st.arr[2*i]
		(left.update).ApplyUpdate(&left.value)
		right := st.arr[2*i+1]
		(right.update).ApplyUpdate(&right.value)
		st.arr[i].value = left.value.Merge(right.value)
		i = i / 2
	}
}

func NewSTG1(
	init func(int) stValue,
	size int,
	zeroValue stValue,
	zeroUpdate lazy,
) *STG1 {
	if size <= 0 {
		panic("size must be positive")
	}
	log2n := Log2CeilG1(size)
	n := 1 << log2n
	arr := make([]nodeG1, 2*n)
	for i := range size {
		arr[n+i] = nodeG1{init(i), zeroUpdate}
	}
	for i := size; i < n; i++ {
		arr[n+i] = nodeG1{zeroValue, zeroUpdate}
	}
	for i := n - 1; i > 0; i-- {
		arr[i] = nodeG1{arr[2*i].value.Merge(arr[2*i+1].value), zeroUpdate}
	}
	return &STG1{
		log2n:      log2n,
		n:          n,
		arr:        arr,
		zeroValue:  zeroValue,
		zeroUpdate: zeroUpdate,
	}
}
func Log2FloorG1(x uint64) int {
	x64 := uint64(x)
	if x64 == 0 {
		panic("Log2(0) is undefined")
	}
	return 63 - bits.LeadingZeros64(x64)
}
func Log2CeilG1(x int) int {
	x64 := uint64(x)
	if x64 == 0 {
		panic("Log2(0) is undefined")
	}
	if x64 == 1 {
		return 0
	}
	return 1 + Log2FloorG1(x64-1)
}
func IsPowerOf2G1(x int) bool {
	x64 := uint64(x)
	return x64 != 0 && (x64&(x64-1)) == 0
}

