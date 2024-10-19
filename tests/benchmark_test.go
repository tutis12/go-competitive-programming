package tests

import (
	"sync"
	"testing"
)

func isPrime(x int) bool {
	for y := 2; y < x; y++ {
		if x%y == 0 {
			return false
		}
	}
	return true
}

// 312883958
func BenchmarkSetValues(b *testing.B) {
	for range b.N {
		slice := make([]int, 100000)
		for i := range slice {
			if isPrime(i) {
				slice[i] = i * 10
			} else {
				slice[i] = i * 100
			}
		}
	}
}

// 65245901
func BenchmarkSetValuesGo(b *testing.B) {
	for range b.N {
		slice := make([]int, 100000)
		wg := sync.WaitGroup{}
		wg.Add(len(slice))
		for i := range slice {
			go func() {
				if isPrime(i) {
					slice[i] = i * 10
				} else {
					slice[i] = i * 100
				}
				wg.Done()
			}()
		}
		wg.Wait()
	}
}
