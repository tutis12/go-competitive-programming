package main

import "fmt"

type LL[T any] struct {
	Next  *LL[T]
	Value T
	Prev  *LL[T]
}

func (l *LL[T]) Append(value T) {
	newNode := &LL[T]{Value: value}
	if l == nil {
		l = newNode
		return
	}
	for curr := l; curr != nil; curr = curr.Next {
		if curr.Next == nil {
			curr.Next = newNode
			newNode.Prev = curr
			break
		}
	}
}

func (l *LL[T]) Prepend(value T) {
	newNode := &LL[T]{Value: value}
	if l == nil {
		l = newNode
		return
	}
	newNode.Next = l
	l.Prev = newNode
	l = newNode
}

func (l *LL[T]) Remove() {
	if l == nil {
		return
	}
	if l.Prev != nil {
		l.Prev.Next = l.Next
	}
	if l.Next != nil {
		l.Next.Prev = l.Prev
	}
	l.Next = nil
	l.Prev = nil
}

func (l *LL[T]) Print() {
	for curr := l; curr != nil; curr = curr.Next {
		fmt.Print(curr.Value, " ")
	}
	fmt.Println()
}

func (l *LL[T]) PrintReverse() int {
	for curr := l; curr != nil; curr = curr.Prev {
		fmt.Print(curr.Value, " ")
	}
	fmt.Println()
	return 0
}

func main() {
	var s LL[[]LL[int]]
	fmt.Println(s.PrintReverse())
}
