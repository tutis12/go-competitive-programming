package main

import (
	"fmt"
	"math"
	"math/bits"
	"math/rand/v2"
	"os"
	"runtime"
	"slices"
	"strconv"
	"testing"
	"time"
	"unsafe"
)

//package main
//file ..//go

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

//package main
//file ..//solve_a.go

// var solveX = solveA

func solveA(
	stdin *Reader,
	stdout *Writer,
) {
	t := stdin.Int()
	for range t {
		solveTestA(stdin, stdout)
	}
}

func solveTestA(
	stdin *Reader,
	stdout *Writer,
) {

}

/*input
1
2
*/

/*output

 */
//package main
//file ..//solve_b.go

//var solveX = solveB

func solveB(
	stdin *Reader,
	stdout *Writer,
) {
	t := stdin.Int()
	for range t {
		solveTestB(stdin, stdout)
	}
}

func solveTestB(
	stdin *Reader,
	stdout *Writer,
) {

}

/*input
3
3 5
10101
10100
00101
4 6
011101
010001
100010
101110
5 5
11100
10110
11111
01101
00111
*/

/*output
6 6 6 9 9
6 6 6 9 9
0 0 9 9 9
0 10 8 8 10 10
0 10 8 8 10 10
10 10 8 8 10 0
10 10 8 8 10 0
6 6 6 0 0
6 6 4 4 0
6 4 4 4 6
0 4 4 6 6
0 0 6 6 6

*/
//package main
//file ..//solve_c.go

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

type intHash int

func (h intHash) Hash() uint64 {
	return uint64(h)
}

