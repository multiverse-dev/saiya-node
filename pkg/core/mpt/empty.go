package mpt

import (
	"encoding/json"
	"errors"

	"github.com/ethereum/go-ethereum/common"
	"github.com/multiverse-dev/saiya/pkg/io"
)

// EmptyNode represents empty node.
type EmptyNode struct{}

// DecodeBinary implements io.Serializable interface.
func (e EmptyNode) DecodeBinary(*io.BinReader) {
}

// EncodeBinary implements io.Serializable interface.
func (e EmptyNode) EncodeBinary(*io.BinWriter) {
}

// Size implements Node interface.
func (EmptyNode) Size() int { return 0 }

// MarshalJSON implements Node interface.
func (e EmptyNode) MarshalJSON() ([]byte, error) {
	return []byte(`{}`), nil
}

// UnmarshalJSON implements Node interface.
func (e EmptyNode) UnmarshalJSON(bytes []byte) error {
	var m map[string]interface{}
	err := json.Unmarshal(bytes, &m)
	if err != nil {
		return err
	}
	if len(m) != 0 {
		return errors.New("expected empty node")
	}
	return nil
}

// Hash implements Node interface.
func (e EmptyNode) Hash() common.Hash {
	panic("can't get hash of an EmptyNode")
}

// Type implements Node interface.
func (e EmptyNode) Type() NodeType {
	return EmptyT
}

// Bytes implements Node interface.
func (e EmptyNode) Bytes() []byte {
	return nil
}

// Clone implements Node interface.
func (EmptyNode) Clone() Node { return EmptyNode{} }
