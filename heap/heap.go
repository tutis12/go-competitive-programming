package heap

type MinHeap[T any] struct {
	data []T
	less func(a, b T) bool
}

func NewMinHeap[T any](
	less func(a, b T) bool,
) *MinHeap[T] {
	return &MinHeap[T]{
		data: make([]T, 0),
		less: less,
	}
}

func (h *MinHeap[T]) Len() int {
	return len(h.data)
}

func (h *MinHeap[T]) Push(value T) {
	h.data = append(h.data, value)
	h.up(h.Len() - 1)
}

func (h *MinHeap[T]) Pop() T {
	n := h.Len() - 1
	h.swap(0, n)
	h.down(0, n)

	value := h.data[n]
	h.data = h.data[:n]
	return value
}

func (h *MinHeap[T]) Peek() T {
	return h.data[0]
}

func (h *MinHeap[T]) swap(i, j int) {
	h.data[i], h.data[j] = h.data[j], h.data[i]
}

func (h *MinHeap[T]) up(j int) {
	for j != 0 {
		i := (j - 1) / 2 // parent
		if !h.less(h.data[j], h.data[i]) {
			break
		}
		h.swap(i, j)
		j = i
	}
}

func (h *MinHeap[T]) down(i0, n int) {
	i := i0
	for {
		j1 := 2*i + 1
		j2 := 2*i + 2
		if j2 < n && h.less(h.data[j2], h.data[i]) {
			h.swap(i, j2)
			i = j2
			continue
		} else if j1 < n && h.less(h.data[j1], h.data[i]) {
			h.swap(i, j1)
			i = j1
			continue
		} else {
			break
		}
	}
}
