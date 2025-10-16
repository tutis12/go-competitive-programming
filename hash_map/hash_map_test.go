package hash_map

import (
	"math/rand/v2"
	"testing"
)

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
	hashMap := NewHashMap[int, intHasher, int](b.N)
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
