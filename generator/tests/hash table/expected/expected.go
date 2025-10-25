package main

import (
	"fmt"
	"math/bits"
	"math/rand/v2"
	"unsafe"
)

const maxOffset = 8
const checkHashFirst = true

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

func LogCeil(x uint64) int {
	if x == 0 {
		panic("Log2(0) is undefined")
	}
	if x == 1 {
		return 0
	}
	return 1 + LogFloor(x-1)
}

func LogFloor(x uint64) int {
	if x == 0 {
		panic("Log2(0) is undefined")
	}
	return 63 - bits.LeadingZeros64(x)
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

type intHash int

func (x intHash) Hash() uint64 {
	return uint64(x)
}

type intHasher int

func (x intHasher) Hash() uint64 {
	return uint64(x)
}

func main() {
	X := NewHashTableG1intG2intHashG3int(0)
	fmt.Println(X)
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

func Get[T any](slice []T, index int) *T {
	return (*T)(unsafe.Pointer(uintptr(unsafe.Pointer(unsafe.SliceData(slice))) + uintptr(index)*unsafe.Sizeof(*new(T))))
}

func GetArr[T any](slice []T, offset int) *[8]T {
	data := unsafe.Add(unsafe.Pointer(unsafe.SliceData(slice)), uintptr(offset)*unsafe.Sizeof(*new(T)))
	return (*[8]T)(data)
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

func ReverseBits64(x uint64) uint64 {
	x = (x>>1)&0x5555555555555555 | (x&0x5555555555555555)<<1
	x = (x>>2)&0x3333333333333333 | (x&0x3333333333333333)<<2
	x = (x>>4)&0x0F0F0F0F0F0F0F0F | (x&0x0F0F0F0F0F0F0F0F)<<4
	x = (x>>8)&0x00FF00FF00FF00FF | (x&0x00FF00FF00FF00FF)<<8
	x = (x>>16)&0x0000FFFF0000FFFF | (x&0x0000FFFF0000FFFF)<<16
	x = (x>>32)&0x00000000FFFFFFFF | (x&0x00000000FFFFFFFF)<<32
	return x
}
func NewHashTableG1intG2intHashG3int(size int) *HashTableG1intG2intHashG3int {
	log2Size := LogCeil(uint64(
		size*
			2 + 1))
	return &HashTableG1intG2intHashG3int{entries1: make([]hashTableEntryG1intG2int, (1<<
		log2Size)+maxOffset), entries2: make([][]hashTableEntryG1intG2int, 1<<log2Size), log2Size: log2Size, oddSalt1: rand.Uint64() | 1, oddSalt2: rand.Uint64() | 1, count: 0}
}

type HashTableG1intG2intHashG3int struct {
	entries1 []hashTableEntryG1intG2int
	entries2 [][]hashTableEntryG1intG2int
	log2Size int
	oddSalt1 uint64
	oddSalt2 uint64
	count    int
}

func (hm *HashTableG1intG2intHashG3int,

) Set(key int,

	value int,

) {
	if hm.
		count*
		2 >= (1 << hm.log2Size) {
		hm.resize()
	}
	hash := hm.hash(key)
	index1 := hm.index1(hash)
	arr := GetArrG1hashTableEntryOfintCint(hm.
		entries1,
		int(index1))
	for i := range maxOffset {
		e := &arr[i]
		if e.hash == 0 {
			*e = hashTableEntryG1intG2int{hash: hash, key: key, value: value}
			hm.count++
			return
		} else if (!checkHashFirst ||
			e.hash ==
				hash) && e.key ==
			key {
			e.value = value
			return
		}
	}
	index2 := hm.index2(hash)
	entries := GetG1SlicehashTableEntryOfintCint(hm.entries2,

		int(index2))
	for i, val := range *entries {
		if val.
			hash == hash && val.key ==
			key {
			GetG1hashTableEntryOfintCint(
				*entries, i).value = value
			return

		}
	}
	*entries = append(*entries, hashTableEntryG1intG2int{hash: hash, key: key, value: value})
	hm.
		count++
}
func (hm *HashTableG1intG2intHashG3int,

) hash(key int,

) uint64 {
	val :=
		(*(*intHash)(unsafe.Pointer(
			&key))).Hash()
	if val == 0 {
		return 1
	} else {
		return val
	}
}
func (hm *HashTableG1intG2intHashG3int,

) index1(hash uint64) uint64 {

	hash *= hm.oddSalt1
	hash = ReverseBits64(hash)
	return hash & (1<<hm.log2Size - 1)
}
func (hm *HashTableG1intG2intHashG3int,

) index2(hash uint64) uint64 {

	hash *= hm.oddSalt2
	hash = ReverseBits64(hash)
	return hash & (1<<hm.log2Size - 1)
}
func (hm *HashTableG1intG2intHashG3int,

) resize() {
	hm.log2Size++
	newEntries1 := make([]hashTableEntryG1intG2int, (1<<hm.log2Size)+maxOffset)
	newEntries2 := make([][]hashTableEntryG1intG2int,
		1<<hm.log2Size,
	)
	add := func(e hashTableEntryG1intG2int) {
		hash := hm.hash(e.key)
		index1 := hm.index1(hash)
		arr :=
			GetArrG1hashTableEntryOfintCint(newEntries1, int(index1))
		for j := range arr {
			if arr[j].hash == 0 {
				arr[j] = e
				return
			}
		}
		index2 := hm.
			index2(hash)
		entries := GetG1SlicehashTableEntryOfintCint(
			newEntries2,
			int(index2))
		*entries = append(*entries,
			e)
	}
	for _, e := range hm.entries1 {
		if e.
			hash ==
			0 {
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
func GetG1SlicehashTableEntryOfintCint(slice [][]hashTableEntryG1intG2int,
	index int) *[]hashTableEntryG1intG2int {
	return (*[]hashTableEntryG1intG2int)(unsafe.
		Pointer(uintptr(unsafe.
			Pointer(unsafe.SliceData(slice))) + uintptr(index)*unsafe.Sizeof(*new([]hashTableEntryG1intG2int))),
	)
}
func GetG1hashTableEntryOfintCint(slice []hashTableEntryG1intG2int,
	index int) *hashTableEntryG1intG2int {
	return (*hashTableEntryG1intG2int)(unsafe.
		Pointer(uintptr(unsafe.
			Pointer(unsafe.SliceData(slice))) + uintptr(index)*unsafe.Sizeof(*new(hashTableEntryG1intG2int))),
	)
}
func GetArrG1hashTableEntryOfintCint(
	slice []hashTableEntryG1intG2int,
	offset int) *[8]hashTableEntryG1intG2int {
	data :=
		unsafe.
			Add(unsafe.Pointer(unsafe.SliceData(slice)),

				uintptr(offset)*unsafe.Sizeof(*new(hashTableEntryG1intG2int)))
	return (*[8]hashTableEntryG1intG2int)(data)
}

type hashTableEntryG1intG2int struct {
	hash uint64

	key int

	value int
}
