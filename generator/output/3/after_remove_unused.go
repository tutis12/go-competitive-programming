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

	SolveX(stdin, stdout)
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

var SolveX = SolveA

type (
	HashTableG1intG2intHashG3int struct {
		entries1 []hashTableEntryG1KG2V

		entries2 [][]hashTableEntryG1KG2V

		log2Size int

		oddSalt1 uint64
		oddSalt2 uint64

		count int
	}
	SegmentTreeG1valueG2update struct {
		log2n int
		n     int

		arr []segmentTreeNodeG1valueG2update

		zeroValue value

		zeroUpdate update
	}
	segmentTreeNodeG1valueG2update struct {
		value value

		update update
	}

	HashTableG1intG2intHasherG3int struct {
		entries1 []hashTableEntryG1KG2V

		entries2 [][]hashTableEntryG1KG2V

		log2Size int

		oddSalt1 uint64
		oddSalt2 uint64

		count int
	}
	intHash int

	intHasher int
	stValue   struct {
		minA  int
		minDP int
	}

	SegmentTreeG1stValueG2lazy struct {
		log2n int
		n     int

		arr       []segmentTreeNodeG1valueG2update
		zeroValue stValue

		zeroUpdate lazy
	}
	input struct {
		N int
		S string
	}
	lazy struct {
		addDP int
	}

	segmentTreeNodeG1stValueG2lazy struct {
		value stValue

		update lazy
	}
	HashTableG1KG2HG3V struct {
		entries1 []hashTableEntryG1KG2V

		entries2 [][]hashTableEntryG1KG2V

		log2Size int

		oddSalt1 uint64
		oddSalt2 uint64

		count int
	}
	hashTableEntryG1KG2V struct {
		hash uint64
		key  K

		value V
	}
)

const (
	checkHashFirst = false
	maxIntSize     = 128

	maxOffset = 8
	buffSize  = 100000
)

func NewHashTableG1intG2intHashG3int(size int) *HashTableG1intG2intHashG3int {
	log2Size :=
		LogCeil(uint64(size*2 +
			1))
	return &HashTableG1intG2intHashG3int{entries1: make([]hashTableEntryG1KG2V,

		(1<<log2Size)+maxOffset), entries2: make([][]hashTableEntryG1KG2V,

		1<<log2Size), log2Size: log2Size, oddSalt1: rand.Uint64() | 1,
		oddSalt2: rand.Uint64() | 1, count: 0}
}

func Get[T any](slice []T, index int) *T {
	return (*T)(unsafe.Pointer(uintptr(unsafe.Pointer(unsafe.SliceData(slice))) + uintptr(index)*unsafe.Sizeof(*new(T))))
}

func IsPowerOf2[T int | uint64](x T) bool {
	x64 := uint64(x)
	return x64 != 0 && (x64&(x64-1)) == 0
}

