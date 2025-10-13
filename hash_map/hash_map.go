package hash_map

import (
	"slices"
)

type entry[K comparable, V any] struct {
	hash  uint64
	key   *K
	value *V
}

type HashMap[K comparable, V any] struct {
	buckets [][]entry[K, V]
	logSize int
	size    int
	hasher  func(K) uint64
}

func NewHashMap[K comparable, V any](
	hasher func(K) uint64,
) *HashMap[K, V] {
	const logSize = 3
	return &HashMap[K, V]{
		buckets: make([][]entry[K, V], 1<<logSize),
		logSize: logSize,
		size:    0,
		hasher:  hasher,
	}
}

func (hm *HashMap[K, V]) Get(key K) (V, bool) {
	hash := hm.hasher(key)
	bucket := hm.buckets[hash%(1<<hm.logSize)]
	for _, e := range bucket {
		if e.hash == hash && *e.key == key {
			return *e.value, true
		}
	}
	var zero V
	return zero, false
}

func (hm *HashMap[K, V]) Delete(key K) bool {
	hash := hm.hasher(key)
	index := hash % (1 << hm.logSize)
	bucket := hm.buckets[index]
	for i := range bucket {
		e := &bucket[i]
		if e.hash == hash && *e.key == key {
			bucket[i] = bucket[len(bucket)-1]
			hm.buckets[index] = bucket[:len(bucket)-1]
			hm.size--
			return true
		}
	}
	return false
}

func (hm *HashMap[K, V]) Set(key K, value V) {
	hash := hm.hasher(key)
	index := hash % (1 << hm.logSize)
	bucket := hm.buckets[index]
	for i := range bucket {
		e := &bucket[i]
		if e.hash == hash && *e.key == key {
			e.value = &value
			return
		}
	}
	hm.buckets[index] = append(bucket, entry[K, V]{
		hash:  hash,
		key:   &key,
		value: &value,
	})
	hm.size++
	if hm.size > (1 << hm.logSize) {
		hm.resize()
	}
}

func (hm *HashMap[K, V]) resize() {
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
		hm.buckets[i+(1<<hm.logSize)] = slices.Clone(bucket[len(bucket)-cntMove:])
		hm.buckets[i] = bucket[:len(bucket)-cntMove]
	}
	hm.logSize++
}
