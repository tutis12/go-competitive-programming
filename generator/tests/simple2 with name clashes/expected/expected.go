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
	TG1XOfXOfint(s)
	var T []XG1int
	PrintSliceG1XOfint(T)
	{
		type T struct{}
		var X []XG1T
		PrintSliceG1XOfT(X)
	}
	{
		type T struct{}
		var X []XG1XOfT
		PrintSliceG1XOfXOfT(X)
	}
	{
		type X struct{}
		var T []X
		PrintSliceG1X(T)
	}
	var X []XG1int
	PrintSliceG1XOfint(X)
}
func TG1XOfXOfint(T []XG1XOfint,

) {
	for _, T := range T {
		fmt.
			Println(T)
	}
}
func PrintSliceG1XOfint(X []XG1int) {
	for _, X := range X {
		fmt.Println(X)
	}
}

type XG1T struct {
	Value T
}

func PrintSliceG1XOfT(X []XG1T) {
	for _, X := range X {
		fmt.Println(X)
	}
}
func PrintSliceG1X(X []X,

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
type XG1XOfT struct {
	Value XG1XOfT
}

func PrintSliceG1XOfXOfT(X []XG1XOfT) {
	for _, X := range X {
		fmt.Println(X)
	}
}
func PrintSliceG1XOfXOfint(X []XG1XOfint) {
	for _, X := range X {
		fmt.Println(X)
	}
}
