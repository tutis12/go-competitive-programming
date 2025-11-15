package rbtree

import "iter"

type Color bool

const (
	Red   Color = false
	Black Color = true
)

type Node[T any] struct {
	Color  Color
	Parent *Node[T]
	Childs [2]*Node[T]
	Value  T
}

type RBTree[T any] struct {
	Root *Node[T]
	Size int
	Less func(a, b T) bool
}

func NewRBTree[T any](
	less func(a, b T) bool,
) *RBTree[T] {
	return &RBTree[T]{
		Root: nil,
		Size: 0,
		Less: less,
	}
}

func (tree *RBTree[T]) Insert(value T) *Node[T] {
	return tree.insert(value, false)
}

func (tree *RBTree[T]) InsertOrReplace(value T) *Node[T] {
	return tree.insert(value, true)
}

func (tree *RBTree[T]) insert(value T, override bool) *Node[T] {
	if tree.Root == nil {
		newNode := &Node[T]{
			Color:  Black,
			Childs: [2]*Node[T]{nil, nil},
			Value:  value,
		}
		tree.Root = newNode
		return newNode
	}

	cur := tree.Root
	var parent *Node[T]
	var lastLeftTurn *Node[T]
	var lastRightTurn *Node[T]

	for cur != nil {
		parent = cur
		if tree.Less(value, cur.Value) {
			lastRightTurn = cur
			cur = cur.Childs[0]
		} else {
			lastLeftTurn = cur
			cur = cur.Childs[1]
		}
	}

	if override {
		if lastLeftTurn != nil && !tree.Less(value, lastLeftTurn.Value) {
			lastLeftTurn.Value = value
			return lastLeftTurn
		}
		if lastRightTurn != nil && !tree.Less(lastRightTurn.Value, value) {
			lastRightTurn.Value = value
			return lastRightTurn
		}
	}

	newNode := &Node[T]{
		Color:  Red,
		Parent: parent,
		Childs: [2]*Node[T]{nil, nil},
		Value:  value,
	}
	if tree.Less(value, parent.Value) {
		parent.Childs[0] = newNode
	} else {
		parent.Childs[1] = newNode
	}

	tree.insertFixup(newNode)
	return newNode
}

func (tree *RBTree[T]) insertFixup(node *Node[T]) {
	for node.Parent != nil && node.Parent.Color == Red {
		var parentDir uint8
		if node.Parent == node.Parent.Parent.Childs[0] {
			parentDir = 0
		} else {
			parentDir = 1
		}
		uncle := node.Parent.Parent.Childs[1-parentDir]
		if uncle != nil && uncle.Color == Red {
			node.Parent.Color = Black
			uncle.Color = Black
			node.Parent.Parent.Color = Red
			node = node.Parent.Parent
		} else {
			if node == node.Parent.Childs[1-parentDir] {
				node = node.Parent
				tree.rotate(node, parentDir)
			}
			node.Parent.Color = Black
			node.Parent.Parent.Color = Red
			tree.rotate(node.Parent.Parent, 1-parentDir)
		}
	}
	tree.Root.Color = Black
}

func (tree *RBTree[T]) rotate(node *Node[T], dir uint8) {
	oppositeDir := 1 - dir
	child := node.Childs[oppositeDir]
	node.Childs[oppositeDir] = child.Childs[dir]
	if child.Childs[dir] != nil {
		child.Childs[dir].Parent = node
	}
	child.Parent = node.Parent
	if node.Parent == nil {
		tree.Root = child
	} else {
		var parentDir int
		if node == node.Parent.Childs[0] {
			parentDir = 0
		} else {
			parentDir = 1
		}
		node.Parent.Childs[parentDir] = child
	}
	child.Childs[dir] = node
	node.Parent = child
}

func (tree *RBTree[T]) Find(value T) (*Node[T], bool) {
	cur := tree.Root
	for cur != nil {
		if tree.Less(value, cur.Value) {
			cur = cur.Childs[0]
		} else if tree.Less(cur.Value, value) {
			cur = cur.Childs[1]
		} else {
			return cur, true
		}
	}
	return nil, false
}

func (tree *RBTree[T]) Minimum(node *Node[T]) *Node[T] {
	cur := node
	for cur.Childs[0] != nil {
		cur = cur.Childs[0]
	}
	return cur
}

func (tree *RBTree[T]) Maximum(node *Node[T]) *Node[T] {
	cur := node
	for cur.Childs[1] != nil {
		cur = cur.Childs[1]
	}
	return cur
}

func (node *Node[T]) Successor() *Node[T] {
	if node.Childs[1] != nil {
		cur := node.Childs[1]
		for cur.Childs[0] != nil {
			cur = cur.Childs[0]
		}
		return cur
	}
	cur := node
	parent := node.Parent
	for parent != nil && cur == parent.Childs[1] {
		cur = parent
		parent = parent.Parent
	}
	return parent
}

