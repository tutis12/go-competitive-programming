package treap

import (
	"math/rand/v2"
)

type Controller[V any] struct {
	Push func(*Node[V])
	Pull func(*Node[V])
	Less func(*V, *V) bool
}

type Node[V any] struct {
	C     [2]*Node[V]
	Value V
	Sz    int
	Seed  uint32
}

func NewNode[V any](value V) *Node[V] {
	return &Node[V]{
		Value: value,
		Sz:    1,
		Seed:  rand.Uint32(),
	}
}

func (n *Node[V]) Size() int {
	if n == nil {
		return 0
	}
	return n.Sz
}

func (c *Controller[V]) GetI(x *Node[V], i int) (*Node[V], bool) {
	for {
		if x == nil {
			return nil, false
		}
		c.Push(x)
		sz0 := x.C[0].Size()
		if sz0 == i {
			return x, true
		}
		if i < sz0 {
			x = x.C[0]
		} else {
			x, i = x.C[1], i-1-sz0
		}
	}
}

func (c *Controller[V]) Contains(x *Node[V], ctx *V) bool {
	for x != nil {
		c.Push(x)
		if c.Less(ctx, &x.Value) {
			x = x.C[0]
		} else {
			if !c.Less(&x.Value, ctx) {
				return true
			}
			x = x.C[1]
		}
	}
	return false
}

func (c *Controller[Value]) InsertI(
	root *Node[Value],
	v Value,
	i int,
) *Node[Value] {
	left, right := c.SplitK(root, i)
	return c.Merge(c.Merge(left, NewNode(v)), right)
}

func (c *Controller[Value]) Insert(
	root *Node[Value],
	v Value,
	allowRepetition bool,
) *Node[Value] {
	left, right := c.Split(root, &v)

	if !allowRepetition {
		maybeSame := c.Last(left) // <=v
		if maybeSame != nil && !c.Less(&maybeSame.Value, &v) {
			return c.Merge(left, right)
		}
	}
	return c.Merge(c.Merge(left, NewNode(v)), right)
}

func (c *Controller[V]) Merge(x, y *Node[V]) *Node[V] {
	if x == nil && y == nil {
		return nil
	}
	if x == nil {
		c.Push(y)
		return y
	}
	if y == nil {
		c.Push(x)
		return x
	}
	if x.Seed > y.Seed {
		c.Push(x)
		x.C[1] = c.Merge(x.C[1], y)
		c.Pull(x)
		return x
	} else {
		c.Push(y)
		y.C[0] = c.Merge(x, y.C[0])
		c.Pull(y)
		return y
	}
}

// (first k, else)
func (c *Controller[V]) SplitK(x *Node[V], k int) (*Node[V], *Node[V]) {
	if x == nil {
		return nil, nil
	}
	c.Push(x)
	sz0 := x.C[0].Size()
	if sz0 >= k {
		a, b := c.SplitK(x.C[0], k)
		x.C[0] = b
		c.Pull(x)
		return a, x
	} else {
		a, b := c.SplitK(x.C[1], k-1-sz0)
		x.C[1] = a
		c.Pull(x)
		return x, b
	}
}

// (<=ctx, >ctx)
func (c *Controller[V]) Split(x *Node[V], ctx *V) (*Node[V], *Node[V]) {
	if x == nil {
		return nil, nil
	}
	c.Push(x)
	if c.Less(ctx, &x.Value) {
		a, b := c.Split(x.C[0], ctx)
		x.C[0] = b
		c.Pull(x)
		return a, x
	} else {
		a, b := c.Split(x.C[1], ctx)
		x.C[1] = a
		c.Pull(x)
		return x, b
	}
}

func (c *Controller[V]) Array(x *Node[V]) []V {
	if x == nil {
		return nil
	}
	c.Push(x)
	arr := c.Array(x.C[0])
	arr = append(arr, x.Value)
	arr = append(arr, c.Array(x.C[1])...)
	return arr
}

func (c *Controller[V]) Last(n *Node[V]) *Node[V] {
	if n == nil {
		return nil
	}
	for n.C[1] != nil {
		c.Push(n)
		n = n.C[1]
	}
	c.Push(n)
	return n
}

func (c *Controller[V]) First(n *Node[V]) *Node[V] {
	if n == nil {
		return nil
	}
	for n.C[0] != nil {
		c.Push(n)
		n = n.C[0]
	}
	c.Push(n)
	return n
}

func (c *Controller[V]) GetValue(root *Node[V]) (V, bool) {
	if root == nil {
		var zero V
		return zero, false
	}
	c.Push(root)
	return root.Value, true
}
