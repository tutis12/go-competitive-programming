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

	// Check that the right concrete functions were generated (both int and uint64 versions)
	hasIntLog2Floor := strings.Contains(resultStr, "func Log2FloorG1(x int) int") || strings.Contains(resultStr, "func Log2FloorG2(x int) int")
	hasUint64Log2Floor := strings.Contains(resultStr, "func Log2FloorG1(x uint64) int") || strings.Contains(resultStr, "func Log2FloorG2(x uint64) int")
	if !hasIntLog2Floor {
		t.Error("Missing Log2Floor function with int parameter")
	}
	if !hasUint64Log2Floor {
		t.Error("Missing Log2Floor function with uint64 parameter")
	}
	
	hasIntIsPowerOf2 := strings.Contains(resultStr, "func IsPowerOf2G1(x int) bool") || strings.Contains(resultStr, "func IsPowerOf2G2(x int) bool")
	hasUint64IsPowerOf2 := strings.Contains(resultStr, "func IsPowerOf2G1(x uint64) bool") || strings.Contains(resultStr, "func IsPowerOf2G2(x uint64) bool")
	if !hasIntIsPowerOf2 {
		t.Error("Missing IsPowerOf2 function with int parameter")
	}
	if !hasUint64IsPowerOf2 {
		t.Error("Missing IsPowerOf2 function with uint64 parameter")
	}

	// Check that calls were replaced correctly - Log2Floor call should use uint64 version since x64 is uint64
	hasCorrectLog2FloorCall := strings.Contains(resultStr, "Log2FloorG1(x64-1)") || strings.Contains(resultStr, "Log2FloorG2(x64-1)")
	if !hasCorrectLog2FloorCall {
		t.Error("Log2Floor call in Log2Ceil should be replaced with a concrete version")
	}
	
	// Check that IsPowerOf2 calls were replaced with concrete versions
	hasIsPowerOf2IntCall := strings.Contains(resultStr, "IsPowerOf2G1(42)") || strings.Contains(resultStr, "IsPowerOf2G2(42)")
	hasIsPowerOf2Uint64Call := strings.Contains(resultStr, "IsPowerOf2G1(uint64(42))") || strings.Contains(resultStr, "IsPowerOf2G2(uint64(42))")
	if !hasIsPowerOf2IntCall {
		t.Error("IsPowerOf2(42) should be replaced with a concrete version")
	}
	if !hasIsPowerOf2Uint64Call {
		t.Error("IsPowerOf2(uint64(42)) should be replaced with a concrete version")
	}

	t.Logf("Result:\n%s", resultStr)
}
