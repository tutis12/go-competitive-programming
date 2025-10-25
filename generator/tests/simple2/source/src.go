package main

import "fmt"

func PrintSlice[T any](X []T) {
	for _, X := range X {
		fmt.Println(X)
	}
}

func T[X any](T []X) {
	for _, T := range T {
		fmt.Println(T)
	}
}

type X[T any] struct {
	Value T
}

func main() {
	var s []X[X[int]]
	PrintSlice(s)
	var t []X[int]
	PrintSlice(t)
}
