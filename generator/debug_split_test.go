package main

import (
	"go/format"
	"strings"
	"testing"
)

func TestDebugSpecificGenericFunctions(t *testing.T) {
	// This test focuses on the specific generic functions that are causing issues
	// by testing them individually to isolate the problems

	t.Run("UtilityGenericFunctions", func(t *testing.T) {
		testCase := `package main

import "math/bits"

func Log2Floor[T int | uint64](x T) int {
	x64 := uint64(x)
	if x64 == 0 {
		panic("Log2(0) is undefined")
	}
	return 63 - bits.LeadingZeros64(x64)
}

func Log2Ceil[T int | uint64](x T) int {
	x64 := uint64(x)
	if x64 == 0 {
		panic("Log2(0) is undefined")
	}
	if x64 == 1 {
		return 0
	}
	return 1 + Log2Floor(x64-1)
}

func IsPowerOf2[T int | uint64](x T) bool {
	x64 := uint64(x)
	return x64 != 0 && (x64&(x64-1)) == 0
}

func main() {
	size := 10
	log2n := Log2Ceil(size)        // should become Log2CeilG1(size)
	isPow := IsPowerOf2(8)         // should become IsPowerOf2G1(8)
	fmt.Println(log2n, isPow)
}`

		result := RemoveGenerics([]byte(testCase))
		resultStr := string(result)

		formatted, err := format.Source(result)
		if err != nil {
			t.Logf("Formatting error: %v", err)
		} else {
			resultStr = string(formatted)
		}

		// Check that calls are converted
		if strings.Contains(resultStr, "Log2Ceil(size)") {
			t.Errorf("❌ Log2Ceil(size) call not converted to Log2CeilG1(size)")
		}
		if strings.Contains(resultStr, "IsPowerOf2(8)") {
			t.Errorf("❌ IsPowerOf2(8) call not converted to IsPowerOf2G1(8)")
		}
		if strings.Contains(resultStr, "Log2Floor(x64-1)") {
			t.Errorf("❌ Log2Floor(x64-1) call not converted to Log2FloorG1(x64-1)")
		}

		// Check that concrete functions exist
		if !strings.Contains(resultStr, "func Log2CeilG1(") {
			t.Errorf("❌ Concrete Log2CeilG1 function not generated")
		}
		if !strings.Contains(resultStr, "func IsPowerOf2G1(") {
			t.Errorf("❌ Concrete IsPowerOf2G1 function not generated")
		}
		if !strings.Contains(resultStr, "func Log2FloorG1(") {
			t.Errorf("❌ Concrete Log2FloorG1 function not generated")
		}

		t.Logf("Generated utility functions code:\n%s", resultStr)
	})

	t.Run("HashMapGenericTypes", func(t *testing.T) {
		testCase := `package main

type Hasher[K any] interface {
	*K
	Hash() uint64
}

type entry[K comparable, V any] struct {
	hash  uint64
	key   K
	value V
}

type HashMap[K comparable, V any, H Hasher[K]] struct {
	buckets [][]entry[K, V]
	logSize int
	size    int
	oddSalt uint64
}

func NewHashMap[K comparable, V any, H Hasher[K]](size int) *HashMap[K, V, H] {
	return &HashMap[K, V, H]{
		buckets: make([][]entry[K, V], 1<<1),
		logSize: 1,
		size:    0,
		oddSalt: 1234567,
	}
}

type intHash int

func (h *intHash) Hash() uint64 {
	return uint64(*h)
}

func main() {
	hm := NewHashMap[intHash, int, *intHash](10)
	fmt.Println(hm)
}`

		result := RemoveGenerics([]byte(testCase))
		resultStr := string(result)

		formatted, err := format.Source(result)
		if err != nil {
			t.Logf("Formatting error: %v", err)
		} else {
			resultStr = string(formatted)
		}

		// Check that explicit generic calls are converted
		if strings.Contains(resultStr, "NewHashMap[intHash, int, *intHash]") {
			t.Errorf("❌ Explicit NewHashMap call not converted")
		}

		// Check that concrete types exist
		if !strings.Contains(resultStr, "HashMapG1") {
			t.Errorf("❌ Concrete HashMapG1 type not generated")
		}
		if !strings.Contains(resultStr, "entryG1") {
			t.Errorf("❌ Concrete entryG1 type not generated")
		}

		t.Logf("Generated HashMap code:\n%s", resultStr)
	})

	t.Run("SegmentTreeGenericTypes", func(t *testing.T) {
		testCase := `package main

type node[value, update any] struct {
	value  value
	update update
}

type ST[value, update any] struct {
	log2n      int
	n          int
	arr        []node[value, update]
	zeroValue  value
	zeroUpdate update
}

func NewST[V, U any](size int, zeroValue V, zeroUpdate U) *ST[V, U] {
	n := 1
	arr := make([]node[V, U], 2*n)
	return &ST[V, U]{
		log2n:      1,
		n:          n,
		arr:        arr,
		zeroValue:  zeroValue,
		zeroUpdate: zeroUpdate,
	}
}

func main() {
	st := NewST[int, bool](10, 0, false)
	fmt.Println(st)
}`

		result := RemoveGenerics([]byte(testCase))
		resultStr := string(result)

		formatted, err := format.Source(result)
		if err != nil {
			t.Logf("Formatting error: %v", err)
		} else {
			resultStr = string(formatted)
		}

		// Check conversions
		if strings.Contains(resultStr, "NewST[int, bool]") {
			t.Errorf("❌ Explicit NewST call not converted")
		}

		// Check concrete types
		if !strings.Contains(resultStr, "STG1") {
			t.Errorf("❌ Concrete STG1 type not generated")
		}
		if !strings.Contains(resultStr, "nodeG1") {
			t.Errorf("❌ Concrete nodeG1 type not generated")
		}

		t.Logf("Generated SegmentTree code:\n%s", resultStr)
	})

	t.Run("InferredVsExplicitCalls", func(t *testing.T) {
		testCase := `package main

func GenericFunc[T any](x T) T {
	return x
}

func main() {
	// Explicit call - should be converted
	result1 := GenericFunc[int](42)
	
	// Inferred call - should also be converted 
	result2 := GenericFunc(42)
	
	// Another inferred call with different type
	result3 := GenericFunc("hello")
	
	fmt.Println(result1, result2, result3)
}`

		result := RemoveGenerics([]byte(testCase))
		resultStr := string(result)

		formatted, err := format.Source(result)
		if err != nil {
			t.Logf("Formatting error: %v", err)
		} else {
			resultStr = string(formatted)
		}

		// Check that both explicit and inferred calls are converted
		if strings.Contains(resultStr, "GenericFunc[int]") {
			t.Errorf("❌ Explicit GenericFunc[int] call not converted")
		}
		if strings.Contains(resultStr, "GenericFunc(42)") && !strings.Contains(resultStr, "GenericFuncG1(42)") {
			t.Errorf("❌ Inferred GenericFunc(42) call not converted to GenericFuncG1")
		}
		if strings.Contains(resultStr, "GenericFunc(\"hello\")") && !strings.Contains(resultStr, "GenericFuncG2(\"hello\")") {
			t.Errorf("❌ Inferred GenericFunc(\"hello\") call not converted to GenericFuncG2")
		}

		t.Logf("Generated inferred vs explicit calls code:\n%s", resultStr)
	})

	t.Run("ComplexNestedGenerics", func(t *testing.T) {
		testCase := `package main

func Log2Ceil[T int | uint64](x T) int {
	return 0 // simplified
}

func NewHashMap[K comparable, V any](size int) map[K]V {
	return make(map[K]V)
}

type intHash int

func main() {
	n := 10
	
	// These are the exact patterns from the real failing code
	a := NewHashMap[intHash, int](n)    // Should become NewHashMapG1(n)
	dp := NewHashMap[intHash, int](n)   // Should become NewHashMapG1(n) 
	log2n := Log2Ceil(n)                // Should become Log2CeilG1(n)
	
	fmt.Println(a, dp, log2n)
}`

		result := RemoveGenerics([]byte(testCase))
		resultStr := string(result)

		formatted, err := format.Source(result)
		if err != nil {
			t.Logf("Formatting error: %v", err)
		} else {
			resultStr = string(formatted)
		}

		// These are the critical checks that were failing in the real code
		if strings.Contains(resultStr, "NewHashMap[intHash, int]") {
			t.Errorf("❌ CRITICAL: NewHashMap[intHash, int] not converted - this causes undefined errors!")
		}
		if strings.Contains(resultStr, "Log2Ceil(n)") && !strings.Contains(resultStr, "Log2CeilG1(n)") {
			t.Errorf("❌ CRITICAL: Log2Ceil(n) not converted - this causes undefined errors!")
		}

		// Check that we have the expected conversions
		if !strings.Contains(resultStr, "NewHashMapG1(n)") {
			t.Errorf("❌ Expected NewHashMapG1(n) conversion not found")
		}
		if !strings.Contains(resultStr, "Log2CeilG1(n)") {
			t.Errorf("❌ Expected Log2CeilG1(n) conversion not found")
		}

		t.Logf("Generated complex nested generics code:\n%s", resultStr)
	})
}

func TestDebugMethodCallSyntax(t *testing.T) {
	// Test the method call syntax issues we saw in the generated code
	testCase := `package main

type lazy struct {
	addDP int
}

func (up lazy) ApplyUpdate(val *int) {
	*val += up.addDP
}

func (top lazy) Push(existing *lazy) {
	existing.addDP += top.addDP
}

func main() {
	update := lazy{addDP: 5}
	val := 10
	other := lazy{}
	
	// These should work - method calls on variables
	update.ApplyUpdate(&val)
	update.Push(&other)
	
	fmt.Println(val, other)
}`

	result := RemoveGenerics([]byte(testCase))
	resultStr := string(result)

	formatted, err := format.Source(result)
	if err != nil {
		t.Logf("Formatting error: %v", err)
	} else {
		resultStr = string(formatted)
	}

	// Check for problematic syntax
	if strings.Contains(resultStr, "(lazy).ApplyUpdate") {
		t.Errorf("❌ Found problematic (lazy).ApplyUpdate syntax")
	}
	if strings.Contains(resultStr, "(lazy).Push") {
		t.Errorf("❌ Found problematic (lazy).Push syntax")
	}

	t.Logf("Generated method call syntax code:\n%s", resultStr)
}
