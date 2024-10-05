package slice_map

type KeyValue[K, V any] struct {
	Key   K
	Value V
}

type SliceMap[K comparable, V any] []KeyValue[K, V]

func (m SliceMap[K, V]) Get(key K) (value V, ok bool) {
	for _, kv := range m {
		if kv.Key == key {
			return kv.Value, true
		}
	}
	return value, false
}

func (m *SliceMap[K, V]) Append(key K, value V) {
	*m = append(*m, KeyValue[K, V]{
		Key:   key,
		Value: value,
	})
}
