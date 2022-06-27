package hash

import (
	"errors"

	"github.com/ethereum/go-ethereum/common"
)

// MerkleTree implementation.
type MerkleTree struct {
	root  *MerkleTreeNode
	depth int
}

// NewMerkleTree returns new MerkleTree object.
func NewMerkleTree(hashes []common.Hash) (*MerkleTree, error) {
	if len(hashes) == 0 {
		return nil, errors.New("length of the hashes cannot be zero")
	}

	nodes := make([]*MerkleTreeNode, len(hashes))
	for i := 0; i < len(hashes); i++ {
		nodes[i] = &MerkleTreeNode{
			hash: hashes[i],
		}
	}

	return &MerkleTree{
		root:  buildMerkleTree(nodes),
		depth: 1,
	}, nil
}

// Root returns the computed root hash of the MerkleTree.
func (t *MerkleTree) Root() common.Hash {
	return t.root.hash
}

func buildMerkleTree(leaves []*MerkleTreeNode) *MerkleTreeNode {
	if len(leaves) == 0 {
		panic("length of leaves cannot be zero")
	}
	if len(leaves) == 1 {
		return leaves[0]
	}

	parents := make([]*MerkleTreeNode, (len(leaves)+1)/2)
	for i := 0; i < len(parents); i++ {
		parents[i] = &MerkleTreeNode{}
		parents[i].leftChild = leaves[i*2]
		leaves[i*2].parent = parents[i]

		if i*2+1 == len(leaves) {
			parents[i].rightChild = parents[i].leftChild
		} else {
			parents[i].rightChild = leaves[i*2+1]
			leaves[i*2+1].parent = parents[i]
		}

		b1 := parents[i].leftChild.hash.Bytes()
		b2 := parents[i].rightChild.hash.Bytes()
		b1 = append(b1, b2...)
		parents[i].hash = DoubleKeccak256(b1)
	}

	return buildMerkleTree(parents)
}

// CalcMerkleRoot calculcates Merkle root hash value for a given slice of hashes.
// It doesn't create a full MerkleTree structure and it uses given slice as a
// scratchpad, so it will destroy its contents in the process. But it's much more
// memory efficient if you only need root hash value, while NewMerkleTree would
// make 3*N allocations for N hashes, this function will only make 4. It also is
// an error to call this function for zero-length hashes slice, the function will
// panic.
func CalcMerkleRoot(hashes []common.Hash) common.Hash {
	if len(hashes) == 0 {
		return common.Hash{}
	}
	if len(hashes) == 1 {
		return hashes[0]
	}

	scratch := make([]byte, 64)
	parents := hashes[:(len(hashes)+1)/2]
	for i := 0; i < len(parents); i++ {
		copy(scratch, hashes[i*2].Bytes())

		if i*2+1 == len(hashes) {
			copy(scratch[32:], hashes[i*2].Bytes())
		} else {
			copy(scratch[32:], hashes[i*2+1].Bytes())
		}

		parents[i] = DoubleKeccak256(scratch)
	}

	return CalcMerkleRoot(parents)
}

// MerkleTreeNode represents a node in the MerkleTree.
type MerkleTreeNode struct {
	hash       common.Hash
	parent     *MerkleTreeNode
	leftChild  *MerkleTreeNode
	rightChild *MerkleTreeNode
}

// IsLeaf returns whether this node is a leaf node or not.
func (n *MerkleTreeNode) IsLeaf() bool {
	return n.leftChild == nil && n.rightChild == nil
}

// IsRoot returns whether this node is a root node or not.
func (n *MerkleTreeNode) IsRoot() bool {
	return n.parent == nil
}
