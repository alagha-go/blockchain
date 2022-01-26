package blockchain

import (
	"crypto/sha256"
)


type MarkleTree struct {
	RootNode 					*MarkleNode
}

type MarkleNode struct {
	Left						*MarkleNode
	Right						*MarkleNode
	Data						[]byte
}


func NewMarkleNode(left, right *MarkleNode, data []byte) *MarkleNode {
	node := MarkleNode{}

	if left == nil && right == nil {
		hash := sha256.Sum256(data)
		node.Data = hash[:]
	}else{
		previousHashes := append(left.Data, right.Data...)
		hash := sha256.Sum256(previousHashes)
		node.Data = hash[:]
	}
	node.Left = left
	node.Right = right

	return &node
}


func NewMarkleTree(data [][]byte) *MarkleTree {
	var nodes []MarkleNode

	if len(data)%2 != 0 {
		data = append(data, data[len(data)-1])
	}

	for _, dat := range data {
		node := NewMarkleNode(nil, nil, dat)
		nodes = append(nodes, *node)
	}

	for i := 0; i < len(data)/2; i++ {
		var level []MarkleNode

		for j := 0; j < len(nodes); j += 2 {
			node := NewMarkleNode(&nodes[j], &nodes[j+1], nil)
			level = append(level, *node)
		}
		nodes =  level
	}

	tree := MarkleTree{&nodes[0]}

	return &tree
}