package main

import (
	"strings"
	"testing"
)

func TestGenericArrayTypes(t *testing.T) {
	code := `package main

// Generic array type (with fixed size)
type GenericArray[T any] [10]T

// Generic slice type
type GenericSlice[T any] []T

// Generic map type
type GenericMap[K comparable, V any] map[K]V

// Generic channel type
type GenericChan[T any] chan T

// Generic pointer type
type GenericPtr[T any] *T

func testFunc() {
	// Test instantiations
	var a GenericArray[int]
	var b GenericSlice[string]
	var c GenericMap[string, int]
	var d GenericChan[bool]
	var e GenericPtr[float64]
	
	_ = a
	_ = b
	_ = c
	_ = d
	_ = e
}
`

	result := RemoveGenerics([]byte(code))
	resultStr := string(result)

	// Check that concrete types were generated for all generic type variants
	tests := []struct {
		name     string
		expected string
	}{
		{"GenericArray concrete type", "type GenericArrayG1 [10]int"},
		{"GenericSlice concrete type", "type GenericSliceG1 []string"},
		{"GenericMap concrete type", "type GenericMapG1 map[string]int"},
		{"GenericChan concrete type", "type GenericChanG1 chan bool"},
		{"GenericPtr concrete type", "type GenericPtrG1 *float64"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if !strings.Contains(resultStr, tt.expected) {
				t.Errorf("Missing expected concrete type: %s", tt.expected)
			}
		})
	}

	// Check that original generic declarations were removed
	if strings.Contains(resultStr, "type GenericArray[T any]") {
		t.Error("Original generic GenericArray declaration should be removed")
	}
	if strings.Contains(resultStr, "type GenericSlice[T any]") {
		t.Error("Original generic GenericSlice declaration should be removed")
	}

	t.Logf("Result:\n%s", resultStr)
}
