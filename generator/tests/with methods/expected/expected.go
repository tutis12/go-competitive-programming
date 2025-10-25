package main

import "fmt"

type X[T any] struct {
	Value *T
}

func (x X[T]) Print1() {
	fmt.Println(x.Value)
}

func (x *X[T]) Print2() {
	fmt.Println(x.Value)
}

func main() {
	var s XG1int
	s.Print1()
	s.Print2()
}

type XG1int struct{ Value *int }

func (x XG1int) Print1() {
	fmt.Println(x.
		Value)
}
func (x *XG1int) Print2() {
	fmt.Println(x.
		Value)
}
