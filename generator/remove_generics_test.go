package main

import (
	"fmt"
	"go/format"
	"strings"
	"testing"
)

func TestRemoveGenerics(t *testing.T) {
	// Test case 1: Simple generic struct with method
	testCase1 := `package main

type Container[T any] struct {
	value T
}

func (c *Container[T]) Get() T {
	return c.value
}

func (c *Container[T]) Set(v T) {
	c.value = v
}

func NewContainer[T any](initial T) *Container[T] {
	return &Container[T]{value: initial}
}

func main() {
	intContainer := NewContainer[int](42)
	stringContainer := NewContainer[string]("hello")
	
	intContainer.Set(100)
	stringContainer.Set("world")
	
	fmt.Println(intContainer.Get())
	fmt.Println(stringContainer.Get())
}`

	result := RemoveGenerics([]byte(testCase1))
	resultStr := string(result)

	// Format the result for better readability
	formatted, err := format.Source(result)
	if err != nil {
		t.Logf("Formatting error: %v", err)
		t.Logf("Raw result:\n%s", resultStr)
	} else {
		resultStr = string(formatted)
	}

	// Verify expected transformations
	expectedTransformations := []struct {
		description      string
		shouldContain    string
		shouldNotContain string
	}{
		{"Generic struct should be removed", "", "type Container[T any]"},
		{"Concrete struct ContainerG1 should exist", "type ContainerG1 struct", ""},
		{"Concrete struct ContainerG2 should exist", "type ContainerG2 struct", ""},
		{"Generic method should be removed", "", "func (c *Container[T])"},
		{"Concrete method for ContainerG1 should exist", "func (c *ContainerG1) Get() int", ""},
		{"Concrete method for ContainerG2 should exist", "func (c *ContainerG2) Get() string", ""},
		{"Generic function should be removed", "", "func NewContainer[T any]"},
		{"Concrete function NewContainerG1 should exist", "func NewContainerG1(initial int) *ContainerG1", ""},
		{"Concrete function NewContainerG2 should exist", "func NewContainerG2(initial string) *ContainerG2", ""},
		{"Generic calls should be converted", "NewContainerG1(42)", "NewContainer[int]"},
		{"Generic calls should be converted", "NewContainerG2(\"hello\")", "NewContainer[string]"},
	}

	for _, check := range expectedTransformations {
		if check.shouldContain != "" {
			if !strings.Contains(resultStr, check.shouldContain) {
				t.Errorf("❌ %s: Expected to contain '%s'", check.description, check.shouldContain)
			} else {
				t.Logf("✅ %s: Found '%s'", check.description, check.shouldContain)
			}
		}
		if check.shouldNotContain != "" {
			if strings.Contains(resultStr, check.shouldNotContain) {
				t.Errorf("❌ %s: Should not contain '%s'", check.description, check.shouldNotContain)
			} else {
				t.Logf("✅ %s: Correctly removed '%s'", check.description, check.shouldNotContain)
			}
		}
	}

	fmt.Println("\n=== GENERATED CODE ===")
	fmt.Println(resultStr)
	fmt.Println("=== END GENERATED CODE ===")
}

