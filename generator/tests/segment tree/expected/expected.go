package main

import "fmt"

func PrintSlice[T any](s []T) {
	for _, v := range s {
		fmt.Println(v)
	}
}

func main() {
	var s []int
	PrintSliceG1int(s)
}
func PrintSliceG1int(s []int,

) {
	for _, v := range s {
		fmt.Println(v)
	}
}
