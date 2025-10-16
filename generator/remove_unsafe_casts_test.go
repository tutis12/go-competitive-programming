package main

import (
	"strings"
	"testing"
)

func TestRemoveUnsafeCasts(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name: "Simple unsafe cast removal",
			input: `package main

import "unsafe"

func test() {
	var key int
	result := (*(*H)(unsafe.Pointer(&key))).Hash()
}`,
			expected: `package main

func test() {
	var key int
	result := H(key).Hash()
}`,
		},
		{
			name: "Multiple unsafe casts",
			input: `package main

import "unsafe"

func test() {
	var key1 int
	var key2 string
	result1 := (*(*intHash)(unsafe.Pointer(&key1))).Hash()
	result2 := (*(*stringHash)(unsafe.Pointer(&key2))).Hash()
}`,
			expected: `package main

func test() {
	var key1 int
	var key2 string
	result1 := intHash(key1).Hash()
	result2 := stringHash(key2).Hash()
}`,
		},
		{
			name: "No unsafe usage - should not remove import",
			input: `package main

import "unsafe"

func test() {
	ptr := unsafe.Pointer(&someVar)
}`,
			expected: `package main

import "unsafe"

func test() {
	ptr := unsafe.Pointer(&someVar)
}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := string(RemoveUnsafeCasts([]byte(tt.input)))

			// Normalize whitespace for comparison - remove extra blank lines
			result = strings.TrimSpace(result)
			expected := strings.TrimSpace(tt.expected)

			// Remove multiple consecutive blank lines
			result = strings.Join(strings.Fields(strings.ReplaceAll(result, "\n", " NEWLINE ")), " ")
			result = strings.ReplaceAll(result, " NEWLINE NEWLINE ", " NEWLINE ")
			result = strings.ReplaceAll(result, " NEWLINE ", "\n")

			expected = strings.Join(strings.Fields(strings.ReplaceAll(expected, "\n", " NEWLINE ")), " ")
			expected = strings.ReplaceAll(expected, " NEWLINE NEWLINE ", " NEWLINE ")
			expected = strings.ReplaceAll(expected, " NEWLINE ", "\n")

			if result != expected {
				t.Errorf("RemoveUnsafeCasts() failed\nInput:\n%s\n\nExpected:\n%s\n\nGot:\n%s", tt.input, expected, result)
			}
		})
	}
}
