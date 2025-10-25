package main

import (
	"fmt"
	"math"
	"math/bits"
	"math/rand/v2"
	"os"
	"runtime"
	"strconv"
	"sync"
	"sync/atomic"
	"testing"
	"time"
	"unsafe"
)

//package main
//file ..//go

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
	defer Recover()

	//Hackercup(stdin, stdout)
	SolveX(stdin, stdout)
}

/*input
6
7
ABBAAAB
1
A
1
B
2
AB
6
AAAAAA
7
BBBBBBA

*/

/*output
Case #1: Alice
Case #2: Alice
Case #3: Bob
Case #4: Bob
Case #5: Alice
Case #6: Alice

*/
//package main
//file ..//solve_A.go

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
	a := NewHashTable[int, intHash, int](0)
	for i := range n {
		a.Set(i, stdin.Int())
	}
	st := NewSegmentTree[stValue, lazy](
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

	dp := NewHashTable[int, intHash, int](0)

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

/*input
4
5
3 1 4 1 5
10
9 2 6 5 3 5 8 9 7 9
8
1 2 3 4 5 6 7 8
2
1 1000000000000000000

*/

/*output
2
4
5
2

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

func (w *Writer) Printf(format string, a ...any) {
	w.String(fmt.Sprintf(format, a...))
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

func (w *Writer) Float(f float64) {
	str := strconv.FormatFloat(f, 'f', -1, 64)
	w.bytes([]byte(str))
}

//package hash_map
//file ..//hash_map/go

const (
	maxOffset      = 8
	checkHashFirst = false
)

type hashTableEntry[K comparable, V any] struct {
	hash  uint64
	key   K
	value V
}

type Hasher interface {
	Hash() uint64
}

type HashTable[K comparable, H Hasher, V any] struct {
	entries1 []hashTableEntry[K, V]
	entries2 [][]hashTableEntry[K, V]
	log2Size int
	oddSalt1 uint64
	oddSalt2 uint64
	count    int
}

func NewHashTable[K comparable, H Hasher, V any](
	size int,
) *HashTable[K, H, V] {
	log2Size := LogCeil(uint64(size*2 + 1))
	return &HashTable[K, H, V]{
		entries1: make([]hashTableEntry[K, V], (1<<log2Size)+maxOffset),
		entries2: make([][]hashTableEntry[K, V], 1<<log2Size),
		log2Size: log2Size,
		oddSalt1: rand.Uint64() | 1,
		oddSalt2: rand.Uint64() | 1,
		count:    0,
	}
}

func (hm *HashTable[K, H, V]) hash(key K) uint64 {
	val := (*(*H)(unsafe.Pointer(&key))).Hash()
	if val == 0 {
		return 1
	} else {
		return val
	}
}

func (hm *HashTable[K, H, V]) index1(hash uint64) uint64 {
	hash *= hm.oddSalt1
	hash = ReverseBits64(hash)
	return hash & (1<<hm.log2Size - 1)
}

func (hm *HashTable[K, H, V]) index2(hash uint64) uint64 {
	hash *= hm.oddSalt2
	hash = ReverseBits64(hash)
	return hash & (1<<hm.log2Size - 1)
}

func (hm *HashTable[K, H, V]) Get(key K) V {
	val, ok := hm.Get2(key)
	if !ok {
		var zero V
		return zero
	}
	return val
}

func (hm *HashTable[K, H, V]) Get2(key K) (V, bool) {
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

func (hm *HashTable[K, H, V]) Delete(key K) bool {
	hash := hm.hash(key)
	index1 := hm.index1(hash)
	arr := GetArr(hm.entries1, int(index1))
	for i := range maxOffset {
		e := &arr[i]
		if e.hash == 0 {
			return false
		}
		if (!checkHashFirst || e.hash == hash) && e.key == key {
			e.hash = 0
			hm.count--
			return true
		}
	}
	index2 := hm.index2(hash)
	slice := *Get(hm.entries2, int(index2))
	for i := range slice {
		val := &slice[i]
		if (!checkHashFirst || val.hash == hash) && val.key == key {
			*val = *Get(slice, len(slice)-1)
			slice = slice[:len(slice)-1]
			*Get(hm.entries2, int(index2)) = slice
			hm.count--
			return true
		}
	}
	return false
}

func (hm *HashTable[K, H, V]) Set(key K, value V) {
	if hm.count*2 >= (1 << hm.log2Size) {
		hm.resize()
	}
	hash := hm.hash(key)
	index1 := hm.index1(hash)
	arr := GetArr(hm.entries1, int(index1))
	for i := range maxOffset {
		e := &arr[i]
		if e.hash == 0 {
			*e = hashTableEntry[K, V]{
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
	*entries = append(*entries, hashTableEntry[K, V]{
		hash:  hash,
		key:   key,
		value: value,
	})
	hm.count++
}

func (hm *HashTable[K, H, V]) resize() {
	hm.log2Size++
	newEntries1 := make([]hashTableEntry[K, V], (1<<hm.log2Size)+maxOffset)
	newEntries2 := make([][]hashTableEntry[K, V], 1<<hm.log2Size)
	add := func(e hashTableEntry[K, V]) {
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

//package hash_map
//file ..//hash_map/hash_map_test.go

type intHasher int

func (x intHasher) Hash() uint64 {
	return uint64(x)
}

func BenchmarkHashMap(b *testing.B) {
	a := make([]int, b.N)
	for i := 0; i < b.N; i++ {
		a[i] = rand.Int()
	}
	b.ResetTimer()
	hashMap := NewHashTable[int, intHasher, int](b.N)
	for range b.N {
		for _, a := range a {
			if rand.IntN(2) == 0 {
				hashMap.Set(a, a)
			} else {
				hashMap.Get(a)
			}
		}
	}
}

func BenchmarkBuiltinMap(b *testing.B) {
	a := make([]int, b.N)
	for i := 0; i < b.N; i++ {
		a[i] = i*37 + 17
	}
	b.ResetTimer()
	m := make(map[int]int, b.N)
	for range b.N {
		for _, a := range a {
			if rand.IntN(2) == 0 {
				m[a] = a
			} else {
				_ = m[a]
			}
		}
	}
}

//package debug
//file ..//debug/go

func Go(fn func()) {
	go func() {
		defer Recover()
		fn()
	}()
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

func Hackercup(
	stdin *Reader,
	stdout *Writer,
) {
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

//package hackercup
//file ..//hackercup/io.go

type input struct {
	N int
	S string
}

func (input *input) Read(stdin *Reader) {
	input.N = stdin.Int()
	input.S = stdin.String()
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

//package hackercup
//file ..//hackercup/solve.go

func solve(test int, input *input) output {
	s := []byte("X" + input.S + "X")
	N := input.N
	al := make([]int, N+2)
	br := make([]int, N+2)
	queue := make([][]Pair[int, int], N+4)
	for i := 0; i < N+2; i++ {
		al[i] = 0
		br[i] = N + 1
		queue[i+1] = append(queue[i+1], Pair[int, int]{X: 0, Y: i})
		queue[N+2-i] = append(queue[N+2-i], Pair[int, int]{X: 1, Y: i})
	}
	for i := 1; i <= N; i++ {
		if s[i] == 'A' {
			al[i] = i
			queue[1] = append(queue[1], Pair[int, int]{X: 0, Y: i})
		} else {
			br[i] = i
			queue[1] = append(queue[1], Pair[int, int]{X: 1, Y: i})
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
						queue[dist+1] = append(queue[dist+1], Pair[int, int]{X: 0, Y: r + 1})
					}
				} else if l != r {
					if br[l+1] > r+1 {
						br[l+1] = r + 1
						queue[dist] = append(queue[dist], Pair[int, int]{X: 1, Y: l + 1})
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
						queue[dist+1] = append(queue[dist+1], Pair[int, int]{X: 1, Y: l - 1})
					}
				} else if l != r {
					if al[r-1] < l-1 {
						al[r-1] = l - 1
						queue[dist] = append(queue[dist], Pair[int, int]{X: 0, Y: r - 1})
					}
				}
			}
		}
	}

	return output{
		alice: al[N] >= 1,
	}
}

func arit(a, b int) int {
	avg2 := (a + b)
	return (avg2 * (b - a + 1)) / 2
}

func matrixPower(m [][]int, p int) [][]int {
	n := len(m)
	ret := make([][]int, n)
	for i := range n {
		ret[i] = make([]int, n)
		ret[i][i] = 1
	}
	for p != 0 {
		if p%2 == 1 {
			ret = matrixMultiply(ret, m)
		}
		m = matrixMultiply(m, m)
		p /= 2
	}
	return ret
}

func matrixMultiply(a, b [][]int) [][]int {
	n := len(a)
	result := make([][]int, n)
	for i := range n {
		result[i] = make([]int, n)
	}
	for i := range n {
		for j := range n {
			if a[i][j] == 0 {
				continue
			}
			for k := range n {
				result[i][k] += a[i][j] * b[j][k]
				if result[i][k] >= mod {
					result[i][k] %= mod
				}
			}
		}
	}
	return result
}

const mod = 1_000_000_007

//package segment_tree_iter
//file ..//segment_tree_iter/go

type segmentTreeNode[value, update any] struct {
	value  value
	update update
}

type SegmentTree[
	value interface{ Merge(value) value },
	update interface {
		ApplyUpdate(*value)
		Push(*update)
	},
] struct {
	log2n      int
	n          int // power of two, number of leaves
	arr        []segmentTreeNode[value, update]
	zeroValue  value
	zeroUpdate update
}

func NewSegmentTree[
	value interface{ Merge(value) value },
	update interface {
		ApplyUpdate(*value)
		Push(*update)
	},
](
	init func(int) value,
	size int,
	zeroValue value,
	zeroUpdate update,
) *SegmentTree[value, update] {
	if size <= 0 {
		panic("size must be positive")
	}
	log2n := LogCeil(uint64(size))
	n := 1 << log2n
	arr := make([]segmentTreeNode[value, update], 2*n)
	for i := range size {
		*Get(arr, n+i) = segmentTreeNode[value, update]{init(i), zeroUpdate}
	}
	for i := size; i < n; i++ {
		*Get(arr, n+i) = segmentTreeNode[value, update]{zeroValue, zeroUpdate}
	}
	for i := n - 1; i > 0; i-- {
		*Get(arr, i) = segmentTreeNode[value, update]{(Get(arr, 2*i).value).Merge(Get(arr, 2*i+1).value), zeroUpdate}
	}
	return &SegmentTree[value, update]{
		log2n:      log2n,
		n:          n,
		arr:        arr,
		zeroValue:  zeroValue,
		zeroUpdate: zeroUpdate,
	}
}

/*
n = 16 log2n = 4

0 |  1  1  1  1  1  1  1  1  1  1  1  1  1  1  1  1
1 |  2  2  2  2  2  2  2  2  3  3  3  3  3  3  3  3
2 |  4  4  4  4  5  5  5  5  6  6  6  6  7  7  7  7
3 |  8  8  9  9 10 10 11 11 12 12 13 13 14 14 15 15
4 | 16 17 18 19 20 21 22 23 24 25 26 27 28 29 30 31

	[0 ... val(i) ... size-1] [size ... zero ... n]
*/

func (st SegmentTree[value, update]) SetValue(
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

func (st SegmentTree[value, update]) Update(
	l, r int,
	upd update,
) {
	l = max(l, 0)
	r = min(r, st.n-1)
	if l > r {
		return
	}

	st.pushUpdates(l)
	st.pushUpdates(r)
	{
		l, r := l+st.n, r+st.n
		for l <= r {
			if l%2 == 1 {
				(upd).Push(&(*Get(st.arr, l)).update)
				l = l/2 + 1
			} else {
				l = l / 2
			}
			if r%2 == 0 {
				(upd).Push(&(*Get(st.arr, r)).update)
				r = r/2 - 1
			} else {
				r = r / 2
			}
		}
	}
	st.rebuild(l)
	st.rebuild(r)
}

func (st SegmentTree[value, update]) Get(
	l, r int,
) value {
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
			// apply update on-the-fly without copying node back
			(Get(st.arr, l).update).ApplyUpdate(&Get(st.arr, l).value)
			summedL = summedL.Merge((*Get(st.arr, l)).value)
			l = l/2 + 1
		} else {
			l = l / 2
		}
		if r%2 == 0 {
			(Get(st.arr, r).update).ApplyUpdate(&Get(st.arr, r).value)
			summedR = Get(st.arr, r).value.Merge(summedR)
			r = r/2 - 1
		} else {
			r = r / 2
		}
	}
	return summedL.Merge(summedR)
}

