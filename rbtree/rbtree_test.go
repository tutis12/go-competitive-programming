package rbtree_test

import (
	"main/rbtree"
	"reflect"
	"testing"
)

type rbTreeMockOnSlice[T any] struct {
	slice []T
	less  func(a, b T) bool
}

func newRBTreeMockOnSlice[T any](less func(a, b T) bool) *rbTreeMockOnSlice[T] {
	return &rbTreeMockOnSlice[T]{less: less}
}

func (m *rbTreeMockOnSlice[T]) Insert(value T) {
	idx := m.lowerBound(value)
	m.slice = append(m.slice, value)
	copy(m.slice[idx+1:], m.slice[idx:])
	m.slice[idx] = value
}

func (m *rbTreeMockOnSlice[T]) InsertOrReplace(value T) {
	if idx, ok := m.findEqual(value); ok {
		m.slice[idx] = value
		return
	}
	m.Insert(value)
}

func (m *rbTreeMockOnSlice[T]) Remove(value T) bool {
	if idx, ok := m.findEqual(value); ok {
		m.slice = append(m.slice[:idx], m.slice[idx+1:]...)
		return true
	}
	return false
}

func (m *rbTreeMockOnSlice[T]) Contains(value T) bool {
	_, ok := m.findEqual(value)
	return ok
}

func (m *rbTreeMockOnSlice[T]) Minimum() (T, bool) {
	if len(m.slice) == 0 {
		var zero T
		return zero, false
	}
	return m.slice[0], true
}

func (m *rbTreeMockOnSlice[T]) Maximum() (T, bool) {
	if len(m.slice) == 0 {
		var zero T
		return zero, false
	}
	return m.slice[len(m.slice)-1], true
}

func (m *rbTreeMockOnSlice[T]) Values() []T {
	out := make([]T, len(m.slice))
	copy(out, m.slice)
	return out
}

func (m *rbTreeMockOnSlice[T]) ReverseValues() []T {
	n := len(m.slice)
	out := make([]T, n)
	for i := range m.slice {
		out[n-1-i] = m.slice[i]
	}
	return out
}

func (m *rbTreeMockOnSlice[T]) Clear() {
	m.slice = nil
}

func (m *rbTreeMockOnSlice[T]) IsEmpty() bool {
	return len(m.slice) == 0
}

func (m *rbTreeMockOnSlice[T]) lowerBound(value T) int {
	idx := 0
	for idx < len(m.slice) && m.less(m.slice[idx], value) {
		idx++
	}
	return idx
}

func (m *rbTreeMockOnSlice[T]) findEqual(value T) (int, bool) {
	for idx, cur := range m.slice {
		if !m.less(cur, value) && !m.less(value, cur) {
			return idx, true
		}
	}
	return 0, false
}

func TestRBTreeMatchesSliceModel(t *testing.T) {
	less := func(a, b int) bool { return a < b }
	tree := rbtree.NewRBTree[int](less)
	mock := newRBTreeMockOnSlice[int](less)
	sample := []int{-5, -1, 0, 1, 3, 4, 5, 7, 9, 11}

	for _, v := range []int{5, 3, 7, 3, 9, 1, 5} {
		tree.Insert(v)
		mock.Insert(v)
		verifyState(t, tree, mock, sample)
	}

	for _, v := range []int{3, 5, 7} {
		tree.InsertOrReplace(v)
		mock.InsertOrReplace(v)
		verifyState(t, tree, mock, sample)
	}

	for _, v := range []int{3, 5, 1, 9, 42} {
		if node, ok := tree.Find(v); ok {
			tree.Remove(node)
			if !mock.Remove(v) {
				t.Fatalf("mock failed to remove %d", v)
			}
		} else if mock.Remove(v) {
			t.Fatalf("mock removed missing value %d", v)
		}
		verifyState(t, tree, mock, sample)
	}

	tree.Clear()
	mock.Clear()
	verifyState(t, tree, mock, sample)
}

