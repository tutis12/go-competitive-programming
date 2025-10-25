package treap_test

import (
	"fmt"
	"main/treap"
	"testing"

	"github.com/test-go/testify/assert"
)

type nodeStruct struct {
	value int
}

type controller struct{}

func (controller) Push(n *treap.Node[nodeStruct]) {
}

func (controller) Pull(n *treap.Node[nodeStruct]) {
	n.Sz = 1 + n.C[0].Size() + n.C[1].Size()
}

func (controller) Less(x, y *nodeStruct) bool {
	return x.value < y.value
}

func TestTreap(t *testing.T) {
	node5 := treap.NewNode(nodeStruct{5})
	node7 := treap.NewNode(nodeStruct{7})
	node57 := treap.Merge[nodeStruct, controller](node5, node7)
	arr := treap.Array[nodeStruct, controller](node57)
	str := fmt.Sprint(arr)
	assert.Equal(t, "[{5} {7}]", str)
}

func TestTreap2(t *testing.T) {
	node5 := treap.NewNode(nodeStruct{5})
	node7 := treap.NewNode(nodeStruct{7})
	node57 := treap.Merge[nodeStruct, controller](node5, node7)
	node5_, node7_ := treap.Split[nodeStruct, controller](node57, &nodeStruct{5})
	assert.Equal(t, 5, node5_.Value.value)
	assert.Equal(t, 7, node7_.Value.value)
}
