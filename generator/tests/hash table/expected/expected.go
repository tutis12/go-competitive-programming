package main

import (
	"fmt"
	"math/bits"
	"math/rand/v2"
)

const maxOffset = 8

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

func main() {
	X := NewHashTableG1intG2intHashG3int(0)
	fmt.Println(X)
}
func NewHashTableG1intG2intHashG3int(size int) *HashTableG1intG2intHashG3int {
	log2Size := LogCeil(uint64(size*2 + 1))
	return &HashTableG1intG2intHashG3int{entries1: make([]hashTableEntryG1intG2int, (1<<log2Size)+maxOffset), entries2: make([][]hashTableEntryG1intG2int, 1<<log2Size), log2Size: log2Size,
		oddSalt1: rand.Uint64() | 1, oddSalt2: rand.Uint64() | 1, count: 0}
}

type HashTableG1intG2intHashG3int struct {
	entries1 []hashTableEntryG1intG2int

	entries2 [][]hashTableEntryG1intG2int

	log2Size int
	oddSalt1 uint64
	oddSalt2 uint64
	count    int
}
type hashTableEntryG1intG2int struct {
	hash uint64
	key  int

	value int
}