func (node *Node[T]) Predecessor() *Node[T] {
	if node.Childs[0] != nil {
		cur := node.Childs[0]
		for cur.Childs[1] != nil {
			cur = cur.Childs[1]
		}
		return cur
	}
	cur := node
	parent := node.Parent
	for parent != nil && cur == parent.Childs[0] {
		cur = parent
		parent = parent.Parent
	}
	return parent
}

func (tree *RBTree[T]) Iter() iter.Seq[T] {
	node := tree.Minimum(tree.Root)
	return func(yield func(T) bool) {
		for node != nil {
			if !yield(node.Value) {
				return
			}
			node = node.Successor()
		}
	}
}

func (tree *RBTree[T]) ReverseIter() iter.Seq[T] {
	node := tree.Maximum(tree.Root)
	return func(yield func(T) bool) {
		for node != nil {
			if !yield(node.Value) {
				return
			}
			node = node.Predecessor()
		}
	}
}

func (tree *RBTree[T]) Clear() {
	tree.Root = nil
}

func (tree *RBTree[T]) IsEmpty() bool {
	return tree.Root == nil
}

func isBlack[T any](node *Node[T]) bool {
	return node == nil || node.Color == Black
}

func (tree *RBTree[T]) Remove(node *Node[T]) {
	if node == nil {
		return
	}

	transplant := func(u, v *Node[T]) {
		if u.Parent == nil {
			tree.Root = v
		} else if u == u.Parent.Childs[0] {
			u.Parent.Childs[0] = v
		} else {
			u.Parent.Childs[1] = v
		}
		if v != nil {
			v.Parent = u.Parent
		}
	}

	y := node
	yOriginalColor := y.Color
	var x *Node[T]
	var xParent *Node[T]

	if node.Childs[0] == nil {
		x = node.Childs[1]
		xParent = node.Parent
		transplant(node, node.Childs[1])
	} else if node.Childs[1] == nil {
		x = node.Childs[0]
		xParent = node.Parent
		transplant(node, node.Childs[0])
	} else {
		y = tree.Minimum(node.Childs[1])
		yOriginalColor = y.Color
		x = y.Childs[1]
		if y.Parent == node {
			xParent = y
		} else {
			xParent = y.Parent
			transplant(y, y.Childs[1])
			y.Childs[1] = node.Childs[1]
			y.Childs[1].Parent = y
		}
		transplant(node, y)
		y.Childs[0] = node.Childs[0]
		y.Childs[0].Parent = y
		y.Color = node.Color
	}

	if yOriginalColor == Black {
		tree.deleteFixup(x, xParent)
	}
}

func (tree *RBTree[T]) deleteFixup(node *Node[T], parent *Node[T]) {
	for node != tree.Root && isBlack(node) {
		if parent == nil {
			break
		}
		if node == parent.Childs[0] {
			sibling := parent.Childs[1]
			if sibling != nil && sibling.Color == Red {
				sibling.Color = Black
				parent.Color = Red
				tree.rotate(parent, 0)
				sibling = parent.Childs[1]
			}
			if sibling == nil {
				node = parent
				parent = node.Parent
				continue
			}
			if isBlack(sibling.Childs[0]) && isBlack(sibling.Childs[1]) {
				sibling.Color = Red
				node = parent
				parent = node.Parent
			} else {
				if isBlack(sibling.Childs[1]) {
					if sibling.Childs[0] != nil {
						sibling.Childs[0].Color = Black
					}
					sibling.Color = Red
					tree.rotate(sibling, 1)
					sibling = parent.Childs[1]
				}
				sibling.Color = parent.Color
				parent.Color = Black
				if sibling.Childs[1] != nil {
					sibling.Childs[1].Color = Black
				}
				tree.rotate(parent, 0)
				node = tree.Root
				parent = nil
			}
		} else {
			sibling := parent.Childs[0]
			if sibling != nil && sibling.Color == Red {
				sibling.Color = Black
				parent.Color = Red
				tree.rotate(parent, 1)
				sibling = parent.Childs[0]
			}
			if sibling == nil {
				node = parent
				parent = node.Parent
				continue
			}
			if isBlack(sibling.Childs[0]) && isBlack(sibling.Childs[1]) {
				sibling.Color = Red
				node = parent
				parent = node.Parent
			} else {
				if isBlack(sibling.Childs[0]) {
					if sibling.Childs[1] != nil {
						sibling.Childs[1].Color = Black
					}
					sibling.Color = Red
					tree.rotate(sibling, 0)
					sibling = parent.Childs[0]
				}
				sibling.Color = parent.Color
				parent.Color = Black
				if sibling.Childs[0] != nil {
					sibling.Childs[0].Color = Black
				}
				tree.rotate(parent, 1)
				node = tree.Root
				parent = nil
			}
		}
	}
	if node != nil {
		node.Color = Black
	}
	if tree.Root != nil {
		tree.Root.Color = Black
	}
}
