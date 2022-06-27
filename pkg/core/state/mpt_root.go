package state

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/multiverse-dev/saiya/pkg/core/transaction"
	"github.com/multiverse-dev/saiya/pkg/crypto/hash"
	"github.com/multiverse-dev/saiya/pkg/io"
)

// MPTRoot represents storage state root together with sign info.
type MPTRoot struct {
	Version byte                `json:"version"`
	Index   uint32              `json:"index"`
	Root    common.Hash         `json:"roothash"`
	Witness transaction.Witness `json:"witnesses"`
}

// Hash returns hash of s.
func (s *MPTRoot) Hash() common.Hash {
	buf := io.NewBufBinWriter()
	s.EncodeBinaryUnsigned(buf.BinWriter)
	return hash.Keccak256(buf.Bytes())
}

// DecodeBinaryUnsigned decodes hashable part of state root.
func (s *MPTRoot) DecodeBinaryUnsigned(r *io.BinReader) {
	s.Version = r.ReadB()
	s.Index = r.ReadU32LE()
	r.ReadBytes(s.Root[:])
}

// EncodeBinaryUnsigned encodes hashable part of state root..
func (s *MPTRoot) EncodeBinaryUnsigned(w *io.BinWriter) {
	w.WriteB(s.Version)
	w.WriteU32LE(s.Index)
	w.WriteBytes(s.Root[:])
}

// DecodeBinary implements io.Serializable.
func (s *MPTRoot) DecodeBinary(r *io.BinReader) {
	s.DecodeBinaryUnsigned(r)
	s.Witness.DecodeBinary(r)
}

// EncodeBinary implements io.Serializable.
func (s *MPTRoot) EncodeBinary(w *io.BinWriter) {
	s.EncodeBinaryUnsigned(w)
	s.Witness.EncodeBinary(w)
}
