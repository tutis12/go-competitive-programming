package main

import (
	"fmt"
	"math"
	"math/bits"
	"math/rand/v2"
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

type intHash int

func (x intHash) Hash() uint64 {
	return uint64(x)
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
	a := NewHashTableG1intG2intHashG3int(0)
	for i := range n {
		a.Set(i, stdin.Int())
	}
	st := NewSegmentTreeG1stValueG2lazy(
		func(i int) stValue {
			return stValue{
				minA:  a.Get(i),
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

	dp := NewHashTableG1intG2intHashG3int(0)

	for i := range n {
		ai := a.Get(i)
		dp.Set(i, i+1)
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
			dp.Set(i, min(dp.Get(i), total))
		}
		st.SetValue(i, stValue{
			minA:  ai,
			minDP: dp.Get(i),
		})
	}
	stdout.Int(dp.Get(n-1), '\n')
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

const (
	maxOffset      = 8
	checkHashFirst = false
)

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

func ReverseBits64(x uint64) uint64 {
	x = (x>>1)&0x5555555555555555 | (x&0x5555555555555555)<<1
	x = (x>>2)&0x3333333333333333 | (x&0x3333333333333333)<<2
	x = (x>>4)&0x0F0F0F0F0F0F0F0F | (x&0x0F0F0F0F0F0F0F0F)<<4
	x = (x>>8)&0x00FF00FF00FF00FF | (x&0x00FF00FF00FF00FF)<<8
	x = (x>>16)&0x0000FFFF0000FFFF | (x&0x0000FFFF0000FFFF)<<16
	x = (x>>32)&0x00000000FFFFFFFF | (x&0x00000000FFFFFFFF)<<32
	return x
}

func IsPowerOf2G1int(x int,

) bool {
	x64 := uint64(x)
	return x64 !=
		0 && (x64&
		(x64-
			1)) == 0
}
func NewHashTableG1intG2intHashG3int(size int) *HashTableG1intG2intHashG3int {
	log2Size :=
		LogCeil(uint64(size*2 +
			1))
	return &HashTableG1intG2intHashG3int{entries1: make([]hashTableEntryG1intG2int, (1<<log2Size)+maxOffset), entries2: make([][]hashTableEntryG1intG2int, 1<<log2Size), log2Size: log2Size, oddSalt1: rand.Uint64() | 1,
		oddSalt2: rand.Uint64() | 1, count: 0}
}

type HashTableG1intG2intHashG3int struct {
	entries1 []hashTableEntryG1intG2int
	entries2 [][]hashTableEntryG1intG2int
	log2Size int

	oddSalt1 uint64
	oddSalt2 uint64

	count int
}

func (hm *HashTableG1intG2intHashG3int,

) hash(key int,

) uint64 {
	val := (*(*intHash)(unsafe.
		Pointer(&key))).Hash()
	if val ==
		0 {
		return 1
	} else {
		return val
	}
}
func (hm *HashTableG1intG2intHashG3int,

) index1(hash uint64) uint64 {
	hash *= hm.oddSalt1
	hash = ReverseBits64(hash)
	return hash &
		(1<<
			hm.log2Size -
			1)
}
func (hm *HashTableG1intG2intHashG3int,

) index2(hash uint64) uint64 {
	hash *= hm.oddSalt2
	hash = ReverseBits64(hash)
	return hash &
		(1<<
			hm.log2Size -
			1)
}
func (hm *HashTableG1intG2intHashG3int,

) Get(key int,

) int {
	val, ok :=
		hm.Get2(key)
	if !ok {
		var zero int

		return zero
	}
	return val
}
func (hm *HashTableG1intG2intHashG3int,

) Get2(key int,

) (int,

	bool) {
	hash :=
		hm.hash(key)
	index1 := hm.index1(hash)
	arr := GetArrG1hashTableEntryOfintCint(
		hm.entries1,
		int(index1))
	var zero int

	for _, val := range arr {
		if val.hash ==
			0 {
			return zero, false
		}
		if (!checkHashFirst ||
			val.hash == hash) &&
			val.key == key {
			return val.value, true
		}
	}
	index2 :=
		hm.
			index2(hash)
	for _, val := range *GetG1SlicehashTableEntryOfintCint(hm.entries2, int(index2)) {
		if (!checkHashFirst ||
			val.hash == hash) &&
			val.
				key == key {
			return val.
				value, true
		}
	}
	return zero, false
}

func (hm *HashTableG1intG2intHashG3int,

) Set(key int,

	value int,

) {
	if hm.count*
		2 >= (1 <<
		hm.log2Size) {
		hm.resize()
	}
	hash := hm.
		hash(key)
	index1 :=
		hm.index1(hash)
	arr := GetArrG1hashTableEntryOfintCint(hm.entries1,
		int(index1),
	)
	for i := range maxOffset {
		e := &arr[i]
		if e.hash ==
			0 {
			*e = hashTableEntryG1intG2int{hash: hash, key: key, value: value}
			hm.count++

			return
		} else if (!checkHashFirst || e.hash == hash) && e.key == key {
			e.value = value
			return
		}
	}
	index2 := hm.index2(hash)
	entries := GetG1SlicehashTableEntryOfintCint(hm.entries2, int(index2))
	for i, val := range *entries {
		if val.hash == hash &&
			val.key == key {
			GetG1hashTableEntryOfintCint(*entries, i).value = value
			return
		}
	}
	*entries = append(*entries, hashTableEntryG1intG2int{hash: hash,

		key: key,

		value: value,
	})

	hm.
		count++
}
func (hm *HashTableG1intG2intHashG3int,

) resize() {
	hm.log2Size++
	newEntries1 := make([]hashTableEntryG1intG2int, (1<<hm.
		log2Size)+
		maxOffset,
	)
	newEntries2 := make([][]hashTableEntryG1intG2int, 1<<
		hm.log2Size)
	add := func(e hashTableEntryG1intG2int) {
		hash := hm.hash(e.
			key)
		index1 := hm.index1(hash)
		arr :=
			GetArrG1hashTableEntryOfintCint(newEntries1, int(
				index1))
		for j := range arr {
			if arr[j].hash == 0 {
				arr[j] = e
				return
			}
		}
		index2 := hm.index2(hash)
		entries := GetG1SlicehashTableEntryOfintCint(newEntries2,
			int(index2))
		*entries = append(*entries, e)
	}
	for _, e := range hm.entries1 {
		if e.hash == 0 {
			continue
		}
		add(e)
	}
	for _, bucket := range hm.entries2 {
		for _, e := range bucket {
			add(e)
		}
	}
	hm.entries1 = newEntries1

	hm.
		entries2 = newEntries2

}
func GetG1SlicehashTableEntryOfintCint(slice [][]hashTableEntryG1intG2int,

	index int) *[]hashTableEntryG1intG2int {
	return (*[]hashTableEntryG1intG2int)(
		unsafe.
			Pointer(uintptr(unsafe.
				Pointer(unsafe.SliceData(slice)),
			) + uintptr(index)*
				unsafe.
					Sizeof(*new([]hashTableEntryG1intG2int))))
}
func GetG1hashTableEntryOfintCint(slice []hashTableEntryG1intG2int,

	index int) *hashTableEntryG1intG2int {
	return (*hashTableEntryG1intG2int)(
		unsafe.
			Pointer(uintptr(unsafe.
				Pointer(unsafe.SliceData(slice)),
			) + uintptr(index)*
				unsafe.
					Sizeof(*new(hashTableEntryG1intG2int))))
}
func GetArrG1hashTableEntryOfintCint(
	slice []hashTableEntryG1intG2int,

	offset int) *[8]hashTableEntryG1intG2int {
	data := unsafe.
		Add(unsafe.
			Pointer(unsafe.
				SliceData(slice)), uintptr(offset)*
			unsafe.Sizeof(*new(hashTableEntryG1intG2int)))
	return (*[8]hashTableEntryG1intG2int)(data)
}

func NewSegmentTreeG1stValueG2lazy(init func(int) stValue,

	size int, zeroValue stValue,

	zeroUpdate lazy,

) *SegmentTreeG1stValueG2lazy {
	if size <= 0 {
		panic("size must be positive")
	}
	log2n := LogCeil(uint64(size))
	n := 1 << log2n
	arr := make([]segmentTreeNodeG1stValueG2lazy, 2*n)
	for i := range size {
		*GetG1segmentTreeNodeOfstValueClazy(arr, n+i) = segmentTreeNodeG1stValueG2lazy{init(i), zeroUpdate}
	}
	for i := size; i <
		n; i++ {
		*GetG1segmentTreeNodeOfstValueClazy(arr, n+i) = segmentTreeNodeG1stValueG2lazy{zeroValue,
			zeroUpdate}
	}
	for i := n - 1; i > 0; i-- {

		*GetG1segmentTreeNodeOfstValueClazy(arr, i) = segmentTreeNodeG1stValueG2lazy{
			(GetG1segmentTreeNodeOfstValueClazy(arr, 2*i).value).Merge(GetG1segmentTreeNodeOfstValueClazy(arr, 2*i+1).value),
			zeroUpdate,
		}

	}
	return &SegmentTreeG1stValueG2lazy{log2n: log2n,

		n:   n,
		arr: arr, zeroValue: zeroValue,
		zeroUpdate: zeroUpdate,
	}
}
func GetG1segmentTreeNodeOfstValueClazy(slice []segmentTreeNodeG1stValueG2lazy,

	index int) *segmentTreeNodeG1stValueG2lazy {
	return (*segmentTreeNodeG1stValueG2lazy)(
		unsafe.
			Pointer(uintptr(unsafe.
				Pointer(unsafe.SliceData(slice)),
			) + uintptr(index)*
				unsafe.
					Sizeof(*new(segmentTreeNodeG1stValueG2lazy))))
}

type SegmentTreeG1stValueG2lazy struct {
	log2n int
	n     int

	arr       []segmentTreeNodeG1stValueG2lazy
	zeroValue stValue

	zeroUpdate lazy
}

func (st SegmentTreeG1stValueG2lazy,

) SetValue(i int, val stValue,

) {
	if i <
		0 || i >=
		st.n {
		panic("index out of bounds")
	}
	st.pushUpdates(i)
	(*GetG1segmentTreeNodeOfstValueClazy(st.
		arr, i+st.n)).
		value = val
	(*GetG1segmentTreeNodeOfstValueClazy(st.arr, i+
		st.n)).update = st.zeroUpdate
	st.rebuild(i)
}

func (st SegmentTreeG1stValueG2lazy,

) Get(l, r int) stValue {
	l = max(l, 0)
	r = min(r,
		st.n-1,
	)
	if l > r {
		return st.
			zeroValue
	}
	st.pushUpdates(l)
	st.
		pushUpdates(r)

	summedL := st.zeroValue
	summedR := st.zeroValue
	l, r = l+st.n,
		r+
			st.n
	for l <=
		r {
		if l%2 == 1 {
			(GetG1segmentTreeNodeOfstValueClazy(st.arr, l).
				update).ApplyUpdate(&GetG1segmentTreeNodeOfstValueClazy(st.
				arr, l).value)
			summedL = summedL.
				Merge((*GetG1segmentTreeNodeOfstValueClazy(st.arr, l)).value)
			l = l/2 + 1
		} else {
			l = l / 2
		}
		if r%2 == 0 {
			(GetG1segmentTreeNodeOfstValueClazy(st.
				arr, r).update).ApplyUpdate(&GetG1segmentTreeNodeOfstValueClazy(st.arr, r).value)
			summedR = GetG1segmentTreeNodeOfstValueClazy(st.arr,
				r).value.Merge(summedR)
			r = r/2 - 1
		} else {
			r = r / 2
		}
	}
	return summedL.Merge(summedR)
}
func (st SegmentTreeG1stValueG2lazy,

) LongestRangeWherePredicate(rMax int,
	predicate func(stValue) bool) (int,
	bool) {
	if rMax <
		0 || rMax >=

		st.n {
		panic("invalid rMax")
	}
	st.
		pushUpdates(rMax)
	i := rMax + st.n
	if !predicate(GetG1segmentTreeNodeOfstValueClazy(st.
		arr,
		i).value) {
		return -1, false
	}
	summedValue :=
		st.
			zeroValue
	for i > 0 {
		arrVal := GetG1segmentTreeNodeOfstValueClazy(st.
			arr,

			i)
		(arrVal.update).ApplyUpdate(&arrVal.value)
		val := arrVal.value.
			Merge(summedValue)
		if predicate(val) {

			if IsPowerOf2G1int(i) {

				return 0, true
			}
			if i%2 == 0 {
				i = i/
					2 - 1
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
	for i < st.
		n {
		val := GetG1segmentTreeNodeOfstValueClazy(st.arr,

			2*
				i+1)
		(upd).Push(&val.
			update,
		)
		(upd).ApplyUpdate(
			&val.
				value)
		combined := val.
			value.Merge(summedValue)
		if predicate(
			combined) {
			summedValue = combined

			i = 2 *
				i
		} else {
			upd =
				val.
					update
			i = 2*i +
				1
		}
	}
	return i - st.n + 1, true
}
func (st SegmentTreeG1stValueG2lazy,

) pushUpdates(i int) {
	i +=
		st.n
	for shift := st.
		log2n; shift > 0; shift-- {
		i := i >>
			shift
		update := GetG1segmentTreeNodeOfstValueClazy(st.arr,
			i).update
		GetG1segmentTreeNodeOfstValueClazy(st.arr,
			i).update = st.zeroUpdate
		(update).ApplyUpdate(&GetG1segmentTreeNodeOfstValueClazy(
			st.
				arr, i).value)
		(update).Push(&GetG1segmentTreeNodeOfstValueClazy(
			st.arr, 2*i).update,
		)
		(update).Push(&GetG1segmentTreeNodeOfstValueClazy(st.
			arr, 2*i+1).update,
		)
	}
	(GetG1segmentTreeNodeOfstValueClazy(st.arr,
		i).update).ApplyUpdate(&GetG1segmentTreeNodeOfstValueClazy(st.arr, i).value)
	GetG1segmentTreeNodeOfstValueClazy(st.arr,
		i).update = st.
		zeroUpdate
}
func (st SegmentTreeG1stValueG2lazy,

) rebuild(i int) {
	i += st.
		n
	i /= 2
	for i != 0 {
		left := GetG1segmentTreeNodeOfstValueClazy(st.arr,
			2*i)
		(left.
			update).ApplyUpdate(&left.value)
		right := GetG1segmentTreeNodeOfstValueClazy(st.arr,
			2*i+1)
		(right.update).ApplyUpdate(&right.value)
		GetG1segmentTreeNodeOfstValueClazy(
			st.
				arr, i).value = left.value.Merge(right.
			value)
		i = i /
			2
	}
}

type hashTableEntryG1intG2int struct {
	hash uint64
	key  int

	value int
}
type segmentTreeNodeG1stValueG2lazy struct {
	value stValue

	update lazy
}