func TestRemoveGenericsSegmentTree(t *testing.T) {
	// Test case 2: More complex example with nested generics (simplified segment tree)
	testCase2 := `package main

type node[V, U any] struct {
	value  V
	update U
}

type ST[V, U any] struct {
	arr        []node[V, U]
	zeroValue  V
	zeroUpdate U
}

func NewST[V, U any](zeroValue V, zeroUpdate U) *ST[V, U] {
	return &ST[V, U]{
		arr:        make([]node[V, U], 10),
		zeroValue:  zeroValue,
		zeroUpdate: zeroUpdate,
	}
}

func (st *ST[V, U]) Get(i int) V {
	return st.arr[i].value
}

func main() {
	intST := NewST[int, int](0, 0)
	stringST := NewST[string, bool]("", false)
	
	intST.Get(0)
	stringST.Get(1)
}`

	result := RemoveGenerics([]byte(testCase2))
	resultStr := string(result)

	// Format the result for better readability
	formatted, err := format.Source(result)
	if err != nil {
		t.Logf("Formatting error: %v", err)
		t.Logf("Raw result:\n%s", resultStr)
	} else {
		resultStr = string(formatted)
	}

	// Verify expected transformations for segment tree
	expectedTransformations := []struct {
		description      string
		shouldContain    string
		shouldNotContain string
	}{
		{"Generic node should be removed", "", "type node[V, U any]"},
		{"Generic ST should be removed", "", "type ST[V, U any]"},
		{"Concrete nodeG1 should exist", "nodeG1", ""},
		{"Concrete nodeG2 should exist", "nodeG2", ""},
		{"Concrete STG1 should exist", "STG1", ""},
		{"Concrete STG2 should exist", "STG2", ""},
		{"Generic NewST should be removed", "", "func NewST[V, U any]"},
		{"Concrete NewSTG1 should exist", "NewSTG1", ""},
		{"Concrete NewSTG2 should exist", "NewSTG2", ""},
		{"Generic calls should be converted", "NewSTG1(0, 0)", "NewST[int, int]"},
		{"Generic calls should be converted", "NewSTG2(\"\", false)", "NewST[string, bool]"},
	}

	for _, check := range expectedTransformations {
		if check.shouldContain != "" {
			if !strings.Contains(resultStr, check.shouldContain) {
				t.Errorf("❌ %s: Expected to contain '%s'", check.description, check.shouldContain)
			} else {
				t.Logf("✅ %s: Found '%s'", check.description, check.shouldContain)
			}
		}
		if check.shouldNotContain != "" {
			if strings.Contains(resultStr, check.shouldNotContain) {
				t.Errorf("❌ %s: Should not contain '%s'", check.description, check.shouldNotContain)
			} else {
				t.Logf("✅ %s: Correctly removed '%s'", check.description, check.shouldNotContain)
			}
		}
	}

	fmt.Println("\n=== SEGMENT TREE GENERATED CODE ===")
	fmt.Println(resultStr)
	fmt.Println("=== END SEGMENT TREE GENERATED CODE ===")
}

func TestRemoveGenericsHashMap(t *testing.T) {
	// Test case 3: HashMap with interface constraints
	testCase3 := `package main

type Hasher[T any] interface {
	Hash() uint64
}

type entry[K Hasher[K], V any] struct {
	hash  uint64
	key   K
	value V
}

type HashMap[K Hasher[K], V any] struct {
	buckets [][]entry[K, V]
	size    int
}

func NewHashMap[K Hasher[K], V any](initialSize int) *HashMap[K, V] {
	return &HashMap[K, V]{
		buckets: make([][]entry[K, V], initialSize),
		size:    0,
	}
}

func (hm *HashMap[K, V]) Set(key K, value V) {
	hash := key.Hash()
	hm.buckets[0] = append(hm.buckets[0], entry[K, V]{
		hash:  hash,
		key:   key,
		value: value,
	})
}

type intHash int

func (h intHash) Hash() uint64 {
	return uint64(h)
}

func main() {
	hashMap := NewHashMap[intHash, int](10)
	hashMap.Set(intHash(1), 42)
}`

	result := RemoveGenerics([]byte(testCase3))
	resultStr := string(result)

	// Format the result for better readability
	formatted, err := format.Source(result)
	if err != nil {
		t.Logf("Formatting error: %v", err)
		t.Logf("Raw result:\n%s", resultStr)
	} else {
		resultStr = string(formatted)
	}

	// Verify expected transformations for HashMap
	expectedTransformations := []struct {
		description      string
		shouldContain    string
		shouldNotContain string
	}{
		{"Generic HashMap should be removed", "", "type HashMap[K Hasher[K], V any]"},
		{"Generic entry should be removed", "", "type entry[K Hasher[K], V any]"},
		{"Concrete HashMapG1 should exist", "HashMapG1", ""},
		{"Concrete entryG1 should exist", "entryG1", ""},
		{"Generic NewHashMap should be removed", "", "func NewHashMap[K Hasher[K], V any]"},
		{"Concrete NewHashMapG1 should exist", "NewHashMapG1", ""},
		{"Generic calls should be converted", "NewHashMapG1(10)", "NewHashMap[intHash, int]"},
		{"intHash type should be preserved", "type intHash int", ""},
		{"Hash method should be preserved", "func (h intHash) Hash() uint64", ""},
	}

	for _, check := range expectedTransformations {
		if check.shouldContain != "" {
			if !strings.Contains(resultStr, check.shouldContain) {
				t.Errorf("❌ %s: Expected to contain '%s'", check.description, check.shouldContain)
			} else {
				t.Logf("✅ %s: Found '%s'", check.description, check.shouldContain)
			}
		}
		if check.shouldNotContain != "" {
			if strings.Contains(resultStr, check.shouldNotContain) {
				t.Errorf("❌ %s: Should not contain '%s'", check.description, check.shouldNotContain)
			} else {
				t.Logf("✅ %s: Correctly removed '%s'", check.description, check.shouldNotContain)
			}
		}
	}

	fmt.Println("\n=== HASHMAP GENERATED CODE ===")
	fmt.Println(resultStr)
	fmt.Println("=== END HASHMAP GENERATED CODE ===")
}

