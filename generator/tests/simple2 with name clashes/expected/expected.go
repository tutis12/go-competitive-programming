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

		var X []XG1Local_main_B6_B7_T
		PrintSliceG1XOfLocal_main_B6_B7_T(X)
	}
	{

		var X []XG1XOfLocal_main_B10_B11_T
		PrintSliceG1XOfXOfLocal_main_B10_B11_T(X)
	}
	{

		var T []Local_main_B14_B15_X
		PrintSliceG1Local_main_B14_B15_X(T)
	}
	var X []XG1int
	PrintSliceG1XOfint(X)
}

type Local_main_B6_B7_T struct{}
type Local_main_B10_B11_T struct{}
type Local_main_B14_B15_X struct{}

func PrintSliceG1Local_main_B14_B15_X(X []Local_main_B14_B15_X,

) {
	for _, X := range X {
		fmt.Println(X)
	}
}
func PrintSliceG1XOfLocal_main_B6_B7_T(X []XG1Local_main_B6_B7_T,

) {
	for _, X := range X {
		fmt.Println(X)
	}
}
func PrintSliceG1XOfXOfLocal_main_B10_B11_T(X []XG1XOfLocal_main_B10_B11_T,

) {
	for _, X := range X {
		fmt.Println(X)
	}
}
func PrintSliceG1XOfXOfint(X []XG1XOfint,

) {
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
func TG1XOfXOfint(T []XG1XOfint,

) {
	for _, T := range T {
		fmt.
			Println(T)
	}
}

type XG1Local_main_B10_B11_T struct{ Value Local_main_B10_B11_T }
type XG1Local_main_B6_B7_T struct{ Value Local_main_B6_B7_T }
type XG1XOfLocal_main_B10_B11_T struct{ Value XG1Local_main_B10_B11_T }
type XG1XOfint struct{ Value XG1int }
type XG1int struct{ Value int }
