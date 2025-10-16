package hash_map

import (
	"main/utils"
	"math/rand/v2"
	"slices"
	"unsafe"
)

type entry[K comparable, V any] struct {
	hash  uint64
	key   K
	value V
}

type Hasher interface {
	Hash() uint64
}

type HashMap[K comparable, H Hasher, V any] struct {
	buckets [][]entry[K, V]
	logSize int
	size    int
	oddSalt uint64
}

func NewHashMap[K comparable, H Hasher, V any](
	size int,
) *HashMap[K, H, V] {
	logSize := utils.Log2Ceil(size + 1)
	return &HashMap[K, H, V]{
		buckets: make([][]entry[K, V], 1<<logSize),
		logSize: logSize,
		size:    0,
		oddSalt: rand.Uint64() | 1,
	}
}

func (hm *HashMap[K, H, V]) Hash(key K) uint64 {
	hash := (*(*H)(unsafe.Pointer(&key))).Hash()
	hash *= hm.oddSalt
	hash = utils.ReverseBits64(hash)
	return hash
}

func (hm *HashMap[K, H, V]) Get(key K) V {
	hash := hm.Hash(key)
	for _, e := range hm.buckets[hash%(1<<hm.logSize)] {
		if e.hash == hash && e.key == key {
			return e.value
		}
	}
	var zero V
	return zero
}

func (hm *HashMap[K, H, V]) Get2(key K) (V, bool) {
	hash := hm.Hash(key)
	for _, e := range hm.buckets[hash%(1<<hm.logSize)] {
		if e.hash == hash && e.key == key {
			return e.value, true
		}
	}
	var zero V
	return zero, false
}

func (hm *HashMap[K, H, V]) Delete(key K) bool {
	hash := hm.Hash(key)
	index := hash % (1 << hm.logSize)
	buckets := hm.buckets[index]
	for i := range buckets {
		e := &buckets[i]
		if e.hash == hash && e.key == key {
			buckets[i] = buckets[len(buckets)-1]
			hm.buckets[index] = buckets[:len(buckets)-1]
			hm.size--
			return true
		}
	}
	return false
}

func (hm *HashMap[K, H, V]) Set(key K, value V) {
	hash := hm.Hash(key)
	index := hash % (1 << hm.logSize)
	buckets := hm.buckets[index]
	for i := range buckets {
		e := &buckets[i]
		if e.hash == hash && e.key == key {
			e.value = value
			return
		}
	}
	hm.buckets[index] = append(buckets, entry[K, V]{
		hash:  hash,
		key:   key,
		value: value,
	})
	hm.size++
	if hm.size*4 > (1<<hm.logSize)*3 { // Resize at load factor 0.75
		hm.resize()
	}
}

func (hm *HashMap[K, H, V]) resize() {
	hm.buckets = append(hm.buckets, make([][]entry[K, V], 1<<hm.logSize)...)
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
