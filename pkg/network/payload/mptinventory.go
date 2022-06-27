package payload

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/multiverse-dev/saiya/pkg/io"
)

// MaxMPTHashesCount is the maximum number of requested MPT nodes hashes.
const MaxMPTHashesCount = 32

// MPTInventory payload.
type MPTInventory struct {
	// A list of requested MPT nodes hashes.
	Hashes []common.Hash
}

// NewMPTInventory return a pointer to an MPTInventory.
func NewMPTInventory(hashes []common.Hash) *MPTInventory {
	return &MPTInventory{
		Hashes: hashes,
	}
}

// DecodeBinary implements Serializable interface.
func (p *MPTInventory) DecodeBinary(br *io.BinReader) {
	count := br.ReadVarUint()
	if count > MaxMPTHashesCount {
		count = MaxMPTHashesCount
	}
	p.Hashes = make([]common.Hash, count)
	for i := uint64(0); i < count; i++ {
		br.ReadBytes(p.Hashes[i][:])
	}
}

// EncodeBinary implements Serializable interface.
func (p *MPTInventory) EncodeBinary(bw *io.BinWriter) {
	bw.WriteVarUint(uint64(len(p.Hashes)))
	for _, hash := range p.Hashes {
		bw.WriteBytes(hash[:])
	}
}
