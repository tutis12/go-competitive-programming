package main

import (
	"math/bits"
)

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

func CastBool(b bool) int {
	if b {
		return 1
	} else {
		return 0
	}
}

func MapArray[X, Y any](arr []X, f func(X) Y) []Y {
	result := make([]Y, len(arr))
	for i, v := range arr {
		result[i] = f(v)
	}
	return result
}

func Transpose(X *[][]int) {
	M := *X
	n := len(M)
	m := len(M[0])
	M2 := make([][]int, m)
	for i := 0; i < m; i++ {
		M2[i] = make([]int, n)
		for j := 0; j < n; j++ {
			M2[i][j] = M[j][i]
		}
	}
	*X = M2
}

func CollectMap[X comparable](m map[X]struct{}) []X {
	res := make([]X, 0, len(m))
	for k := range m {
		res = append(res, k)
	}
	return res
}
