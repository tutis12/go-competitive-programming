package treap

import "math/rand/v2"

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
	if x == nil {
		return nil, false
	}
	c.Push(x)
	sz0 := x.C[0].Size()
	if sz0 == i {
		return x, true
	}
	if i < sz0 {
		return c.GetI(x.C[0], i)
	} else {
		return c.GetI(x.C[1], i-1-sz0)
	}
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
		xx := *x
		xx.C[1] = c.Merge(x.C[1], y)
		c.Pull(&xx)
		return &xx
	} else {
		c.Push(y)
		yy := *y
		yy.C[0] = c.Merge(x, y.C[0])
		c.Pull(&yy)
		return &yy
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
