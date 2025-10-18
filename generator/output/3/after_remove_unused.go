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
	fromFile   = false
	inputFile  = "input.txt"
	outputFile = "output.txt"
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

var solveX = solveC

func (w *Writer) Fprintf(format string, a ...any) {
	w.String(fmt.Sprintf(format, a...))
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

type (
	HashTableG1 struct {
		entries1 []hashTableEntryG1
		entries2 [][]hashTableEntryG1
		log2Size int
		oddSalt1 uint64
		oddSalt2 uint64
		count    int
	}

	SegmentTreeG1 struct {
		log2n      int
		n          int
		arr        []segmentTreeNodeG1
		zeroValue  stValue
		zeroUpdate lazy
	}
	intHash int
	lazy    struct {
		addDP int
	}

	segmentTreeNodeG1 struct {
		value  stValue
		update lazy
	}
	HashTableG2 struct {
		entries1 []hashTableEntryG1
		entries2 [][]hashTableEntryG1
		log2Size int
		oddSalt1 uint64
		oddSalt2 uint64
		count    int
	}

	hashTableEntryG1 struct {
		hash  uint64
		key   int
		value int
	}
	intHasher int
	stValue   struct {
		minA  int
		minDP int
	}
)

const (
	maxOffset      = 8
	checkHashFirst = false
)

func GetArrG1(slice []hashTableEntryG1, offset int) *[8]hashTableEntryG1 {
	data := unsafe.Add(unsafe.Pointer(unsafe.SliceData(slice)), uintptr(offset)*unsafe.Sizeof(*new(hashTableEntryG1)))
	return (*[8]hashTableEntryG1)(data)
}

func solveTestC(
	stdin *Reader,
	stdout *Writer,
) {
	n := stdin.Int()
	a := NewHashTableG1(0)
	for i := range n {
		a.Set(i, stdin.Int())
	}
	st := NewSegmentTreeG1(
		func(i int) stValue {
			return stValue{
				minA:  a.GetG1(i),
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

	dp := NewHashTableG1(0)

	for i := range n {
		ai := a.GetG1(i)
		dp.Set(i, i+1)
		for cost := 1; cost <= 3; cost++ {
			l, _ := st.LongestRangeWherePredicate(i, func(x stValue) bool {
				return x.minA*cost >= ai
			})
			var total int
			if l == 0 {
				total = cost
			} else {
				total = st.GetG1(l-1, i-1).minDP + cost
			}
			dp.Set(i, min(dp.GetG1(i), total))
		}
		st.SetValue(i, stValue{
			minA:  ai,
			minDP: dp.GetG1(i),
		})
	}
	stdout.Int(dp.GetG1(n-1), '\n')
}

func GetG3(slice []segmentTreeNodeG1, index int) *segmentTreeNodeG1 {
	return (*segmentTreeNodeG1)(unsafe.Pointer(uintptr(unsafe.Pointer(unsafe.SliceData(slice))) + uintptr(index)*unsafe.Sizeof(*new(segmentTreeNodeG1))))
}

func NewSegmentTreeG1(
	init func(int) stValue,
	size int,
	zeroValue stValue,
	zeroUpdate lazy,
) *SegmentTreeG1 {
	if size <= 0 {
		panic("size must be positive")
	}
	log2n := LogCeilG1(size)
	n := 1 << log2n
	arr := make([]segmentTreeNodeG1, 2*n)
	for i := range size {
		*GetG1(arr, n+i) = segmentTreeNodeG1{init(i), zeroUpdate}
	}
	for i := size; i < n; i++ {
		*GetG1(arr, n+i) = segmentTreeNodeG1{zeroValue, zeroUpdate}
	}
	for i := n - 1; i > 0; i-- {
		*GetG1(arr, i) = segmentTreeNodeG1{(GetG1(arr, 2*i).value).Merge(GetG1(arr, 2*i+1).value), zeroUpdate}
	}
	return &SegmentTreeG1{
		log2n:      log2n,
		n:          n,
		arr:        arr,
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

func GetG1(slice [][]hashTableEntryG1, index int) *[]hashTableEntryG1 {
	return (*[]hashTableEntryG1)(unsafe.Pointer(uintptr(unsafe.Pointer(unsafe.SliceData(slice))) + uintptr(index)*unsafe.Sizeof(*new([]hashTableEntryG1))))
}

func LogCeilG1(x int) int {
	x64 := uint64(x)
	if x64 == 0 {
		panic("Log2(0) is undefined")
	}
	if x64 == 1 {
		return 0
	}
	return 1 + LogFloorG1(x64-1)
}

func LogFloorG1(x int) int {
	x64 := uint64(x)
	if x64 == 0 {
		panic("Log2(0) is undefined")
	}
	return 63 - bits.LeadingZeros64(x64)
}

func NewHashTableG1(
	size int,
) *HashTableG1 {
	log2Size := LogCeilG1(size*2 + 1)
	return &HashTableG1{
		entries1: make([]hashTableEntryG1, (1<<log2Size)+maxOffset),
		entries2: make([][]hashTableEntryG1, 1<<log2Size),
		log2Size: log2Size,
		oddSalt1: rand.Uint64() | 1,
		oddSalt2: rand.Uint64() | 1,
		count:    0,
	}
}

func solveC(
	stdin *Reader,
	stdout *Writer,
) {
	t := stdin.Int()
	for range t {
		solveTestC(stdin, stdout)
	}
}

func GetG2(slice []hashTableEntryG1, index int) *hashTableEntryG1 {
	return (*hashTableEntryG1)(unsafe.Pointer(uintptr(unsafe.Pointer(unsafe.SliceData(slice))) + uintptr(index)*unsafe.Sizeof(*new(hashTableEntryG1))))
}

func IsPowerOf2G1(x int) bool {
	x64 := uint64(x)
	return x64 != 0 && (x64&(x64-1)) == 0
}

func (up lazy) ApplyUpdate(val *stValue) {
	val.minDP += up.addDP
}

func (h intHash) Hash() uint64 {
	return uint64(h)
}

func (x intHasher) Hash() uint64 {
	return uint64(x)
}

func (hm *HashTableG1) Set(key int, value int) {
	if hm.count*2 >= (1 << hm.log2Size) {
		hm.resize()
	}
	hash := hm.hash(key)
	index1 := hm.index1(hash)
	arr := GetArrG1(hm.entries1, int(index1))
	for i := range maxOffset {
		e := &arr[i]
		if e.hash == 0 {
			*e = hashTableEntryG1{
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
	entries := GetG1(hm.entries2, int(index2))
	for i, val := range *entries {
		if val.hash == hash && val.key == key {
			GetG2(*entries, i).value = value
			return
		}
	}
	*entries = append(*entries, hashTableEntryG1{
		hash:  hash,
		key:   key,
		value: value,
	})
	hm.count++
}

func (hm *HashTableG2) Set(key int, value int) {
	if hm.count*2 >= (1 << hm.log2Size) {
		hm.resize()
	}
	hash := hm.hash(key)
	index1 := hm.index1(hash)
	arr := GetArrG1(hm.entries1, int(index1))
	for i := range maxOffset {
		e := &arr[i]
		if e.hash == 0 {
			*e = hashTableEntryG1{
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
	entries := GetG1(hm.entries2, int(index2))
	for i, val := range *entries {
		if val.hash == hash && val.key == key {
			GetG2(*entries, i).value = value
			return
		}
	}
	*entries = append(*entries, hashTableEntryG1{
		hash:  hash,
		key:   key,
		value: value,
	})
	hm.count++
}

func (r *Reader) peek() (byte, bool) {
	if r.from == r.to {
		if !r.read() {
			return 0, false
		}
	}
	return r.bytes[r.from], true
}

func (hm *HashTableG1) Get2(key int) (int, bool) {
	hash := hm.hash(key)
	index1 := hm.index1(hash)
	arr := GetArrG1(hm.entries1, int(index1))
	var zero int
	for _, val := range arr {
		if val.hash == 0 {
			return zero, false
		}
		if (!checkHashFirst || val.hash == hash) && val.key == key {
			return val.value, true
		}
	}
	index2 := hm.index2(hash)
	for _, val := range *GetG1(hm.entries2, int(index2)) {
		if (!checkHashFirst || val.hash == hash) && val.key == key {
			return val.value, true
		}
	}
	return zero, false
}

func (hm *HashTableG2) Get2(key int) (int, bool) {
	hash := hm.hash(key)
	index1 := hm.index1(hash)
	arr := GetArrG1(hm.entries1, int(index1))
	var zero int
	for _, val := range arr {
		if val.hash == 0 {
			return zero, false
		}
		if (!checkHashFirst || val.hash == hash) && val.key == key {
			return val.value, true
		}
	}
	index2 := hm.index2(hash)
	for _, val := range *GetG1(hm.entries2, int(index2)) {
		if (!checkHashFirst || val.hash == hash) && val.key == key {
			return val.value, true
		}
	}
	return zero, false
}

func (hm *HashTableG1) hash(key int) uint64 {
	val := (*(*intHash)(unsafe.Pointer(&key))).Hash()
	if val == 0 {
		return 1
	} else {
		return val
	}
}

func (hm *HashTableG2) hash(key int) uint64 {
	val := (*(*intHasher)(unsafe.Pointer(&key))).Hash()
	if val == 0 {
		return 1
	} else {
		return val
	}
}

func (hm *HashTableG1) index2(hash uint64) uint64 {
	hash *= hm.oddSalt2
	hash = ReverseBits64(hash)
	return hash & (1<<hm.log2Size - 1)
}

func (hm *HashTableG2) index2(hash uint64) uint64 {
	hash *= hm.oddSalt2
	hash = ReverseBits64(hash)
	return hash & (1<<hm.log2Size - 1)
}

func (st SegmentTreeG1) pushUpdates(
	i int,
) {
	i += st.n
	for shift := st.log2n; shift > 0; shift-- {
		i := i >> shift
		update := GetG3(st.arr, i).update

		GetG3(st.arr, i).update = st.zeroUpdate
		(update).ApplyUpdate(&GetG3(st.arr, i).value)

		(update).Push(&GetG3(st.arr, 2*i).update)
		(update).Push(&GetG3(st.arr, 2*i+1).update)
	}
	(GetG3(st.arr, i).update).ApplyUpdate(&GetG3(st.arr, i).value)
	GetG3(st.arr, i).update = st.zeroUpdate
}

func (st SegmentTreeG1) rebuild(
	i int,
) {
	i += st.n
	i /= 2
	for i != 0 {
		left := GetG3(st.arr, 2*i)
		(left.update).ApplyUpdate(&left.value)
		right := GetG3(st.arr, 2*i+1)
		(right.update).ApplyUpdate(&right.value)
		GetG3(st.arr, i).value = left.value.Merge(right.value)
		i = i / 2
	}
}

func (a stValue) Merge(b stValue) stValue {
	return stValue{minA: min(a.minA, b.minA), minDP: min(a.minDP, b.minDP)}
}

func (top lazy) Push(existing *lazy) {
	existing.addDP += top.addDP
}

func (st SegmentTreeG1) SetValue(
	i int,
	val stValue,
) {
	if i < 0 || i >= st.n {
		panic("index out of bounds")
	}
	st.pushUpdates(i)
	(*GetG3(st.arr, i+st.n)).value = val
	(*GetG3(st.arr, i+st.n)).update = st.zeroUpdate
	st.rebuild(i)
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

func (hm *HashTableG1) resize() {
	hm.log2Size++
	newEntries1 := make([]hashTableEntryG1, (1<<hm.log2Size)+maxOffset)
	newEntries2 := make([][]hashTableEntryG1, 1<<hm.log2Size)
	add := func(e hashTableEntryG1) {
		hash := hm.hash(e.key)
		index1 := hm.index1(hash)
		arr := GetArrG1(newEntries1, int(index1))
		for j := range arr {
			if arr[j].hash == 0 {
				arr[j] = e
				return
			}
		}
		index2 := hm.index2(hash)
		entries := GetG1(newEntries2, int(index2))
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

func (hm *HashTableG2) resize() {
	hm.log2Size++
	newEntries1 := make([]hashTableEntryG1, (1<<hm.log2Size)+maxOffset)
	newEntries2 := make([][]hashTableEntryG1, 1<<hm.log2Size)
	add := func(e hashTableEntryG1) {
		hash := hm.hash(e.key)
		index1 := hm.index1(hash)
		arr := GetArrG1(newEntries1, int(index1))
		for j := range arr {
			if arr[j].hash == 0 {
				arr[j] = e
				return
			}
		}
		index2 := hm.index2(hash)
		entries := GetG1(newEntries2, int(index2))
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

func (st SegmentTreeG1) LongestRangeWherePredicate(
	rMax int,
	predicate func(stValue) bool,
) (int, bool) {
	if rMax < 0 || rMax >= st.n {
		panic("invalid rMax")
	}

	st.pushUpdates(rMax)
	i := rMax + st.n
	if !predicate(GetG3(st.arr, i).value) {
		return -1, false
	}
	summedValue := st.zeroValue
	for i > 0 {
		arrVal := GetG3(st.arr, i)
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

	upd := GetG3(st.arr, i).update
	for i < st.n {
		val := GetG3(st.arr, 2*i+1)
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

func (hm *HashTableG1) index1(hash uint64) uint64 {
	hash *= hm.oddSalt1
	hash = ReverseBits64(hash)
	return hash & (1<<hm.log2Size - 1)
}

func (hm *HashTableG2) index1(hash uint64) uint64 {
	hash *= hm.oddSalt1
	hash = ReverseBits64(hash)
	return hash & (1<<hm.log2Size - 1)
}

func (r *Reader) seek() {
	r.from++
}