func TestRBTreeBasicOperations(t *testing.T) {
	less := func(a, b int) bool { return a < b }
	tree := rbtree.NewRBTree[int](less)

	if !tree.IsEmpty() {
		t.Fatal("expected new tree to be empty")
	}

	for _, v := range []int{4, 2, 6, 1, 3, 5, 7} {
		tree.Insert(v)
	}

	tree.InsertOrReplace(4)

	var forward []int
	tree.Iter()(func(v int) bool {
		forward = append(forward, v)
		return true
	})
	expectedForward := []int{1, 2, 3, 4, 5, 6, 7}
	if !reflect.DeepEqual(forward, expectedForward) {
		t.Fatalf("forward traversal mismatch: got %v, want %v", forward, expectedForward)
	}

	var backward []int
	tree.ReverseIter()(func(v int) bool {
		backward = append(backward, v)
		return true
	})
	expectedBackward := []int{7, 6, 5, 4, 3, 2, 1}
	if !reflect.DeepEqual(backward, expectedBackward) {
		t.Fatalf("reverse traversal mismatch: got %v, want %v", backward, expectedBackward)
	}

	min := tree.Minimum(tree.Root)
	if min == nil || min.Value != 1 {
		t.Fatalf("minimum mismatch: got %v", valueOrZero(min))
	}

	max := tree.Maximum(tree.Root)
	if max == nil || max.Value != 7 {
		t.Fatalf("maximum mismatch: got %v", valueOrZero(max))
	}

	if _, ok := tree.Find(4); !ok {
		t.Fatal("expected to find value 4")
	}
	if _, ok := tree.Find(42); ok {
		t.Fatal("did not expect to find value 42")
	}

	for _, v := range []int{4, 6} {
		node, ok := tree.Find(v)
		if !ok {
			t.Fatalf("missing value %d", v)
		}
		tree.Remove(node)
	}

	var afterRemoval []int
	tree.Iter()(func(v int) bool {
		afterRemoval = append(afterRemoval, v)
		return true
	})
	if !reflect.DeepEqual(afterRemoval, []int{1, 2, 3, 5, 7}) {
		t.Fatalf("post-removal traversal mismatch: got %v", afterRemoval)
	}

	tree.Clear()
	if !tree.IsEmpty() || tree.Root != nil {
		t.Fatal("expected tree to be cleared")
	}
}

func verifyState[T comparable](t *testing.T, tree *rbtree.RBTree[T], mock *rbTreeMockOnSlice[T], sample []T) {
	t.Helper()

	forward := make([]T, 0)
	if tree.Root != nil {
		tree.Iter()(func(v T) bool {
			forward = append(forward, v)
			return true
		})
	}
	if !reflect.DeepEqual(forward, mock.Values()) {
		t.Fatalf("forward traversal mismatch: got %v, want %v", forward, mock.Values())
	}

	backward := make([]T, 0)
	if tree.Root != nil {
		tree.ReverseIter()(func(v T) bool {
			backward = append(backward, v)
			return true
		})
	}
	if !reflect.DeepEqual(backward, mock.ReverseValues()) {
		t.Fatalf("reverse traversal mismatch: got %v, want %v", backward, mock.ReverseValues())
	}

	mockMin, hasMin := mock.Minimum()
	if tree.Root == nil {
		if hasMin {
			t.Fatalf("tree empty but mock has min %v", mockMin)
		}
	} else {
		node := tree.Minimum(tree.Root)
		if node == nil || node.Value != mockMin || !hasMin {
			t.Fatalf("minimum mismatch: got node %v (exists %v), want %v", valueOrZero(node), node != nil, mockMin)
		}
	}

	mockMax, hasMax := mock.Maximum()
	if tree.Root == nil {
		if hasMax {
			t.Fatalf("tree empty but mock has max %v", mockMax)
		}
	} else {
		node := tree.Maximum(tree.Root)
		if node == nil || node.Value != mockMax || !hasMax {
			t.Fatalf("maximum mismatch: got node %v (exists %v), want %v", valueOrZero(node), node != nil, mockMax)
		}
	}

	if tree.IsEmpty() != mock.IsEmpty() {
		t.Fatalf("IsEmpty mismatch: tree=%v mock=%v", tree.IsEmpty(), mock.IsEmpty())
	}

	for _, v := range sample {
		_, treeOK := tree.Find(v)
		mockOK := mock.Contains(v)
		if treeOK != mockOK {
			t.Fatalf("Find mismatch for %v: tree=%v mock=%v", v, treeOK, mockOK)
		}
	}
}

func valueOrZero[T comparable](node *rbtree.Node[T]) T {
	if node == nil {
		var zero T
		return zero
	}
	return node.Value
}

func BenchmarkRBTreeInsertDelete(b *testing.B) {
	less := func(a, b int) bool { return a < b }
	benchmarks := []struct {
		name string
		size int
	}{
		{name: "n=32", size: 32},
		{name: "n=256", size: 256},
		{name: "n=1024", size: 1024},
		{name: "n=4096", size: 4096},
	}

	for _, bm := range benchmarks {
		b.Run(bm.name, func(b *testing.B) {
			values := make([]int, bm.size)
			for i := range values {
				values[i] = i
			}

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				tree := rbtree.NewRBTree[int](less)
				for _, v := range values {
					tree.Insert(v)
				}
				for _, v := range values {
					node, ok := tree.Find(v)
					if !ok {
						b.Fatalf("value %d missing", v)
					}
					tree.Remove(node)
				}
			}
		})
	}
}