func solveTestC(
	stdin *Reader,
	stdout *Writer,
) {
	n := stdin.Int()
	a := NewHashMapG1(0)
	for i := range n {
		a.Set(i, stdin.Int())
	}
	st := NewSTG1(
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

	dp := NewHashMapG1(0)

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
//package main
//file ..//solve_d.go

const (
	mod = 1000000007
)

//var solveX = solveD

func solveD(
	stdin *Reader,
	stdout *Writer,
) {
	t := stdin.Int()
	for range t {
		solveTestD(stdin, stdout)
	}
}

func solveTestD(
	stdin *Reader,
	stdout *Writer,
) {
	n := stdin.Int()
	a := stdin.Ints(n+1, 0)
	answer := count(n, a)
	a[n] = 0
	answer -= count(n, a)
	answer += mod
	answer %= mod
	stdout.Int(answer, '\n')
}

func count(n int, a []int) int {
	for i := range n + 1 {
		if a[i] < -1 || a[i] > n {
			return 0
		}
	}
	cnt := make([]int, n+1)
	for i := range n + 1 {
		if a[i] != -1 {
			cnt[a[i]]++
		}
	}
	for i := 1; i <= n; i++ {
		if cnt[i] >= 2 {
			return 0
		}
	}
	for i := 1; i <= n; i++ {
		if cnt[i] != 0 && a[i] == 0 {
			return 0
		}
	}
	c := 0
	for i := 1; i <= n; i++ {
		if cnt[i] == 0 {
			c++
		}
	}
	ret := 1
	for i := 1; i <= c+1; i++ {
		ret *= i
		ret %= mod
	}
	return ret
}

/*input
6
1
-1 -1
2
-1 2 -1
2
-1 -1 -1
3
-1 -1 3 -1
3
-1 2 3 -1
5
-1 -1 -1 1 0 -1

*/

/*output
1
1
3
2
0
3

*/
//package main
//file ..//solve_e.go

// var solveX = solveE

func solveE(
	stdin *Reader,
	stdout *Writer,
) {
	t := stdin.Int()
	for range t {
		solveTestE(stdin, stdout)
	}
}

func solveTestE(
	stdin *Reader,
	stdout *Writer,
) {

}

/*input

 */

/*output

 */
//package main
//file ..//solve_f.go

// var solveX = solveF

func solveF(
	stdin *Reader,
	stdout *Writer,
) {
	t := stdin.Int()
	for range t {
		solveTestF(stdin, stdout)
	}
}

func solveTestF(
	stdin *Reader,
	stdout *Writer,
) {

}

/*input
1
3 15
4 2 5
1 9 3
7 6 8

4

1

9

6

8

4

4

7

7

8

5

4

9

9

9


*/

/*output





? 1 1

? 1 2

? 1 3

? 1 4

? 1 5

? 2 1

? 2 2

? 2 3

? 2 4

? 2 5

? 3 1

? 3 2

? 3 3

? 3 4

? 3 5

! 1 4 4 4 4 5 6 7 7 8 8 9 9 9 9

*/
//package main
//file ..//solve_g.go

// var solveX = solveG

func solveG(
	stdin *Reader,
	stdout *Writer,
) {
	t := stdin.Int()
	for range t {
		solveTestG(stdin, stdout)
	}
}

func solveTestG(
	stdin *Reader,
	stdout *Writer,
) {

}

/*input

 */

/*output

 */
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

func (w *Writer) Fprintf(format string, a ...any) {
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

const hashMapCheckHashes = false



type Hasher interface {
	Hash() uint64
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
	hashMap := NewHashMapG2(b.N)
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

//package segment_tree_iter
//file ..//segment_tree_iter/go







/*
n = 16 log2n = 4

0 |  1  1  1  1  1  1  1  1  1  1  1  1  1  1  1  1
1 |  2  2  2  2  2  2  2  2  3  3  3  3  3  3  3  3
2 |  4  4  4  4  5  5  5  5  6  6  6  6  7  7  7  7
3 |  8  8  9  9 10 10 11 11 12 12 13 13 14 14 15 15
4 | 16 17 18 19 20 21 22 23 24 25 26 27 28 29 30 31

	[0 ... val(i) ... size-1] [size ... zero ... n]
*/













//package utils
//file ..//utils/go







func CastBool(b bool) int {
	if b {
		return 1
	} else {
		return 0
	}
}



func Transpose(X *[][]int) {
	M := *X
	n := len(M)
	m := len(M[0])
	M2 := make([][]int, m)
	for i := 0; i < m; i++ {
		M2[i] = make([]int, n)
		for j := 0; j < n; j++ {
			M2[i][j] = M[j][i]
		}
	}
	*X = M2
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

// ---- Concrete Types (Generated) ----
type HashMapG1 struct { buckets [][]entryG1
logSize int
size int
oddSalt uint64
 }
type HashMapG2 struct { buckets [][]entryG1
logSize int
size int
oddSalt uint64
 }
type STG1 struct { log2n int
n int
arr []nodeG1
zeroValue stValue
zeroUpdate lazy
 }
type entryG1 struct { hash uint64
key int
value int
 }
type nodeG1 struct { value stValue
update lazy
 }
// ---- Concrete Methods (Generated) ----
func (hm *HashMapG1) Hash(key int) uint64 {
	hash := (*(*intHash)(unsafe.Pointer(&key))).Hash()
	hash *= hm.oddSalt
	hash = ReverseBits64(hash)
	return hash
}
func (hm *HashMapG2) Hash(key int) uint64 {
	hash := (*(*intHasher)(unsafe.Pointer(&key))).Hash()
	hash *= hm.oddSalt
	hash = ReverseBits64(hash)
	return hash
}
func (hm *HashMapG1) Get(key int) int {
	hash := hm.Hash(key)
	for _, e := range hm.buckets[hash%(1<<hm.logSize)] {
		if (!hashMapCheckHashes || e.hash == hash) && e.key == key {
			return e.value
		}
	}
	var zero int
	return zero
}
func (hm *HashMapG2) Get(key int) int {
	hash := hm.Hash(key)
	for _, e := range hm.buckets[hash%(1<<hm.logSize)] {
		if (!hashMapCheckHashes || e.hash == hash) && e.key == key {
			return e.value
		}
	}
	var zero int
	return zero
}
func (hm *HashMapG1) Get2(key int) (int, bool) {
	hash := hm.Hash(key)
	for _, e := range hm.buckets[hash%(1<<hm.logSize)] {
		if (!hashMapCheckHashes || e.hash == hash) && e.key == key {
			return e.value, true
		}
	}
	var zero int
	return zero, false
}
func (hm *HashMapG2) Get2(key int) (int, bool) {
	hash := hm.Hash(key)
	for _, e := range hm.buckets[hash%(1<<hm.logSize)] {
		if (!hashMapCheckHashes || e.hash == hash) && e.key == key {
			return e.value, true
		}
	}
	var zero int
	return zero, false
}
func (hm *HashMapG1) Delete(key int) bool {
	hash := hm.Hash(key)
	index := hash % (1 << hm.logSize)
	buckets := hm.buckets[index]
	for i := range buckets {
		e := &buckets[i]
		if (!hashMapCheckHashes || e.hash == hash) && e.key == key {
			buckets[i] = buckets[len(buckets)-1]
			hm.buckets[index] = buckets[:len(buckets)-1]
			hm.size--
			return true
		}
	}
	return false
}
func (hm *HashMapG2) Delete(key int) bool {
	hash := hm.Hash(key)
	index := hash % (1 << hm.logSize)
	buckets := hm.buckets[index]
	for i := range buckets {
		e := &buckets[i]
		if (!hashMapCheckHashes || e.hash == hash) && e.key == key {
			buckets[i] = buckets[len(buckets)-1]
			hm.buckets[index] = buckets[:len(buckets)-1]
			hm.size--
			return true
		}
	}
	return false
}
func (hm *HashMapG1) Set(key int, value int) {
	hash := hm.Hash(key)
	index := hash % (1 << hm.logSize)
	buckets := hm.buckets[index]
	for i := range buckets {
		e := &buckets[i]
		if (!hashMapCheckHashes || e.hash == hash) && e.key == key {
			e.value = value
			return
		}
	}
	hm.buckets[index] = append(buckets, entryG1{
		hash:	hash,
		key:	key,
		value:	value,
	})
	hm.size++
	if hm.size*2 > 1<<hm.logSize {
		hm.resize()
	}
}
func (hm *HashMapG2) Set(key int, value int) {
	hash := hm.Hash(key)
	index := hash % (1 << hm.logSize)
	buckets := hm.buckets[index]
	for i := range buckets {
		e := &buckets[i]
		if (!hashMapCheckHashes || e.hash == hash) && e.key == key {
			e.value = value
			return
		}
	}
	hm.buckets[index] = append(buckets, entryG1{
		hash:	hash,
		key:	key,
		value:	value,
	})
	hm.size++
	if hm.size*2 > 1<<hm.logSize {
		hm.resize()
	}
}
func (hm *HashMapG1) resize() {
	hm.buckets = append(hm.buckets, make([][]entryG1, 1<<hm.logSize)...)
	for i, bucket := range hm.buckets[:1<<hm.logSize] {
		cntMove := 0
		for i := 0; i < len(bucket)-cntMove; {
			e := bucket[i]
			if e.hash&(1<<hm.logSize) != 0 {
				j := len(bucket) - 1 - cntMove
				bucket[i], bucket[j] = bucket[j], bucket[i]
				cntMove++
			} else {
				i++
			}
		}
		odds := bucket[len(bucket)-cntMove:]
		evens := bucket[:len(bucket)-cntMove]
		if len(odds) <= len(evens) {
			odds = slices.Clone(odds)
		} else {
			evens = slices.Clone(evens)
		}
		hm.buckets[i+(1<<hm.logSize)] = odds
		hm.buckets[i] = evens
	}
	hm.logSize++
}
func (hm *HashMapG2) resize() {
	hm.buckets = append(hm.buckets, make([][]entryG1, 1<<hm.logSize)...)
	for i, bucket := range hm.buckets[:1<<hm.logSize] {
		cntMove := 0
		for i := 0; i < len(bucket)-cntMove; {
			e := bucket[i]
			if e.hash&(1<<hm.logSize) != 0 {
				j := len(bucket) - 1 - cntMove
				bucket[i], bucket[j] = bucket[j], bucket[i]
				cntMove++
			} else {
				i++
			}
		}
		odds := bucket[len(bucket)-cntMove:]
		evens := bucket[:len(bucket)-cntMove]
		if len(odds) <= len(evens) {
			odds = slices.Clone(odds)
		} else {
			evens = slices.Clone(evens)
		}
		hm.buckets[i+(1<<hm.logSize)] = odds
		hm.buckets[i] = evens
	}
	hm.logSize++
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
func (st STG1) Update(
	l, r int,
	upd lazy,
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
				(upd).Push(&st.arr[l].update)
				l = l/2 + 1
			} else {
				l = l / 2
			}
			if r%2 == 0 {
				(upd).Push(&st.arr[r].update)
				r = r/2 - 1
			} else {
				r = r / 2
			}
		}
	}
	st.rebuild(l)
	st.rebuild(r)
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
// ---- Concrete Generic Functions (Generated) ----
func Log2FloorG1(x uint64) int {
	x64 := uint64(x)
	if x64 == 0 {
		panic("Log2(0) is undefined")
	}
	return 63 - bits.LeadingZeros64(x64)
}
func Log2FloorG2(x int) int {
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
func NewHashMapG1(
	size int,
) *HashMapG1 {
	logSize := Log2CeilG1(size*2 + 1)
	return &HashMapG1{
		buckets:	make([][]entryG1, 1<<logSize),
		logSize:	logSize,
		size:		0,
		oddSalt:	rand.Uint64() | 1,
	}
}
func NewHashMapG2(
	size int,
) *HashMapG2 {
	logSize := Log2CeilG1(size*2 + 1)
	return &HashMapG2{
		buckets:	make([][]entryG1, 1<<logSize),
		logSize:	logSize,
		size:		0,
		oddSalt:	rand.Uint64() | 1,
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
		log2n:		log2n,
		n:		n,
		arr:		arr,
		zeroValue:	zeroValue,
		zeroUpdate:	zeroUpdate,
	}
}

