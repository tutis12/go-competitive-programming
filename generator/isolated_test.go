package main

import (
	"strings"
	"testing"
)

func TestIsPowerOf2Isolation(t *testing.T) {
	code := `package main

func IsPowerOf2[T int | uint64](x T) bool {
	return x > 0 && (x&(x-1)) == 0
}

func testFunc() {
	i := 42
	result := IsPowerOf2(i)
	_ = result
}
`

	result := RemoveGenerics([]byte(code))
	resultStr := string(result)

	t.Logf("Input:\n%s", code)
	t.Logf("Output:\n%s", resultStr)

	if strings.Contains(resultStr, "IsPowerOf2(i)") {
		t.Error("Call IsPowerOf2(i) was not replaced")
	}
	if !strings.Contains(resultStr, "IsPowerOf2G1(i)") {
		t.Error("Expected IsPowerOf2G1(i) in output")
	}
}