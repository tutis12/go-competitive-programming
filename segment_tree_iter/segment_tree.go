package segment_tree_iter

import (
	"main/utils"
)

type segmentTreeNode[value, update any] struct {
	value  value
	update update
}

type Controller[value any, update any] interface {
	~struct{}
	Merge(value, value) value
	ApplyUpdate(update, *value)
	Push(update, *update)
	ZeroValue() value
	ZeroUpdate() update
	Predicate(value) bool
}

type SegmentTree[
	value any,
	update any,
	controller Controller[value, update],
] struct {
	log2n int
	n     int // power of two, number of leaves
	arr   []segmentTreeNode[value, update]
}

func NewSegmentTree[
	value any,
	update any,
	controller Controller[value, update],
](
	init func(int) value,
	size int,
) *SegmentTree[value, update, controller] {
	if size <= 0 {
		panic("size must be positive")
	}

	log2n := utils.LogCeil(uint64(size))
	n := 1 << log2n
	arr := make([]segmentTreeNode[value, update], 2*n)
	for i := range size {
		*utils.Get(arr, n+i) = segmentTreeNode[value, update]{
			value:  init(i),
			update: controller{}.ZeroUpdate(),
		}
	}
	for i := size; i < n; i++ {
		*utils.Get(arr, n+i) = segmentTreeNode[value, update]{
			value:  controller{}.ZeroValue(),
			update: controller{}.ZeroUpdate(),
		}
	}
	for i := n - 1; i > 0; i-- {
		*utils.Get(arr, i) = segmentTreeNode[value, update]{
			value:  controller{}.Merge((utils.Get(arr, 2*i).value), (utils.Get(arr, 2*i+1).value)),
			update: controller{}.ZeroUpdate(),
		}
	}
	return &SegmentTree[value, update, controller]{
		log2n: log2n,
		n:     n,
		arr:   arr,
	}
}

/*
n = 16 log2n = 4

0 |  1  1  1  1  1  1  1  1  1  1  1  1  1  1  1  1
1 |  2  2  2  2  2  2  2  2  3  3  3  3  3  3  3  3
2 |  4  4  4  4  5  5  5  5  6  6  6  6  7  7  7  7
3 |  8  8  9  9 10 10 11 11 12 12 13 13 14 14 15 15
4 | 16 17 18 19 20 21 22 23 24 25 26 27 28 29 30 31

	[0 ... val(i) ... size-1] [size ... zero ... n]
*/

func (st *SegmentTree[value, update, controller]) SetValue(
	i int,
	val value,
) {
	if i < 0 || i >= st.n {
		panic("index out of bounds")
	}
	st.pushUpdates(i)
	valI := utils.Get(st.arr, i+st.n)
	valI.value = val
	valI.update = controller{}.ZeroUpdate()
	st.rebuild(i)
}

func (st *SegmentTree[value, update, controller]) Update(
	l, r int,
	upd update,
) {
	l = max(l, 0)
	r = min(r, st.n-1)
	if l > r {
		return
	}

	st.pushUpdates(l)
	st.pushUpdates(r)
	{
		l, r := l+st.n, r+st.n
		for l <= r {
			if l%2 == 1 {
				controller{}.Push(upd, &(*utils.Get(st.arr, l)).update)
				l = l/2 + 1
			} else {
				l = l / 2
			}
			if r%2 == 0 {
				controller{}.Push(upd, &(*utils.Get(st.arr, r)).update)
				r = r/2 - 1
			} else {
				r = r / 2
			}
		}
	}
	st.rebuild(l)
	st.rebuild(r)
}

func (st *SegmentTree[value, update, controller]) Get(
	l, r int,
) value {
	l = max(l, 0)
	r = min(r, st.n-1)
	if l > r {
		return controller{}.ZeroValue()
	}
	st.pushUpdates(l)
	st.pushUpdates(r)
	summedL := controller{}.ZeroValue()
	summedR := controller{}.ZeroValue()
	l, r = l+st.n, r+st.n
	for l <= r {
		if l%2 == 1 {
			valL := utils.Get(st.arr, l)
			controller{}.ApplyUpdate(valL.update, &valL.value)
			summedL = controller{}.Merge(summedL, valL.value)
			l = l/2 + 1
		} else {
			l = l / 2
		}
		if r%2 == 0 {
			valR := utils.Get(st.arr, r)
			controller{}.ApplyUpdate(valR.update, &valR.value)
			summedR = controller{}.Merge(valR.value, summedR)
			r = r/2 - 1
		} else {
			r = r / 2
		}
	}
	return controller{}.Merge(summedL, summedR)
}

func (st *SegmentTree[value, update, controller]) LongestRangeWherePredicate(
	rMax int,
) (int, bool) {
	if rMax < 0 || rMax >= st.n {
		panic("invalid rMax")
	}

	st.pushUpdates(rMax)
	i := rMax + st.n
	if !(controller{}).Predicate(utils.Get(st.arr, i).value) {
		return -1, false
	}
	summedValue := controller{}.ZeroValue()
	for i > 0 {
		arrVal := *utils.Get(st.arr, i)
		controller{}.ApplyUpdate(arrVal.update, &arrVal.value)
		val := controller{}.Merge(arrVal.value, summedValue)
		if (controller{}).Predicate(val) {
			if utils.IsPowerOf2(i) {
				return 0, true
			}
			if i%2 == 0 {
				i = i/2 - 1
				summedValue = controller{}.Merge(val, summedValue)
			} else {
				i = i / 2
			}
		} else {
			break
		}
	}

	upd := utils.Get(st.arr, i).update
	for i < st.n {
		val := *utils.Get(st.arr, 2*i+1)
		controller{}.Push(upd, &val.update)
		controller{}.ApplyUpdate(upd, &val.value)
		combined := controller{}.Merge(val.value, summedValue)
		if (controller{}).Predicate(combined) {
			summedValue = combined
			i = 2 * i
		} else {
			upd = val.update
			i = 2*i + 1
		}
	}
	return i - st.n + 1, true
}

func (st *SegmentTree[value, update, controller]) pushUpdates(
	i int,
) {
	i += st.n
	for shift := st.log2n; shift > 0; shift-- {
		i := i >> shift
		update := utils.Get(st.arr, i).update

		utils.Get(st.arr, i).update = controller{}.ZeroUpdate()
		controller{}.ApplyUpdate(update, &utils.Get(st.arr, i).value)

		controller{}.Push(update, &utils.Get(st.arr, 2*i).update)
		controller{}.Push(update, &utils.Get(st.arr, 2*i+1).update)
	}
	valI := utils.Get(st.arr, i)
	controller{}.ApplyUpdate(valI.update, &valI.value)
	valI.update = controller{}.ZeroUpdate()
}

func (st *SegmentTree[value, update, controller]) rebuild(
	i int,
) {
	i += st.n
	i /= 2
	for i != 0 {
		left := *utils.Get(st.arr, 2*i)
		controller{}.ApplyUpdate(left.update, &left.value)
		right := *utils.Get(st.arr, 2*i+1)
		controller{}.ApplyUpdate(right.update, &right.value)
		utils.Get(st.arr, i).value = controller{}.Merge(left.value, right.value)
		i = i / 2
	}
}
