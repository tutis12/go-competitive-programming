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
	T(s)
	var T []X[int]
	PrintSlice(T)
	{
		type T struct{}
		var X []X[T]
		PrintSlice(X)
	}
	{
		type T struct{}
		var X []X[X[T]]
		PrintSlice(X)
	}
	{
		type X struct{}
		var T []X
		PrintSlice(T)
	}
	var X []X[int]
	PrintSlice(X)
}