func TestGenerateTestProducesCompilableCode(t *testing.T) {
	// This test catches the specific issues that generate_test.go produces
	// It's a minimal version that focuses on the problematic patterns
	testCase := `package main

import "fmt"

type HashMap[K Hasher[K], V any] struct {
	buckets [][]entry[K, V]
	size    int
}

type entry[K Hasher[K], V any] struct {
	key   K
	value V
}

type Hasher[T any] interface {
	Hash() uint64
}

func NewHashMap[K Hasher[K], V any](initialSize int) *HashMap[K, V] {
	return &HashMap[K, V]{
		buckets: make([][]entry[K, V], initialSize),
		size:    0,
	}
}

type intHash int

func (h intHash) Hash() uint64 {
	return uint64(h)
}

func main() {
	// This call should be converted to NewHashMapG1(10)
	a := NewHashMap[intHash, int](10)
	// This call should also be converted to NewHashMapG1(20)  
	b := NewHashMap[intHash, int](20)
	
	fmt.Println(a, b)
}`

	result := RemoveGenerics([]byte(testCase))
	resultStr := string(result)

	// Format the result for better readability
	formatted, err := format.Source(result)
	if err != nil {
		t.Logf("Formatting error: %v", err)
		t.Logf("Raw result:\n%s", resultStr)
	} else {
		resultStr = string(formatted)
	}

	// These are the critical checks that catch the generate_test.go issues
	criticalChecks := []struct {
		description      string
		shouldContain    string
		shouldNotContain string
	}{
		{"Generic HashMap calls should be converted", "NewHashMapG1(10)", "NewHashMap[intHash, int](10)"},
		{"All generic HashMap calls should be converted", "NewHashMapG1(20)", "NewHashMap[intHash, int](20)"},
		{"No generic function calls should remain", "", "NewHashMap["},
		{"Concrete HashMap type should exist", "HashMapG1", ""},
		{"Concrete entry type should exist", "entryG1", ""},
		{"Concrete NewHashMap function should exist", "func NewHashMapG1(", ""},
		{"intHash type should be preserved", "type intHash int", ""},
	}

	hasErrors := false
	for _, check := range criticalChecks {
		if check.shouldContain != "" {
			if !strings.Contains(resultStr, check.shouldContain) {
				t.Errorf("❌ CRITICAL: %s - Expected to contain '%s'", check.description, check.shouldContain)
				hasErrors = true
			} else {
				t.Logf("✅ %s: Found '%s'", check.description, check.shouldContain)
			}
		}
		if check.shouldNotContain != "" {
			if strings.Contains(resultStr, check.shouldNotContain) {
				t.Errorf("❌ CRITICAL: %s - Should not contain '%s'", check.description, check.shouldNotContain)
				hasErrors = true
			} else {
				t.Logf("✅ %s: Correctly removed '%s'", check.description, check.shouldNotContain)
			}
		}
	}

	if hasErrors {
		t.Errorf("❌ CRITICAL: RemoveGenerics failed to properly transform generic calls - this will cause compilation errors in generated code!")
		fmt.Println("\n=== PROBLEMATIC GENERATED CODE ===")
		fmt.Println(resultStr)
		fmt.Println("=== END PROBLEMATIC CODE ===")
	} else {
		t.Logf("✅ All critical transformations passed - generated code should compile")
	}
}

