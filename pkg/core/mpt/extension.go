package mpt

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/multiverse-dev/saiya/pkg/core/storage"
	"github.com/multiverse-dev/saiya/pkg/io"
)

const (
	// maxPathLength is the max length of the extension node key.
	maxPathLength = (storage.MaxStorageKeyLen + 4) * 2

	// MaxKeyLength is the max length of the key to put in trie
	// before transforming to nibbles.
	MaxKeyLength = maxPathLength / 2
)

// ExtensionNode represents MPT's extension node.
type ExtensionNode struct {
	BaseNode
	key  []byte
	next Node
}

var _ Node = (*ExtensionNode)(nil)

// NewExtensionNode returns hash node with the specified key and next node.
// Note: because it is a part of Trie, key must be mangled, i.e. must contain only bytes with high half = 0.
func NewExtensionNode(key []byte, next Node) *ExtensionNode {
	return &ExtensionNode{
		key:  key,
		next: next,
	}
}

// Type implements Node interface.
func (e ExtensionNode) Type() NodeType { return ExtensionT }

// Hash implements BaseNode interface.
func (e *ExtensionNode) Hash() common.Hash {
	return e.getHash(e)
}

// Bytes implements BaseNode interface.
func (e *ExtensionNode) Bytes() []byte {
	return e.getBytes(e)
}

// DecodeBinary implements io.Serializable.
func (e *ExtensionNode) DecodeBinary(r *io.BinReader) {
	sz := r.ReadVarUint()
	if sz > maxPathLength {
		r.Err = fmt.Errorf("extension node key is too big: %d", sz)
		return
	}
	e.key = make([]byte, sz)
	r.ReadBytes(e.key)
	no := new(NodeObject)
	no.DecodeBinary(r)
	e.next = no.Node
	e.invalidateCache()
}

// EncodeBinary implements io.Serializable.
func (e ExtensionNode) EncodeBinary(w *io.BinWriter) {
	w.WriteVarBytes(e.key)
	encodeBinaryAsChild(e.next, w)
}

// Size implements Node interface.
func (e *ExtensionNode) Size() int {
	return io.GetVarSize(len(e.key)) + len(e.key) +
		1 + common.HashLength // e.next is never empty
}

// MarshalJSON implements json.Marshaler.
func (e *ExtensionNode) MarshalJSON() ([]byte, error) {
	m := map[string]interface{}{
		"key":  hex.EncodeToString(e.key),
		"next": e.next,
	}
	return json.Marshal(m)
}

// UnmarshalJSON implements json.Unmarshaler.
func (e *ExtensionNode) UnmarshalJSON(data []byte) error {
	var obj NodeObject
	if err := obj.UnmarshalJSON(data); err != nil {
		return err
	} else if u, ok := obj.Node.(*ExtensionNode); ok {
		*e = *u
		return nil
	}
	return errors.New("expected extension node")
}

// Clone implements Node interface.
func (e *ExtensionNode) Clone() Node {
	res := *e
	return &res
}
