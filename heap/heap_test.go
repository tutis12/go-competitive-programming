package heap

import "testing"

func TestCorrectness(t *testing.T) {
	h := NewMinHeap(func(a, b int) bool {
		return a < b
	})

	h.Push(5)
	h.Push(3)
	h.Push(8)
	h.Push(1)

	expectedOrder := []int{1, 3, 5, 8}
	for _, expected := range expectedOrder {
		actual := h.Pop()
		if actual != expected {
			t.Errorf("Expected %d, but got %d", expected, actual)
		}
	}

	if h.Len() != 0 {
		t.Errorf("Expected heap length to be 0, but got %d", h.Len())
	}
}