func TestGenericUtilityFunctions(t *testing.T) {
	// This test catches the utility function transformation issues that cause
	// "undefined: Log2Ceil" and "undefined: IsPowerOf2" errors
	testCase := `package main

import "fmt"

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

func Log2Floor[T int | uint64](x T) int {
	x64 := uint64(x)
	if x64 == 0 {
		panic("Log2(0) is undefined")
	}
	return 63 - bits.LeadingZeros64(x64)
}

func IsPowerOf2[T int | uint64](x T) bool {
	x64 := uint64(x)
	return x64 != 0 && (x64&(x64-1)) == 0
}

func main() {
	size := 10
	// These calls should be converted to concrete versions
	log2n := Log2Ceil(size)        // should become Log2CeilG1(size) 
	power := IsPowerOf2(8)         // should become IsPowerOf2G1(8)
	log2n2 := Log2Ceil(uint64(20)) // should become Log2CeilG2(uint64(20))
	
	fmt.Println(log2n, power, log2n2)
}`

	result := RemoveGenerics([]byte(testCase))
	resultStr := string(result)

	// Format the result for better readability
	formatted, err := format.Source(result)
	if err != nil {
		t.Logf("Formatting error: %v", err)
		t.Logf("Raw result:\n%s", resultStr)
	} else {
		resultStr = string(formatted)
	}

	// These checks catch the utility function transformation issues
	utilityChecks := []struct {
		description      string
		shouldContain    string
		shouldNotContain string
	}{
		{"Log2Ceil with int should be converted", "Log2CeilG1(size)", "Log2Ceil(size)"},
		{"IsPowerOf2 with int should be converted", "IsPowerOf2G1(8)", "IsPowerOf2(8)"},
		{"Log2Ceil with uint64 should be converted", "Log2CeilG2(uint64(20))", "Log2Ceil(uint64(20))"},
		{"No generic utility calls should remain", "", "Log2Ceil["},
		{"No generic utility calls should remain", "", "IsPowerOf2["},
		{"Concrete Log2CeilG1 function should exist", "func Log2CeilG1(x int) int", ""},
		{"Concrete IsPowerOf2G1 function should exist", "func IsPowerOf2G1(x int) bool", ""},
		{"Concrete Log2CeilG2 function should exist", "func Log2CeilG2(x uint64) int", ""},
	}

	hasUtilityErrors := false
	for _, check := range utilityChecks {
		if check.shouldContain != "" {
			if !strings.Contains(resultStr, check.shouldContain) {
				t.Errorf("❌ UTILITY ISSUE: %s - Expected to contain '%s'", check.description, check.shouldContain)
				hasUtilityErrors = true
			} else {
				t.Logf("✅ %s: Found '%s'", check.description, check.shouldContain)
			}
		}
		if check.shouldNotContain != "" {
			if strings.Contains(resultStr, check.shouldNotContain) {
				t.Errorf("❌ UTILITY ISSUE: %s - Should not contain '%s'", check.description, check.shouldNotContain)
				hasUtilityErrors = true
			} else {
				t.Logf("✅ %s: Correctly removed '%s'", check.description, check.shouldNotContain)
			}
		}
	}

	if hasUtilityErrors {
		t.Errorf("❌ CRITICAL: Generic utility functions not properly transformed - this causes 'undefined' errors in generated code!")
		fmt.Println("\n=== UTILITY TRANSFORMATION ISSUES ===")
		fmt.Println(resultStr)
		fmt.Println("=== END UTILITY ISSUES ===")
	} else {
		t.Logf("✅ All utility function transformations passed")
	}
}

