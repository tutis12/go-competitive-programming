package main

import (
	"strings"
	"testing"
)

func TestCallReplacement(t *testing.T) {
	code := `package main

type intHash int

type Hasher[T any] interface {
	Hash(T) uint64
}

type HashMap[K comparable, V any, H Hasher[K]] struct{}

func Log2Floor[T int | uint64](x T) int {
	return 0
}

func Log2Ceil[T int | uint64](x T) int {
	x64 := uint64(x)
	return 1 + Log2Floor(x64-1) // This should become Log2FloorG2(x64-1) not Log2FloorG1(x64-1)
}

func IsPowerOf2[T int | uint64](x T) bool {
	return x > 0 && (x&(x-1)) == 0
}

func NewHashMap[K comparable, V any, H Hasher[K]](size int) *HashMap[K, V, H] {
	return nil
}

func testFunc() {
	// These should become the correct concrete versions based on argument types
	a := NewHashMap[intHash, int](10)          // should be NewHashMapG1
	b := IsPowerOf2(42)                        // should be IsPowerOf2G1 (int)
	c := IsPowerOf2(uint64(42))                // should be IsPowerOf2G2 (uint64)
	d := Log2Ceil(10)                          // should be Log2CeilG1 (int)
	e := Log2Ceil(uint64(10))                  // should be Log2CeilG2 (uint64)
}
`

	result := RemoveGenerics([]byte(code))
	resultStr := string(result)

	// Check that the right concrete functions were generated
	if !strings.Contains(resultStr, "func Log2FloorG1(x int) int") {
		t.Error("Missing Log2FloorG1 function")
	}
	if !strings.Contains(resultStr, "func Log2FloorG2(x uint64) int") {
		t.Error("Missing Log2FloorG2 function")
	}
	if !strings.Contains(resultStr, "func IsPowerOf2G1(x int) bool") {
		t.Error("Missing IsPowerOf2G1 function")
	}
	if !strings.Contains(resultStr, "func IsPowerOf2G2(x uint64) bool") {
		t.Error("Missing IsPowerOf2G2 function")
	}

	// Check that calls were replaced correctly
	if !strings.Contains(resultStr, "Log2FloorG2(x64-1)") {
		t.Error("Log2Floor call in Log2Ceil should become Log2FloorG2, not Log2FloorG1")
	}
	if !strings.Contains(resultStr, "IsPowerOf2G1(42)") {
		t.Error("IsPowerOf2(42) should become IsPowerOf2G1")
	}
	if !strings.Contains(resultStr, "IsPowerOf2G2(uint64(42))") {
		t.Error("IsPowerOf2(uint64(42)) should become IsPowerOf2G2")
	}

	t.Logf("Result:\n%s", resultStr)
}
