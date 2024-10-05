package segment_tree

type node[value, update any] struct {
	value  value
	update update
}

type ST[value, update any] struct {
	size        int
	arr         []node[value, update]
	zeroUpdate  update
	merge       func(*value, *value) value
	applyUpdate func(*update, *value)
	push        func(top *update, being_updated *update)
}

func NewST[value any](
	init func(int) value,
	size int,
	merge func(*value, *value) value,
) *ST[value, struct{}] {
	return NewLazyST(
		init,
		size,
		struct{}{},
		merge,
		func(*struct{}, *value) {},
		func(*struct{}, *struct{}) {},
	)
}

func NewLazyST[value, update any](
	init func(int) value,
	size int,
	zeroUpdate update,
	merge func(*value, *value) value,
	applyUpdate func(*update, *value),
	push func(top *update, being_updated *update),
) *ST[value, update] {
	if size <= 0 {
		panic("size not positive")
	}
	st := ST[value, update]{
		size:        size,
		merge:       merge,
		zeroUpdate:  zeroUpdate,
		applyUpdate: applyUpdate,
		push:        push,
	}
	var rec func(_, _, _ int)
	rec = func(i, l, r int) {
		if i >= len(st.arr) {
			st.arr = append(st.arr, make([]node[value, update], i-len(st.arr)+1)...)
		}
		if l == r {
			st.arr[i] = node[value, update]{
				init(l),
				zeroUpdate,
			}
		} else {
			mid := (l + r) / 2
			rec(2*i+1, l, mid)
			rec(2*i+2, mid+1, r)
			st.arr[i] = node[value, update]{
				merge(&st.arr[2*i+1].value, &st.arr[2*i+2].value),
				zeroUpdate,
			}
		}
	}
	rec(0, 0, size-1)
	return &st
}

func (st *ST[value, update]) fix(i, l, r int) {
	node := &st.arr[i]
	if l != r {
		st.push(&node.update, &st.arr[2*i+1].update)
		st.push(&node.update, &st.arr[2*i+2].update)
	}
	st.applyUpdate(&node.update, &node.value)
	node.update = st.zeroUpdate
}

func (st *ST[value, update]) Get(x, y int) value {
	x = max(x, 0)
	y = min(y, st.size-1)
	if x > y {
		panic("invalid range")
	}
	i := 0
	l := 0
	r := st.size - 1
	for {
		st.fix(i, l, r)
		if l == r {
			return st.arr[i].value
		}
		mid := (l + r) / 2
		if y <= mid {
			i = 2*i + 1
			r = mid
		} else if x > mid {
			i = 2*i + 2
			l = mid + 1
		} else {
			break
		}
	}
	var ret value
	first := true
	var rec func(_, _, _ int)
	rec = func(i, l, r int) {
		if y < l || r < x {
			return
		}
		st.fix(i, l, r)
		if x <= l && r <= y {
			valPtr := &st.arr[i].value
			if first {
				first = false
				ret = *valPtr
			} else {
				ret = st.merge(&ret, valPtr)
			}
			return
		}
		mid := (l + r) / 2
		rec(2*i+1, l, mid)
		rec(2*i+2, mid+1, r)
	}
	mid := (l + r) / 2
	l_ := l
	r_ := mid
	for j := 2*i + 1; ; {
		st.fix(j, l_, r_)
		if x <= l_ {
			valPtr := &st.arr[j].value
			if first {
				first = false
				ret = *valPtr
			} else {
				ret = st.merge(valPtr, &ret)
			}
			break
		}
		mid_ := (l_ + r_) / 2
		if x <= mid_ {
			st.fix(2*j+2, mid_+1, r_)
			valPtr := &st.arr[2*j+2].value
			if first {
				first = false
				ret = *valPtr
			} else {
				ret = st.merge(valPtr, &ret)
			}
			j = 2*j + 1
			r_ = mid_
		} else {
			j = 2*j + 2
			l_ = mid_ + 1
		}
	}
	rec(2*i+2, mid+1, r)
	return ret
}

func (st *ST[value, update]) Set(x int, v value) {
	if x < 0 || x >= st.size {
		panic("invalid x")
	}
	var rec func(_, _, _ int)
	rec = func(i, l, r int) {
		st.fix(i, l, r)
		if l == r {
			st.arr[i] = node[value, update]{
				v,
				st.zeroUpdate,
			}
			st.fix(i, l, r)
			return
		}
		mid := (l + r) / 2
		if x <= mid {
			rec(2*i+1, l, mid)
			st.fix(2*i+2, mid+1, r)
		} else {
			st.fix(2*i+1, l, mid)
			rec(2*i+2, mid+1, r)
		}
		st.arr[i].value = st.merge(&st.arr[2*i+1].value, &st.arr[2*i+2].value)
	}
	rec(0, 0, st.size-1)
}

func (st *ST[value, update]) Update(x, y int, v update) {
	var rec func(_, _, _ int)
	rec = func(i, l, r int) {
		st.fix(i, l, r)
		if y < l || r < x {
			return
		}
		if x <= l && r <= y {
			st.push(&v, &st.arr[i].update)
			st.fix(i, l, r)
			return
		}
		mid := (l + r) / 2
		rec(2*i+1, l, mid)
		rec(2*i+2, mid+1, r)
		st.arr[i].value = st.merge(&st.arr[2*i+1].value, &st.arr[2*i+2].value)
	}
	rec(0, 0, st.size-1)
}

func (st *ST[value, update]) GetArray() []value {
	arr := make([]value, st.size)
	var rec func(_, _, _ int)
	rec = func(i, l, r int) {
		st.fix(i, l, r)
		if l == r {
			arr[l] = st.arr[i].value
			return
		}
		mid := (l + r) / 2
		rec(2*i+1, l, mid)
		rec(2*i+2, mid+1, r)
	}
	rec(0, 0, st.size-1)
	return arr
}
