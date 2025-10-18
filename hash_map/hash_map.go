package hash_map

import (
	"main/utils"
	"math/rand/v2"
	"unsafe"
)

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
	log2Size := utils.LogCeil(uint64(size*2 + 1))
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
	hash = utils.ReverseBits64(hash)
	return hash & (1<<hm.log2Size - 1)
}

func (hm *HashTable[K, H, V]) index2(hash uint64) uint64 {
	hash *= hm.oddSalt2
	hash = utils.ReverseBits64(hash)
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
	arr := utils.GetArr(hm.entries1, int(index1))
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
	for _, val := range *utils.Get(hm.entries2, int(index2)) {
		if (!checkHashFirst || val.hash == hash) && val.key == key {
			return val.value, true
		}
	}
	return zero, false
}

func (hm *HashTable[K, H, V]) Delete(key K) bool {
	hash := hm.hash(key)
	index1 := hm.index1(hash)
	arr := utils.GetArr(hm.entries1, int(index1))
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
	slice := *utils.Get(hm.entries2, int(index2))
	for i := range slice {
		val := &slice[i]
		if (!checkHashFirst || val.hash == hash) && val.key == key {
			*val = *utils.Get(slice, len(slice)-1)
			slice = slice[:len(slice)-1]
			*utils.Get(hm.entries2, int(index2)) = slice
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
	arr := utils.GetArr(hm.entries1, int(index1))
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
	entries := utils.Get(hm.entries2, int(index2))
	for i, val := range *entries {
		if val.hash == hash && val.key == key {
			utils.Get(*entries, i).value = value
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
		arr := utils.GetArr(newEntries1, int(index1))
		for j := range arr {
			if arr[j].hash == 0 {
				arr[j] = e
				return
			}
		}
		index2 := hm.index2(hash)
		entries := utils.Get(newEntries2, int(index2))
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
