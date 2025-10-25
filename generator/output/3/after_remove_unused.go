package main

import (
	"fmt"
	"math"
	"math/bits"
	"os"
	"runtime"
	"unsafe"
)

const (
	fromFile  = false
	inputFile = "crash_course_input.txt"
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
	defer ExitOnPanic()

	SolveX(stdin, stdout)
}

var SolveX = SolveA

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
	st := *NewSegmentTreeG1stValueG2lazy(
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
	for i := range n {
		ai := a[i]
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

func LogFloor(x uint64) int {
	if x == 0 {
		panic("Log2(0) is undefined")
	}
	return 63 - bits.LeadingZeros64(x)
}

func LogCeil(x uint64) int {
	if x == 0 {
		panic("Log2(0) is undefined")
	}
	if x == 1 {
		return 0
	}
	return 1 + LogFloor(x-1)
}

func IsPowerOf2G1int(x int,

) bool {
	x64 := uint64(x)
	return x64 !=
		0 && (x64&(x64-
		1)) ==
		0
}
func NewSegmentTreeG1stValueG2lazy(init func(int) stValue,

	size int, zeroValue stValue,

	zeroUpdate lazy,

) *SegmentTreeG1stValueG2lazy {
	if size <=
		0 {
		panic("size must be positive")
	}
	log2n := LogCeil(uint64(size))
	n := 1 <<
		log2n

	arr := make([]segmentTreeNodeG1stValueG2lazy, 2*n)
	for i := range size {
		*GetG1segmentTreeNodeOfstValueClazy(arr,
			n+i) = segmentTreeNodeG1stValueG2lazy{value: init(i),

			update: zeroUpdate}
	}
	for i := size; i < n; i++ {
		*GetG1segmentTreeNodeOfstValueClazy(arr,
			n+i) = segmentTreeNodeG1stValueG2lazy{value: zeroValue,

			update: zeroUpdate}
	}
	for i := n - 1; i >
		0; i-- {
		*GetG1segmentTreeNodeOfstValueClazy(arr, i) = segmentTreeNodeG1stValueG2lazy{value: (GetG1segmentTreeNodeOfstValueClazy(arr, 2*
			i,
		).
			value).
			Merge(GetG1segmentTreeNodeOfstValueClazy(arr,

				2*i+1).
				value,
			), update: zeroUpdate}
	}
	return &SegmentTreeG1stValueG2lazy{log2n: log2n, n: n, arr: arr, zeroValue: zeroValue,
		zeroUpdate: zeroUpdate,
	}
}
func GetG1segmentTreeNodeOfstValueClazy(slice []segmentTreeNodeG1stValueG2lazy,

	index int) *segmentTreeNodeG1stValueG2lazy {
	return (*segmentTreeNodeG1stValueG2lazy)(unsafe.
		Pointer(uintptr(unsafe.Pointer(unsafe.
			SliceData(
				slice,
			))) + uintptr(index)*unsafe.
			Sizeof(*new(segmentTreeNodeG1stValueG2lazy))))
}

type SegmentTreeG1stValueG2lazy struct {
	log2n int
	n     int

	arr       []segmentTreeNodeG1stValueG2lazy
	zeroValue stValue

	zeroUpdate lazy
}

func (st *SegmentTreeG1stValueG2lazy,

) SetValue(i int, val stValue,

) {
	if i < 0 ||
		i >= st.n {
		panic("index out of bounds")
	}
	st.pushUpdates(
		i)
	(*GetG1segmentTreeNodeOfstValueClazy(st.arr, i+st.n)).
		value = val
	(*GetG1segmentTreeNodeOfstValueClazy(st.arr, i+st.n)).update = st.zeroUpdate
	st.rebuild(i)
}

func (st *SegmentTreeG1stValueG2lazy,

) Get(l, r int) stValue {
	l = max(l, 0)
	r = min(r, st.n-
		1)
	if l > r {
		return st.zeroValue
	}
	st.pushUpdates(l)
	st.pushUpdates(r)
	summedL := st.zeroValue
	summedR :=
		st.
			zeroValue
	l,
		r = l+st.n, r+st.n
	for l <= r {
		if l%
			2 ==
			1 {
			(GetG1segmentTreeNodeOfstValueClazy(st.arr, l).update).ApplyUpdate(
				&GetG1segmentTreeNodeOfstValueClazy(st.arr, l).
					value)
			summedL = summedL.Merge((*GetG1segmentTreeNodeOfstValueClazy(st.arr, l)).value)
			l = l/2 + 1
		} else {
			l = l /
				2
		}
		if r%2 == 0 {
			(GetG1segmentTreeNodeOfstValueClazy(st.
				arr, r).update).ApplyUpdate(&GetG1segmentTreeNodeOfstValueClazy(st.arr, r).value)
			summedR =
				GetG1segmentTreeNodeOfstValueClazy(st.arr, r).value.Merge(summedR)
			r = r/2 - 1
		} else {
			r = r / 2
		}
	}
	return summedL.Merge(summedR)
}
func (st *SegmentTreeG1stValueG2lazy,

) LongestRangeWherePredicate(rMax int, predicate func(stValue) bool) (int,
	bool) {
	if rMax <
		0 || rMax >=
		st.
			n {
		panic("invalid rMax")
	}
	st.pushUpdates(rMax)
	i := rMax +

		st.n
	if !predicate(GetG1segmentTreeNodeOfstValueClazy(st.arr, i).value) {
		return -1,
			false
	}
	summedValue :=
		st.zeroValue
	for i >
		0 {
		arrVal :=
			*GetG1segmentTreeNodeOfstValueClazy(st.arr, i)
		(arrVal.update).ApplyUpdate(&arrVal.
			value)
		val := arrVal.
			value.Merge(summedValue)
		if predicate(val) {

			if IsPowerOf2G1int(i) {
				return 0, true
			}
			if i%2 == 0 {
				i = i/2 - 1
				summedValue = val.Merge(summedValue)
			} else {
				i = i /
					2
			}
		} else {

			break
		}
	}
	upd := GetG1segmentTreeNodeOfstValueClazy(st.
		arr, i).update
	for i <

		st.n {
		val := *GetG1segmentTreeNodeOfstValueClazy(st.
			arr,

			2*
				i+
				1)
		(upd).
			Push(&val.
				update,
			)
		(upd).ApplyUpdate(&val.value)
		combined := val.
			value.Merge(summedValue)
		if predicate(
			combined,
		) {
			summedValue = combined
			i = 2 * i
		} else {

			upd = val.update

			i = 2*i + 1
		}
	}
	return i -
		st.n + 1, true
}
func (st *SegmentTreeG1stValueG2lazy,

) pushUpdates(i int) {
	i += st.
		n
	for shift := st.log2n; shift >
		0; shift-- {
		i := i >>
			shift
		update := GetG1segmentTreeNodeOfstValueClazy(st.
			arr, i).update
		GetG1segmentTreeNodeOfstValueClazy(st.arr, i).update = st.zeroUpdate

		(update).ApplyUpdate(&GetG1segmentTreeNodeOfstValueClazy(st.arr, i).
			value)
		(update).
			Push(&GetG1segmentTreeNodeOfstValueClazy(st.arr, 2*i).update)
		(update).Push(&GetG1segmentTreeNodeOfstValueClazy(st.arr, 2*i+
			1).update)
	}
	(GetG1segmentTreeNodeOfstValueClazy(st.arr, i).update).ApplyUpdate(&GetG1segmentTreeNodeOfstValueClazy(
		st.arr, i).value)
	GetG1segmentTreeNodeOfstValueClazy(st.arr, i).update = st.
		zeroUpdate
}
func (st *SegmentTreeG1stValueG2lazy,

) rebuild(
	i int) {
	i += st.n
	i /= 2
	for i != 0 {
		left := *GetG1segmentTreeNodeOfstValueClazy(st.arr,

			2*
				i)
		(left.update).ApplyUpdate(&left.
			value)
		right := *GetG1segmentTreeNodeOfstValueClazy(st.arr, 2*i+1)
		(right.update).ApplyUpdate(&right.
			value)
		GetG1segmentTreeNodeOfstValueClazy(st.arr, i).value = left.value.
			Merge(right.value)
		i = i / 2
	}
}

type segmentTreeNodeG1stValueG2lazy struct {
	value stValue

	update lazy
}

