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
	var s LLG1SliceLLOfint
	fmt.Println(s.PrintReverse())
}

type LLG1SliceLLOfint struct {
	Next  *LLG1SliceLLOfint
	Value []LLG1int
	Prev  *LLG1SliceLLOfint
}

func (l *LLG1SliceLLOfint) Append(value []LLG1int) {
	newNode := &LLG1SliceLLOfint{Value: value}
	if l == nil {
		l = newNode
		return
	}
	for curr := l; curr != nil; curr = curr.
		Next {
		if curr.
			Next == nil {
			curr.
				Next = newNode
			newNode.Prev = curr
			break
		}
	}
}
func (l *LLG1SliceLLOfint) Prepend(value []LLG1int) {
	newNode := &LLG1SliceLLOfint{Value: value}
	if l == nil {
		l = newNode
		return
	}
	newNode.
		Next = l
	l.Prev = newNode
	l = newNode
}
func (l *LLG1SliceLLOfint) Remove() {
	if l == nil {
		return
	}
	if l.Prev !=
		nil {
		l.
			Prev.
			Next = l.Next
	}
	if l.Next != nil {
		l.Next.Prev = l.Prev
	}
	l.
		Next = nil
	l.
		Prev = nil
}
func (l *LLG1SliceLLOfint) Print() {
	for curr := l; curr != nil; curr = curr.
		Next {
		fmt.
			Print(curr.Value, " ")
	}
	fmt.Println()
}
func (l *LLG1SliceLLOfint) PrintReverse() int {
	for curr := l; curr !=
		nil; curr = curr.
		Prev {
		fmt.Print(curr.Value, " ")
	}
	fmt.Println()
	return 0
}

type LLG1int struct {
	Next  *LLG1int
	Value int

	Prev *LLG1int
}

func (l *LLG1int) Append(value int,

) {
	newNode := &LLG1int{Value: value}
	if l == nil {
		l = newNode
		return
	}
	for curr := l; curr != nil; curr = curr.
		Next {
		if curr.
			Next == nil {
			curr.
				Next = newNode
			newNode.Prev = curr
			break
		}
	}
}
func (l *LLG1int) Prepend(value int,

) {
	newNode := &LLG1int{Value: value}
	if l == nil {
		l = newNode
		return
	}
	newNode.
		Next = l
	l.Prev = newNode
	l = newNode
}
func (l *LLG1int) Remove() {
	if l == nil {
		return
	}
	if l.Prev !=
		nil {
		l.
			Prev.
			Next = l.Next
	}
	if l.Next != nil {
		l.Next.Prev = l.Prev
	}
	l.
		Next = nil
	l.
		Prev = nil
}
func (l *LLG1int) Print() {
	for curr := l; curr != nil; curr = curr.
		Next {
		fmt.
			Print(curr.Value, " ")
	}
	fmt.Println()
}
func (l *LLG1int) PrintReverse() int {
	for curr := l; curr !=
		nil; curr = curr.
		Prev {
		fmt.Print(curr.Value, " ")
	}
	fmt.Println()
	return 0
}
