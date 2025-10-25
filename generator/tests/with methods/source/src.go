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
	var s X[int]
	s.Print1()
	s.Print2()
}
