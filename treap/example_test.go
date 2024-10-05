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

var controller = &treap.Controller[nodeStruct]{
	Push: func(n *treap.Node[nodeStruct]) {

	},
	Pull: func(n *treap.Node[nodeStruct]) {
		n.Sz = 1 + n.C[0].Size() + n.C[1].Size()
	},
	Less: func(x, y *nodeStruct) bool {
		return x.value < y.value
	},
}

func TestTreap(t *testing.T) {
	node5 := treap.NewNode(nodeStruct{5})
	node7 := treap.NewNode(nodeStruct{10})
	node57 := controller.Merge(node5, node7)
	arr := controller.Array(node57)
	str := fmt.Sprint(arr)
	assert.Equal(t, "[{5} {10}]", str)
}
