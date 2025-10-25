package main

import (
	"fmt"
	"math/bits"
	"unsafe"
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
	log2n := LogCeil(uint64(size))
	n := 1 << log2n
	arr := make([]segmentTreeNode[value, update], 2*n)
	for i := range size {
		*Get(arr, n+i) = segmentTreeNode[value, update]{init(i), zeroUpdate}
	}
	for i := size; i < n; i++ {
		*Get(arr, n+i) = segmentTreeNode[value, update]{zeroValue, zeroUpdate}
	}
	for i := n - 1; i > 0; i-- {
		*Get(arr, i) = segmentTreeNode[value, update]{(Get(arr, 2*i).value).Merge(Get(arr, 2*i+1).value), zeroUpdate}
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
	(*Get(st.arr, i+st.n)).value = val
	(*Get(st.arr, i+st.n)).update = st.zeroUpdate
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
				(upd).Push(&(*Get(st.arr, l)).update)
				l = l/2 + 1
			} else {
				l = l / 2
			}
			if r%2 == 0 {
				(upd).Push(&(*Get(st.arr, r)).update)
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
			(Get(st.arr, l).update).ApplyUpdate(&Get(st.arr, l).value)
			summedL = summedL.Merge((*Get(st.arr, l)).value)
			l = l/2 + 1
		} else {
			l = l / 2
		}
		if r%2 == 0 {
			(Get(st.arr, r).update).ApplyUpdate(&Get(st.arr, r).value)
			summedR = Get(st.arr, r).value.Merge(summedR)
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
	if !predicate(Get(st.arr, i).value) {
		return -1, false
	}
	summedValue := st.zeroValue
	for i > 0 {
		arrVal := Get(st.arr, i)
		(arrVal.update).ApplyUpdate(&arrVal.value)
		val := arrVal.value.Merge(summedValue)
		if predicate(val) {
			if IsPowerOf2G1int(i) {
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

	upd := Get(st.arr, i).update
	for i < st.n {
		val := Get(st.arr, 2*i+1)
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
		update := Get(st.arr, i).update

		Get(st.arr, i).update = st.zeroUpdate
		(update).ApplyUpdate(&Get(st.arr, i).value)

		(update).Push(&Get(st.arr, 2*i).update)
		(update).Push(&Get(st.arr, 2*i+1).update)
	}
	(Get(st.arr, i).update).ApplyUpdate(&Get(st.arr, i).value)
	Get(st.arr, i).update = st.zeroUpdate
}

func (st SegmentTree[value, update]) rebuild(
	i int,
) {
	i += st.n
	i /= 2
	for i != 0 {
		left := Get(st.arr, 2*i)
		(left.update).ApplyUpdate(&left.value)
		right := Get(st.arr, 2*i+1)
		(right.update).ApplyUpdate(&right.value)
		Get(st.arr, i).value = left.value.Merge(right.value)
		i = i / 2
	}
}

type stValue struct {
	minA  int
	minDP int
}

type lazy struct {
	addDP int
}

func (a stValue) Merge(b stValue) stValue {
	return stValue{minA: min(a.minA, b.minA), minDP: min(a.minDP, b.minDP)}
}

func (up lazy) ApplyUpdate(val *stValue) {
	val.minDP += up.addDP
}

func (top lazy) Push(existing *lazy) {
	existing.addDP += top.addDP
}

func main() {
	var s SegmentTreeG1stValueG2lazy
	fmt.Println(s)
}

func Get[T any](slice []T, index int) *T {
	return (*T)(unsafe.Pointer(uintptr(unsafe.Pointer(unsafe.SliceData(slice))) + uintptr(index)*unsafe.Sizeof(*new(T))))
}

func GetArr[T any](slice []T, offset int) *[8]T {
	data := unsafe.Add(unsafe.Pointer(unsafe.SliceData(slice)), uintptr(offset)*unsafe.Sizeof(*new(T)))
	return (*[8]T)(data)
}

func LogCeil(x uint64) int {
	if x == 0 {
		panic("Log2(0) is undefined")
	}
	if x == 1 {
		return 0
	}
	return 1 + LogFloor(x-1)
}

func IsPowerOf2[T int | uint64](x T) bool {
	x64 := uint64(x)
	return x64 != 0 && (x64&(x64-1)) == 0
}

func LogFloor(x uint64) int {
	if x == 0 {
		panic("Log2(0) is undefined")
	}
	return 63 - bits.LeadingZeros64(x)
}
func IsPowerOf2G1int(x int) bool {

	x64 := uint64(x)
	return x64 != 0 && (x64&(x64-1)) ==
		0
}

type SegmentTreeG1stValueG2lazy struct {
	log2n int
	n     int
	arr   []segmentTreeNodeG1stValueG2lazy

	zeroValue stValue

	zeroUpdate lazy
}

func (st SegmentTreeG1stValueG2lazy,

) SetValue(i int, val stValue) {
	if i < 0 || i >= st.n {
		panic("index out of bounds")
	}

	st.pushUpdates(i)
	(*GetG1segmentTreeNodeOfstValueClazy(st.arr, i+st.n)).value = val
	(*GetG1segmentTreeNodeOfstValueClazy(st.arr, i+st.n)).
		update = st.zeroUpdate
	st.rebuild(i)
}
func (st SegmentTreeG1stValueG2lazy,

) Update(l,
	r int, upd lazy) {
	l = max(l, 0)
	r = min(r,
		st.n-1)
	if l >
		r {
		return

	}
	st.pushUpdates(l)
	st.pushUpdates(r)
	{
		l, r := l+st.n,
			r+st.n
		for l <=
			r {
			if l%2 == 1 {
				(upd).Push(&(*GetG1segmentTreeNodeOfstValueClazy(st.arr, l)).
					update)
				l = l/2 + 1
			} else {
				l = l / 2
			}
			if r%2 ==
				0 {
				(upd).Push(&(*GetG1segmentTreeNodeOfstValueClazy(st.arr, r)).update)
				r = r/2 -
					1
			} else {
				r =
					r / 2
			}
		}
	}
	st.
		rebuild(l)
	st.rebuild(r)
}
func (st SegmentTreeG1stValueG2lazy,

) Get(l,

	r int) stValue {
	l = max(l, 0)
	r = min(r, st.n-1)
	if l > r {
		return st.zeroValue

	}
	st.pushUpdates(l)
	st.pushUpdates(r)
	summedL := st.
		zeroValue
	summedR :=
		st.zeroValue
	l, r =
		l+st.n, r+st.n
	for l <= r {
		if l%2 == 1 {
			(GetG1segmentTreeNodeOfstValueClazy(st.arr, l).update).ApplyUpdate(&GetG1segmentTreeNodeOfstValueClazy(st.arr,
				l).value)
			summedL = summedL.Merge((*GetG1segmentTreeNodeOfstValueClazy(st.
				arr, l)).value)
			l = l/2 + 1
		} else {
			l = l / 2
		}
		if r%2 == 0 {
			(GetG1segmentTreeNodeOfstValueClazy(st.arr, r).update).ApplyUpdate(&GetG1segmentTreeNodeOfstValueClazy(
				st.arr, r).value)
			summedR = GetG1segmentTreeNodeOfstValueClazy(st.
				arr,
				r).
				value.Merge(summedR)
			r = r/2 -
				1
		} else {
			r = r / 2
		}
	}
	return summedL.Merge(summedR)
}
func (st SegmentTreeG1stValueG2lazy,

) LongestRangeWherePredicate(rMax int, predicate func(stValue) bool) (int,
	bool) {
	if rMax <
		0 || rMax >= st.n {
		panic("invalid rMax")
	}
	st.pushUpdates(rMax)
	i := rMax +
		st.n
	if !predicate(GetG1segmentTreeNodeOfstValueClazy(st.arr, i).value) {
		return -1, false
	}
	summedValue := st.zeroValue
	for i > 0 {
		arrVal := GetG1segmentTreeNodeOfstValueClazy(st.arr, i)
		(arrVal.update).ApplyUpdate(&arrVal.value)
		val := arrVal.
			value.
			Merge(summedValue)
		if predicate(val) {
			if IsPowerOf2G1int(i) {
				return 0, true
			}
			if i%
				2 == 0 {
				i =
					i/2 -
						1
				summedValue = val.
					Merge(summedValue)
			} else {
				i = i / 2
			}
		} else {
			break
		}
	}
	upd := GetG1segmentTreeNodeOfstValueClazy(st.arr,
		i).update
	for i < st.n {
		val :=
			GetG1segmentTreeNodeOfstValueClazy(st.arr, 2*i+1)
		(upd).Push(
			&val.update)
		(upd).
			ApplyUpdate(&val.value)
		combined := val.value.
			Merge(summedValue)
		if predicate(combined) {
			summedValue = combined
			i = 2 * i
		} else {
			upd = val.update
			i = 2*i +
				1
		}
	}
	return i - st.n + 1, true
}
func (st SegmentTreeG1stValueG2lazy,

) pushUpdates(i int) {
	i += st.n
	for shift := st.log2n; shift >
		0; shift-- {
		i := i >>

			shift
		lazy :=
			GetG1segmentTreeNodeOfstValueClazy(st.arr, i).update
		GetG1segmentTreeNodeOfstValueClazy(st.arr, i).update = st.zeroUpdate
		(lazy).
			ApplyUpdate(&GetG1segmentTreeNodeOfstValueClazy(st.arr,
				i).value,
			)
		(lazy).
			Push(&GetG1segmentTreeNodeOfstValueClazy(
				st.arr, 2*i).update)
		(lazy).
			Push(&GetG1segmentTreeNodeOfstValueClazy(st.arr,
				2*i+1).update)
	}
	(GetG1segmentTreeNodeOfstValueClazy(st.arr, i).update).ApplyUpdate(&GetG1segmentTreeNodeOfstValueClazy(st.
		arr, i).value)
	GetG1segmentTreeNodeOfstValueClazy(st.
		arr, i).update = st.
		zeroUpdate
}
func (st SegmentTreeG1stValueG2lazy,

) rebuild(i int) {
	i += st.n
	i /= 2
	for i != 0 {
		left := GetG1segmentTreeNodeOfstValueClazy(st.arr,
			2*i)
		(left.update).
			ApplyUpdate(&left.
				value)
		right := GetG1segmentTreeNodeOfstValueClazy(st.arr, 2*i+1)
		(right.update).ApplyUpdate(&right.value)
		GetG1segmentTreeNodeOfstValueClazy(st.arr,
			i).value = left.
			value.Merge(right.
			value)
		i = i / 2
	}
}
func GetG1segmentTreeNodeOfstValueClazy(slice []segmentTreeNodeG1stValueG2lazy,
	index int) *segmentTreeNodeG1stValueG2lazy {

	return (*segmentTreeNodeG1stValueG2lazy)(unsafe.Pointer(uintptr(unsafe.Pointer(
		unsafe.SliceData(slice))) + uintptr(index)*unsafe.
		Sizeof(*new(segmentTreeNodeG1stValueG2lazy))))
}

type segmentTreeNodeG1stValueG2lazy struct {
	value stValue

	update lazy
}
