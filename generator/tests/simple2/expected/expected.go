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
	var s []XG1XOfint
	PrintSliceG1XOfXOfint(s)
	var t []XG1int
	PrintSliceG1XOfint(t)
}
func PrintSliceG1XOfXOfint(X []XG1XOfint) {
	for _, X := range X {
		fmt.Println(X)
	}
}
func PrintSliceG1XOfint(X []XG1int,

) {
	for _, X := range X {
		fmt.Println(X)
	}
}

type XG1XOfint struct {
	Value XG1int
}
type XG1int struct {
	Value int
}