func (st SegmentTree[value, update]) LongestRangeWherePredicate(
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

func (st SegmentTree[value, update]) pushUpdates(
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

func (st SegmentTree[value, update]) rebuild(
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

//package utils
//file ..//utils/go

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

func IsPowerOf2[T int | uint64](x T) bool {
	x64 := uint64(x)
	return x64 != 0 && (x64&(x64-1)) == 0
}

func CastBool(b bool) int {
	if b {
		return 1
	} else {
		return 0
	}
}

func MapArray[X, Y any](arr []X, f func(X) Y) []Y {
	result := make([]Y, len(arr))
	for i, v := range arr {
		result[i] = f(v)
	}
	return result
}

func Transpose(X *[][]int) {
	M := *X
	n := len(M)
	m := len(M[0])
	M2 := make([][]int, m)
	for i := range m {
		M2[i] = make([]int, n)
		for j := range n {
			M2[i][j] = M[j][i]
		}
	}
	*X = M2
}

func TransposeSquare(X [][]int) {
	n := len(X)
	for i := 0; i < n; i++ {
		for j := i + 1; j < n; j++ {
			X[i][j], X[j][i] = X[j][i], X[i][j]
		}
	}
}

func CollectMap[X comparable](m map[X]struct{}) []X {
	res := make([]X, 0, len(m))
	for k := range m {
		res = append(res, k)
	}
	return res
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

func ReverseBits32(x uint32) uint32 {
	x = (x>>1)&0x55555555 | (x&0x55555555)<<1
	x = (x>>2)&0x33333333 | (x&0x33333333)<<2
	x = (x>>4)&0x0F0F0F0F | (x&0x0F0F0F0F)<<4
	x = (x>>8)&0x00FF00FF | (x&0x00FF00FF)<<8
	x = (x>>16)&0x0000FFFF | (x&0x0000FFFF)<<16
	return x
}

func Get[T any](slice []T, index int) *T {
	return (*T)(unsafe.Pointer(uintptr(unsafe.Pointer(unsafe.SliceData(slice))) + uintptr(index)*unsafe.Sizeof(*new(T))))
}

func GetArr[T any](slice []T, offset int) *[8]T {
	data := unsafe.Add(unsafe.Pointer(unsafe.SliceData(slice)), uintptr(offset)*unsafe.Sizeof(*new(T)))
	return (*[8]T)(data)
}

type Pair[X any, Y any] struct {
	X X
	Y Y
}

func Abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

