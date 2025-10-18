package segment_tree_iter

import (
	"main/utils"
)

type segmentTreeNode[value, update any] struct {
	value  value
	update update
}

type SegmentTree[
	value interface{ Merge(value) value },
	update interface {
		ApplyUpdate(*value)
		Push(*update)
	},
] struct {
	log2n      int
	n          int // power of two, number of leaves
	arr        []segmentTreeNode[value, update]
	zeroValue  value
	zeroUpdate update
}

func NewSegmentTree[
	value interface{ Merge(value) value },
	update interface {
		ApplyUpdate(*value)
		Push(*update)
	},
](
	init func(int) value,
	size int,
	zeroValue value,
	zeroUpdate update,
) *SegmentTree[value, update] {
	if size <= 0 {
		panic("size must be positive")
	}
	log2n := utils.LogCeil(uint64(size))
	n := 1 << log2n
	arr := make([]segmentTreeNode[value, update], 2*n)
	for i := range size {
		*utils.Get(arr, n+i) = segmentTreeNode[value, update]{init(i), zeroUpdate}
	}
	for i := size; i < n; i++ {
		*utils.Get(arr, n+i) = segmentTreeNode[value, update]{zeroValue, zeroUpdate}
	}
	for i := n - 1; i > 0; i-- {
		*utils.Get(arr, i) = segmentTreeNode[value, update]{(utils.Get(arr, 2*i).value).Merge(utils.Get(arr, 2*i+1).value), zeroUpdate}
	}
	return &SegmentTree[value, update]{
		log2n:      log2n,
		n:          n,
		arr:        arr,
		zeroValue:  zeroValue,
		zeroUpdate: zeroUpdate,
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

func (st SegmentTree[value, update]) SetValue(
	i int,
	val value,
) {
	if i < 0 || i >= st.n {
		panic("index out of bounds")
	}
	st.pushUpdates(i)
	(*utils.Get(st.arr, i+st.n)).value = val
	(*utils.Get(st.arr, i+st.n)).update = st.zeroUpdate
	st.rebuild(i)
}

func (st SegmentTree[value, update]) Update(
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
				(upd).Push(&(*utils.Get(st.arr, l)).update)
				l = l/2 + 1
			} else {
				l = l / 2
			}
			if r%2 == 0 {
				(upd).Push(&(*utils.Get(st.arr, r)).update)
				r = r/2 - 1
			} else {
				r = r / 2
			}
		}
	}
	st.rebuild(l)
	st.rebuild(r)
}

func (st SegmentTree[value, update]) Get(
	l, r int,
) value {
	l = max(l, 0)
	r = min(r, st.n-1)
	if l > r {
		return st.zeroValue
	}
	st.pushUpdates(l)
	st.pushUpdates(r)
	summedL := st.zeroValue
	summedR := st.zeroValue
	l, r = l+st.n, r+st.n
	for l <= r {
		if l%2 == 1 {
			// apply update on-the-fly without copying node back
			(utils.Get(st.arr, l).update).ApplyUpdate(&utils.Get(st.arr, l).value)
			summedL = summedL.Merge((*utils.Get(st.arr, l)).value)
			l = l/2 + 1
		} else {
			l = l / 2
		}
		if r%2 == 0 {
			(utils.Get(st.arr, r).update).ApplyUpdate(&utils.Get(st.arr, r).value)
			summedR = utils.Get(st.arr, r).value.Merge(summedR)
			r = r/2 - 1
		} else {
			r = r / 2
		}
	}
	return summedL.Merge(summedR)
}

func (st SegmentTree[value, update]) LongestRangeWherePredicate(
	rMax int,
	predicate func(value) bool,
) (int, bool) {
	if rMax < 0 || rMax >= st.n {
		panic("invalid rMax")
	}

	st.pushUpdates(rMax)
	i := rMax + st.n
	if !predicate(utils.Get(st.arr, i).value) {
		return -1, false
	}
	summedValue := st.zeroValue
	for i > 0 {
		arrVal := utils.Get(st.arr, i)
		(arrVal.update).ApplyUpdate(&arrVal.value)
		val := arrVal.value.Merge(summedValue)
		if predicate(val) {
			if utils.IsPowerOf2(i) {
				return 0, true
			}
			if i%2 == 0 {
				i = i/2 - 1
				summedValue = val.Merge(summedValue)
			} else {
				i = i / 2
			}
		} else {
			break
		}
	}

	upd := utils.Get(st.arr, i).update
	for i < st.n {
		val := utils.Get(st.arr, 2*i+1)
		(upd).Push(&val.update)
		(upd).ApplyUpdate(&val.value)
		combined := val.value.Merge(summedValue)
		if predicate(combined) {
			summedValue = combined
			i = 2 * i
		} else {
			upd = val.update
			i = 2*i + 1
		}
	}
	return i - st.n + 1, true
}

func (st SegmentTree[value, update]) pushUpdates(
	i int,
) {
	i += st.n
	for shift := st.log2n; shift > 0; shift-- {
		i := i >> shift
		update := utils.Get(st.arr, i).update

		utils.Get(st.arr, i).update = st.zeroUpdate
		(update).ApplyUpdate(&utils.Get(st.arr, i).value)

		(update).Push(&utils.Get(st.arr, 2*i).update)
		(update).Push(&utils.Get(st.arr, 2*i+1).update)
	}
	(utils.Get(st.arr, i).update).ApplyUpdate(&utils.Get(st.arr, i).value)
	utils.Get(st.arr, i).update = st.zeroUpdate
}

func (st SegmentTree[value, update]) rebuild(
	i int,
) {
	i += st.n
	i /= 2
	for i != 0 {
		left := utils.Get(st.arr, 2*i)
		(left.update).ApplyUpdate(&left.value)
		right := utils.Get(st.arr, 2*i+1)
		(right.update).ApplyUpdate(&right.value)
		utils.Get(st.arr, i).value = left.value.Merge(right.value)
		i = i / 2
	}
}
