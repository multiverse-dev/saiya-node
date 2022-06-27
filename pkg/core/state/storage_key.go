package state

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/multiverse-dev/saiya/pkg/io"
)

type StorageKey struct {
	Hash common.Address
	Key  []byte
}

func (sk *StorageKey) EncodeBinary(bw *io.BinWriter) {
	bw.WriteBytes(sk.Hash[:])
	bw.WriteVarBytes(sk.Key)
}

func (sk *StorageKey) DecodeBinary(br *io.BinReader) {
	br.ReadBytes(sk.Hash[:])
	sk.Key = br.ReadVarBytes()
}
