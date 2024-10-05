package segment_tree_test

import (
	"main/segment_tree"
	"testing"

	"github.com/test-go/testify/assert"
)

type valueStruct struct {
	value int
}

type updateStruct struct {
	add int
}

func TestLazySegmentTree(t *testing.T) {
	st := segment_tree.NewLazyST(
		func(i int) valueStruct {
			return valueStruct{
				i,
			}
		},
		5,
		updateStruct{
			add: 0,
		},
		func(i1, i2 *valueStruct) valueStruct {
			return valueStruct{
				value: max(i1.value, i2.value),
			}
		},
		func(update *updateStruct, value *valueStruct) {
			value.value += update.add
		},
		func(top, being_updated *updateStruct) {
			being_updated.add += top.add
		},
	)
	st.Update(0, 2, updateStruct{add: 5})
	assert.Equal(t, valueStruct{7}, st.Get(1, 3))
}
