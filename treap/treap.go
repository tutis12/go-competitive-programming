package treap

import (
	"math/rand/v2"
)

type Controller[V any] interface {
	~struct{}
	Push(*Node[V])
	Pull(*Node[V])
	Less(*V, *V) bool
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

func GetI[V any, C Controller[V]](x *Node[V], i int) (*Node[V], bool) {
	for {
		if x == nil {
			return nil, false
		}
		C{}.Push(x)
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

func Contains[V any, C Controller[V]](x *Node[V], ctx *V) bool {
	var lastR *Node[V]
	for x != nil {
		C{}.Push(x)
		if (C{}).Less(ctx, &x.Value) {
			x = x.C[0]
		} else {
			lastR = x
			x = x.C[1]
		}
	}
	if lastR != nil && !(C{}).Less(&lastR.Value, ctx) {
		return true
	}
	return false
}

func InsertI[V any, C Controller[V]](
	root *Node[V],
	v V,
	i int,
) *Node[V] {
	left, right := SplitK[V, C](root, i)
	return Merge[V, C](Merge[V, C](left, NewNode(v)), right)
}

func Insert[V any, C Controller[V]](
	root *Node[V],
	v V,
	allowRepetition bool,
) *Node[V] {
	left, right := Split[V, C](root, &v)

	if !allowRepetition {
		maybeSame := Last[V, C](left) // <=v
		if maybeSame != nil && !(C{}).Less(&maybeSame.Value, &v) {
			return Merge[V, C](left, right)
		}
	}
	return Merge[V, C](Merge[V, C](left, NewNode(v)), right)
}

func Merge[V any, C Controller[V]](x, y *Node[V]) *Node[V] {
	if x == nil && y == nil {
		return nil
	}
	if x == nil {
		C{}.Push(y)
		return y
	}
	if y == nil {
		C{}.Push(x)
		return x
	}
	if x.Seed > y.Seed {
		C{}.Push(x)
		x.C[1] = Merge[V, C](x.C[1], y)
		C{}.Pull(x)
		return x
	} else {
		C{}.Push(y)
		y.C[0] = Merge[V, C](x, y.C[0])
		C{}.Pull(y)
		return y
	}
}

// (first k, else)
func SplitK[V any, C Controller[V]](x *Node[V], k int) (*Node[V], *Node[V]) {
	if x == nil {
		return nil, nil
	}
	C{}.Push(x)
	sz0 := x.C[0].Size()
	if sz0 >= k {
		a, b := SplitK[V, C](x.C[0], k)
		x.C[0] = b
		C{}.Pull(x)
		return a, x
	} else {
		a, b := SplitK[V, C](x.C[1], k-1-sz0)
		x.C[1] = a
		C{}.Pull(x)
		return x, b
	}
}

// (<=ctx, >ctx)
func Split[V any, C Controller[V]](x *Node[V], ctx *V) (*Node[V], *Node[V]) {
	if x == nil {
		return nil, nil
	}
	C{}.Push(x)
	if (C{}).Less(ctx, &x.Value) {
		a, b := Split[V, C](x.C[0], ctx)
		x.C[0] = b
		C{}.Pull(x)
		return a, x
	} else {
		a, b := Split[V, C](x.C[1], ctx)
		x.C[1] = a
		C{}.Pull(x)
		return x, b
	}
}

func Array[V any, C Controller[V]](x *Node[V]) []V {
	if x == nil {
		return nil
	}
	C{}.Push(x)
	arr := Array[V, C](x.C[0])
	arr = append(arr, x.Value)
	arr = append(arr, Array[V, C](x.C[1])...)
	return arr
}

func Last[V any, C Controller[V]](n *Node[V]) *Node[V] {
	if n == nil {
		return nil
	}
	for n.C[1] != nil {
		C{}.Push(n)
		n = n.C[1]
	}
	C{}.Push(n)
	return n
}

func First[V any, C Controller[V]](n *Node[V]) *Node[V] {
	if n == nil {
		return nil
	}
	for n.C[0] != nil {
		C{}.Push(n)
		n = n.C[0]
	}
	C{}.Push(n)
	return n
}

func GetValue[V any, C Controller[V]](root *Node[V]) (V, bool) {
	if root == nil {
		var zero V
		return zero, false
	}
	C{}.Push(root)
	return root.Value, true
}