func TestRealWorldPatterns(t *testing.T) {
	// This test reproduces the exact patterns found in the failing generated code
	testCase := `package main

import "fmt"

type HashMap[K Hasher[K], V any] struct {
	buckets [][]entry[K, V]
	size    int
}

type entry[K Hasher[K], V any] struct {
	key   K
	value V
}

type Hasher[T any] interface {
	Hash() uint64
}

func NewHashMap[K Hasher[K], V any](initialSize int) *HashMap[K, V] {
	return &HashMap[K, V]{
		buckets: make([][]entry[K, V], initialSize),
		size:    0,
	}
}

func Log2Ceil[T int | uint64](x T) int {
	return 0 // simplified
}

func IsPowerOf2[T int | uint64](x T) bool {
	return false // simplified
}

type intHash int

func (h intHash) Hash() uint64 {
	return uint64(h)
}

func testFunc() {
	// Variables to simulate the real patterns
	n := 10
	size := 20
	i := 4
	
	// These patterns are found in the failing generated code:
	a := NewHashMap[intHash, int](n)    // line 205 in generated_main.go
	dp := NewHashMap[intHash, int](n)   // line 225 in generated_main.go  
	log2n := Log2Ceil(size)             // line 262 in generated_main.go
	isPow := IsPowerOf2(i)              // line 547 in generated_main.go
	
	fmt.Println(a, dp, log2n, isPow)
}

func main() {
	testFunc()
}`

	result := RemoveGenerics([]byte(testCase))
	resultStr := string(result)

	// Format the result for better readability
	formatted, err := format.Source(result)
	if err != nil {
		t.Logf("Formatting error: %v", err)
		t.Logf("Raw result:\n%s", resultStr)
	} else {
		resultStr = string(formatted)
	}

	// These are the exact patterns causing compilation failures
	realWorldChecks := []struct {
		description      string
		shouldContain    string
		shouldNotContain string
	}{
		{"NewHashMap with explicit types should be converted", "NewHashMapG1(n)", "NewHashMap[intHash, int](n)"},
		{"Log2Ceil with variable should be converted", "Log2CeilG1(size)", "Log2Ceil(size)"},
		{"IsPowerOf2 with variable should be converted", "IsPowerOf2G1(i)", "IsPowerOf2(i)"},
		{"No undefined NewHashMap calls should remain", "", "NewHashMap["},
		{"No undefined Log2Ceil calls should remain", "", "Log2Ceil("},
		{"No undefined IsPowerOf2 calls should remain", "", "IsPowerOf2("},
		{"Concrete NewHashMapG1 should exist", "func NewHashMapG1(", ""},
		{"Concrete Log2CeilG1 should exist", "func Log2CeilG1(", ""},
		{"Concrete IsPowerOf2G1 should exist", "func IsPowerOf2G1(", ""},
	}

	hasRealWorldErrors := false
	for _, check := range realWorldChecks {
		if check.shouldContain != "" {
			if !strings.Contains(resultStr, check.shouldContain) {
				t.Errorf("❌ REAL-WORLD ISSUE: %s - Expected to contain '%s'", check.description, check.shouldContain)
				hasRealWorldErrors = true
			} else {
				t.Logf("✅ %s: Found '%s'", check.description, check.shouldContain)
			}
		}
		if check.shouldNotContain != "" {
			if strings.Contains(resultStr, check.shouldNotContain) {
				t.Errorf("❌ REAL-WORLD ISSUE: %s - Should not contain '%s'", check.description, check.shouldNotContain)
				hasRealWorldErrors = true
			} else {
				t.Logf("✅ %s: Correctly removed '%s'", check.description, check.shouldNotContain)
			}
		}
	}

	if hasRealWorldErrors {
		t.Errorf("❌ CRITICAL: Real-world patterns not properly transformed - this matches the exact compilation failures!")
		fmt.Println("\n=== REAL-WORLD PATTERN ISSUES ===")
		fmt.Println(resultStr)
		fmt.Println("=== END REAL-WORLD ISSUES ===")
	} else {
		t.Logf("✅ All real-world pattern transformations passed")
	}
}