func LogFloor(x uint64) int {
	if x == 0 {
		panic("Log2(0) is undefined")
	}
	return 63 - bits.LeadingZeros64(x)
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
	arr := make([]segmentTreeNodeG1valueG2update,

		2*n)
	for i := range size {
		*Get(arr, n+i) = segmentTreeNodeG1stValueG2lazy{init(i), zeroUpdate}
	}
	for i := size; i <
		n; i++ {
		*Get(arr, n+i) = segmentTreeNodeG1stValueG2lazy{zeroValue,
			zeroUpdate}
	}
	for i := n - 1; i > 0; i-- {

		*Get(arr, i) = segmentTreeNodeG1stValueG2lazy{(Get(arr, 2*i).value).Merge(Get(arr, 2*i+1).value),

			zeroUpdate,
		}
	}
	return &SegmentTreeG1stValueG2lazy{
		log2n: log2n,
		n:     n, arr: arr,
		zeroValue:  zeroValue,
		zeroUpdate: zeroUpdate,
	}
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

func SolveA(
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
			minA: math.MaxInt,
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

func isWhite(c byte) bool {
	return c == ' ' || c == '\n' || c == '\r' || c == '\t'
}

func GetArr[T any](slice []T, offset int) *[8]T {
	data := unsafe.Add(unsafe.Pointer(unsafe.SliceData(slice)), uintptr(offset)*unsafe.Sizeof(*new(T)))
	return (*[8]T)(data)
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

func (hm *HashTableG1KG2HG3V) Get2(key K) (V, bool) {
	hash := hm.hash(key)
	index1 := hm.index1(hash)
	arr := GetArr(hm.entries1, int(index1))
	var zero V
	for _, val := range arr {
		if val.hash == 0 {
			return zero, false
		}
		if (!checkHashFirst || val.hash == hash) && val.key == key {
			return val.value, true
		}
	}
	index2 := hm.index2(hash)
	for _, val := range *Get(hm.entries2, int(index2)) {
		if (!checkHashFirst || val.hash == hash) && val.key == key {
			return val.value, true
		}
	}
	return zero, false
}

func (hm *HashTableG1KG2HG3V,

) Get2(key K,

) (V,

	bool) {
	hash :=
		hm.hash(key)
	index1 := hm.index1(hash)
	arr := GetArr(
		hm.entries1,
		int(index1))
	var zero V

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
	for _, val := range *Get(hm.entries2, int(index2)) {
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

func (hm *HashTableG1intG2intHasherG3int,

) Get2(key int,

) (int,

	bool) {
	hash :=
		hm.hash(key)
	index1 := hm.index1(hash)
	arr := GetArr(
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
	for _, val := range *Get(hm.entries2, int(index2)) {
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

) Get2(key int,

) (int,

	bool) {
	hash :=
		hm.hash(key)
	index1 := hm.index1(hash)
	arr := GetArr(
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
	for _, val := range *Get(hm.entries2, int(index2)) {
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

func (input *input) Read(stdin *Reader) {
	input.N = stdin.Int()
	input.S = stdin.String()
}

func (st SegmentTreeG1valueG2update) SetValue(
	i int,
	val value,
) {
	if i < 0 || i >= st.n {
		panic("index out of bounds")
	}
	st.pushUpdates(i)
	(*Get(st.arr, i+st.n)).value = val
	(*Get(st.arr, i+st.n)).update = st.zeroUpdate
	st.rebuild(i)
}

func (st SegmentTreeG1valueG2update,

) SetValue(i int, val value,

) {
	if i <
		0 || i >=
		st.n {
		panic("index out of bounds")
	}
	st.pushUpdates(i)
	(*Get(st.
		arr, i+st.n)).
		value = val
	(*Get(st.arr, i+
		st.n)).update = st.zeroUpdate
	st.rebuild(i)
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
	(*Get(st.
		arr, i+st.n)).
		value = val
	(*Get(st.arr, i+
		st.n)).update = st.zeroUpdate
	st.rebuild(i)
}

func (hm *HashTableG1KG2HG3V) hash(key K) uint64 {
	val := (*(*H)(unsafe.Pointer(&key))).Hash()
	if val == 0 {
		return 1
	} else {
		return val
	}
}

func (hm *HashTableG1KG2HG3V,

) hash(key K,

) uint64 {
	val := (*(*H)(unsafe.
		Pointer(&key))).Hash()
	if val ==
		0 {
		return 1
	} else {
		return val
	}
}

func (hm *HashTableG1intG2intHasherG3int,

) hash(key int,

) uint64 {
	val := (*(*H)(unsafe.
		Pointer(&key))).Hash()
	if val ==
		0 {
		return 1
	} else {
		return val
	}
}

func (hm *HashTableG1intG2intHashG3int,

) hash(key int,

) uint64 {
	val := (*(*H)(unsafe.
		Pointer(&key))).Hash()
	if val ==
		0 {
		return 1
	} else {
		return val
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

func (up lazy) ApplyUpdate(val *stValue) {
	val.minDP += up.addDP
}

func (x intHash) Hash() uint64 {
	return uint64(x)
}

func (x intHasher) Hash() uint64 {
	return uint64(x)
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

func (hm *HashTableG1KG2HG3V) Set(key K, value V) {
	if hm.count*2 >= (1 << hm.log2Size) {
		hm.resize()
	}
	hash := hm.hash(key)
	index1 := hm.index1(hash)
	arr := GetArr(hm.entries1, int(index1))
	for i := range maxOffset {
		e := &arr[i]
		if e.hash == 0 {
			*e = hashTableEntryG1KG2V{
				hash:  hash,
				key:   key,
				value: value,
			}
			hm.count++
			return
		} else if (!checkHashFirst || e.hash == hash) && e.key == key {
			e.value = value
			return
		}
	}
	index2 := hm.index2(hash)
	entries := Get(hm.entries2, int(index2))
	for i, val := range *entries {
		if val.hash == hash && val.key == key {
			Get(*entries, i).value = value
			return
		}
	}
	*entries = append(*entries, hashTableEntryG1KG2V{
		hash:  hash,
		key:   key,
		value: value,
	})
	hm.count++
}

func (hm *HashTableG1KG2HG3V,

) Set(key K,

	value V,

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
	arr := GetArr(hm.entries1,
		int(index1),
	)
	for i := range maxOffset {
		e := &arr[i]
		if e.hash ==
			0 {
			*e = hashTableEntryG1KG2V{hash: hash, key: key, value: value}
			hm.count++

			return
		} else if (!checkHashFirst || e.hash == hash) && e.key == key {
			e.value = value
			return
		}
	}
	index2 := hm.index2(hash)
	entries := Get(hm.entries2, int(index2))
	for i, val := range *entries {
		if val.hash == hash &&
			val.key == key {
			Get(*entries, i).value = value
			return
		}
	}

	*entries = append(*entries, hashTableEntryG1KG2V{hash: hash,
		key: key, value: value,
	})
	hm.
		count++
}

func (hm *HashTableG1intG2intHasherG3int,

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
	arr := GetArr(hm.entries1,
		int(index1),
	)
	for i := range maxOffset {
		e := &arr[i]
		if e.hash ==
			0 {
			*e = hashTableEntryG1KG2V{hash: hash, key: key, value: value}
			hm.count++

			return
		} else if (!checkHashFirst || e.hash == hash) && e.key == key {
			e.value = value
			return
		}
	}
	index2 := hm.index2(hash)
	entries := Get(hm.entries2, int(index2))
	for i, val := range *entries {
		if val.hash == hash &&
			val.key == key {
			Get(*entries, i).value = value
			return
		}
	}

	*entries = append(*entries,
		hashTableEntryG1KG2V{hash: hash,
			key: key, value: value,
		})
	hm.
		count++
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
	arr := GetArr(hm.entries1,
		int(index1),
	)
	for i := range maxOffset {
		e := &arr[i]
		if e.hash ==
			0 {
			*e = hashTableEntryG1KG2V{hash: hash, key: key, value: value}
			hm.count++

			return
		} else if (!checkHashFirst || e.hash == hash) && e.key == key {
			e.value = value
			return
		}
	}
	index2 := hm.index2(hash)
	entries := Get(hm.entries2, int(index2))
	for i, val := range *entries {
		if val.hash == hash &&
			val.key == key {
			Get(*entries, i).value = value
			return
		}
	}

	*entries = append(*entries,
		hashTableEntryG1KG2V{hash: hash,
			key: key, value: value,
		})
	hm.
		count++
}

func (hm *HashTableG1KG2HG3V) index1(hash uint64) uint64 {
	hash *= hm.oddSalt1
	hash = ReverseBits64(hash)
	return hash & (1<<hm.log2Size - 1)
}

func (hm *HashTableG1KG2HG3V,

) index1(hash uint64) uint64 {
	hash *= hm.oddSalt1
	hash = ReverseBits64(hash)
	return hash &
		(1<<
			hm.log2Size -
			1)
}

func (hm *HashTableG1intG2intHasherG3int,

) index1(hash uint64) uint64 {
	hash *= hm.oddSalt1
	hash = ReverseBits64(hash)
	return hash &
		(1<<
			hm.log2Size -
			1)
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

func (hm *HashTableG1KG2HG3V) resize() {
	hm.log2Size++
	newEntries1 := make([]hashTableEntryG1KG2V, (1<<hm.log2Size)+maxOffset)
	newEntries2 := make([][]hashTableEntryG1KG2V, 1<<hm.log2Size)
	add := func(e hashTableEntryG1KG2V) {
		hash := hm.hash(e.key)
		index1 := hm.index1(hash)
		arr := GetArr(newEntries1, int(index1))
		for j := range arr {
			if arr[j].hash == 0 {
				arr[j] = e
				return
			}
		}
		index2 := hm.index2(hash)
		entries := Get(newEntries2, int(index2))
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
	hm.entries2 = newEntries2
}

func (hm *HashTableG1KG2HG3V,

) resize() {
	hm.log2Size++
	newEntries1 := make([]hashTableEntryG1KG2V,

		(1<<hm.
			log2Size)+
			maxOffset,
	)
	newEntries2 := make([][]hashTableEntryG1KG2V,

		1<<
			hm.log2Size)
	add := func(e hashTableEntryG1KG2V) {
		hash := hm.hash(e.
			key)
		index1 := hm.index1(hash)
		arr :=
			GetArr(newEntries1, int(
				index1))
		for j := range arr {
			if arr[j].hash == 0 {
				arr[j] = e
				return
			}
		}
		index2 := hm.index2(hash)
		entries := Get(newEntries2,
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
		entries2 =
		newEntries2

}

func (hm *HashTableG1intG2intHasherG3int,

) resize() {
	hm.log2Size++
	newEntries1 := make([]hashTableEntryG1KG2V,

		(1<<hm.
			log2Size)+
			maxOffset,
	)
	newEntries2 := make([][]hashTableEntryG1KG2V,

		1<<
			hm.log2Size)
	add := func(e hashTableEntryG1KG2V) {
		hash := hm.hash(e.
			key)
		index1 := hm.index1(hash)
		arr :=
			GetArr(newEntries1, int(
				index1))
		for j := range arr {
			if arr[j].hash == 0 {
				arr[j] = e
				return
			}
		}
		index2 := hm.index2(hash)
		entries := Get(newEntries2,
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
		entries2 =
		newEntries2

}

func (hm *HashTableG1intG2intHashG3int,

) resize() {
	hm.log2Size++
	newEntries1 := make([]hashTableEntryG1KG2V,

		(1<<hm.
			log2Size)+
			maxOffset,
	)
	newEntries2 := make([][]hashTableEntryG1KG2V,

		1<<
			hm.log2Size)
	add := func(e hashTableEntryG1KG2V) {
		hash := hm.hash(e.
			key)
		index1 := hm.index1(hash)
		arr :=
			GetArr(newEntries1, int(
				index1))
		for j := range arr {
			if arr[j].hash == 0 {
				arr[j] = e
				return
			}
		}
		index2 := hm.index2(hash)
		entries := Get(newEntries2,
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
		entries2 =
		newEntries2

}

func (r *Reader) seek() {
	r.from++
}

func (st SegmentTreeG1valueG2update) LongestRangeWherePredicate(
	rMax int,
	predicate func(value) bool,
) (int, bool) {
	if rMax < 0 || rMax >= st.n {
		panic("invalid rMax")
	}

	st.pushUpdates(rMax)
	i := rMax + st.n
	if !predicate(Get(st.arr, i).value) {
		return -1, false
	}
	summedValue := st.zeroValue
	for i > 0 {
		arrVal := Get(st.arr, i)
		(arrVal.update).ApplyUpdate(&arrVal.value)
		val := arrVal.value.Merge(summedValue)
		if predicate(val) {
			if IsPowerOf2(i) {
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

	upd := Get(st.arr, i).update
	for i < st.n {
		val := Get(st.arr, 2*i+1)
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

func (st SegmentTreeG1valueG2update,

) LongestRangeWherePredicate(rMax int,
	predicate func(value) bool) (int,
	bool) {
	if rMax <
		0 || rMax >=

		st.n {
		panic("invalid rMax")
	}
	st.
		pushUpdates(rMax)
	i := rMax + st.n
	if !predicate(Get(st.
		arr,
		i).value) {
		return -1, false
	}
	summedValue :=
		st.
			zeroValue
	for i > 0 {
		arrVal := Get(st.
			arr,

			i)
		(arrVal.update).ApplyUpdate(&arrVal.value)
		val := arrVal.value.
			Merge(summedValue)
		if predicate(val) {

			if IsPowerOf2(i) {
				return 0, true
			}
			if i%2 == 0 {
				i = i/2 - 1
				summedValue =
					val.Merge(summedValue)
			} else {
				i = i / 2
			}
		} else {
			break
		}
	}
	upd := Get(st.
		arr, i).update
	for i < st.n {
		val := Get(st.arr,
			2*
				i+1)
		(upd).
			Push(&val.
				update,
			)

		(upd).ApplyUpdate(&val.
			value,
		)
		combined := val.
			value.Merge(summedValue)
		if predicate(combined) {
			summedValue = combined

			i = 2 * i
		} else {
			upd = val.
				update
			i = 2*i +
				1
		}
	}
	return i - st.n + 1, true
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
	if !predicate(Get(st.
		arr,
		i).value) {
		return -1, false
	}
	summedValue :=
		st.
			zeroValue
	for i > 0 {
		arrVal := Get(st.
			arr,

			i)
		(arrVal.update).ApplyUpdate(&arrVal.value)
		val := arrVal.value.
			Merge(summedValue)
		if predicate(val) {

			if IsPowerOf2(i) {
				return 0, true
			}
			if i%2 == 0 {
				i = i/2 - 1
				summedValue =
					val.Merge(summedValue)
			} else {
				i = i / 2
			}
		} else {
			break
		}
	}
	upd := Get(st.
		arr, i).update
	for i < st.n {
		val := Get(st.arr,
			2*
				i+1)
		(upd).
			Push(&val.
				update,
			)

		(upd).ApplyUpdate(&val.
			value,
		)
		combined := val.
			value.Merge(summedValue)
		if predicate(combined) {
			summedValue = combined

			i = 2 * i
		} else {
			upd = val.
				update
			i = 2*i +
				1
		}
	}
	return i - st.n + 1, true
}

func (a stValue) Merge(b stValue) stValue {
	return stValue{minA: min(a.minA, b.minA), minDP: min(a.minDP, b.minDP)}
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

func (r *Reader) peek() (byte, bool) {
	if r.from == r.to {
		if !r.read() {
			return 0, false
		}
	}
	return r.bytes[r.from], true
}

func (top lazy) Push(existing *lazy) {
	existing.addDP += top.addDP
}

func (hm *HashTableG1KG2HG3V) index2(hash uint64) uint64 {
	hash *= hm.oddSalt2
	hash = ReverseBits64(hash)
	return hash & (1<<hm.log2Size - 1)
}

func (hm *HashTableG1KG2HG3V,

) index2(hash uint64) uint64 {
	hash *= hm.oddSalt2
	hash = ReverseBits64(hash)
	return hash &
		(1<<
			hm.log2Size -
			1)
}

func (hm *HashTableG1intG2intHasherG3int,

) index2(hash uint64) uint64 {
	hash *= hm.oddSalt2
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

func (st SegmentTreeG1valueG2update) pushUpdates(
	i int,
) {
	i += st.n
	for shift := st.log2n; shift > 0; shift-- {
		i := i >> shift
		update := Get(st.arr, i).update

		Get(st.arr, i).update = st.zeroUpdate
		(update).ApplyUpdate(&Get(st.arr, i).value)

		(update).Push(&Get(st.arr, 2*i).update)
		(update).Push(&Get(st.arr, 2*i+1).update)
	}
	(Get(st.arr, i).update).ApplyUpdate(&Get(st.arr, i).value)
	Get(st.arr, i).update = st.zeroUpdate
}

func (st SegmentTreeG1valueG2update,

) pushUpdates(i int) {
	i +=
		st.n
	for shift := st.
		log2n; shift > 0; shift-- {
		i := i >>
			shift
		update := Get(st.arr,
			i).update
		Get(st.arr,
			i).update = st.zeroUpdate
		(update).ApplyUpdate(&Get(
			st.
				arr, i).value)
		(update).Push(&Get(
			st.arr, 2*i).update,
		)
		(update).Push(&Get(st.
			arr, 2*i+1).update,
		)
	}
	(Get(st.arr,
		i).update).ApplyUpdate(&Get(st.arr, i).value)
	Get(st.arr,
		i).update = st.
		zeroUpdate
}

func (st SegmentTreeG1stValueG2lazy,

) pushUpdates(i int) {
	i +=
		st.n
	for shift := st.
		log2n; shift > 0; shift-- {
		i := i >>
			shift
		update := Get(st.arr,
			i).update
		Get(st.arr,
			i).update = st.zeroUpdate
		(update).ApplyUpdate(&Get(
			st.
				arr, i).value)
		(update).Push(&Get(
			st.arr, 2*i).update,
		)
		(update).Push(&Get(st.
			arr, 2*i+1).update,
		)
	}
	(Get(st.arr,
		i).update).ApplyUpdate(&Get(st.arr, i).value)
	Get(st.arr,
		i).update = st.
		zeroUpdate
}

func (st SegmentTreeG1valueG2update) rebuild(
	i int,
) {
	i += st.n
	i /= 2
	for i != 0 {
		left := Get(st.arr, 2*i)
		(left.update).ApplyUpdate(&left.value)
		right := Get(st.arr, 2*i+1)
		(right.update).ApplyUpdate(&right.value)
		Get(st.arr, i).value = left.value.Merge(right.value)
		i = i / 2
	}
}

func (st SegmentTreeG1valueG2update,

) rebuild(i int) {
	i += st.
		n
	i /= 2
	for i != 0 {
		left := Get(st.arr,
			2*i)
		(left.
			update).ApplyUpdate(&left.value)
		right := Get(st.arr,
			2*i+1)
		(right.update).ApplyUpdate(&right.value)
		Get(
			st.
				arr, i).value = left.value.Merge(right.
			value)
		i = i /
			2
	}
}

func (st SegmentTreeG1stValueG2lazy,

) rebuild(i int) {
	i += st.
		n
	i /= 2
	for i != 0 {
		left := Get(st.arr,
			2*i)
		(left.
			update).ApplyUpdate(&left.value)
		right := Get(st.arr,
			2*i+1)
		(right.update).ApplyUpdate(&right.value)
		Get(
			st.
				arr, i).value = left.value.Merge(right.
			value)
		i = i /
			2
	}
}

