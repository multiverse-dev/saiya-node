package block

import (
	"encoding/json"
	"errors"
	"math"

	"github.com/ethereum/go-ethereum/common"
	"github.com/multiverse-dev/saiya/pkg/core/transaction"
	"github.com/multiverse-dev/saiya/pkg/crypto/hash"
	"github.com/multiverse-dev/saiya/pkg/io"
)

const (
	// MaxTransactionsPerBlock is the maximum number of transactions per block.
	MaxTransactionsPerBlock = math.MaxUint16
)

// ErrMaxContentsPerBlock is returned when the maximum number of contents per block is reached.
var ErrMaxContentsPerBlock = errors.New("the number of contents exceeds the maximum number of contents per block")

var expectedHeaderSizeWithEmptyWitness int

func init() {
	expectedHeaderSizeWithEmptyWitness = io.GetVarSize(new(Header))
}

// Block represents one block in the chain.
type Block struct {
	// The base of the block.
	Header

	// Transaction list.
	Transactions []*transaction.Transaction

	// True if this block is created from trimmed data.
	Trimmed bool
}

// auxBlockOut is used for JSON i/o.
type auxBlockOut struct {
	Transactions []*transaction.Transaction `json:"tx"`
}

// auxBlockIn is used for JSON i/o.
type auxBlockIn struct {
	Transactions []json.RawMessage `json:"tx"`
}

// ComputeMerkleRoot computes Merkle tree root hash based on actual block's data.
func (b *Block) ComputeMerkleRoot() common.Hash {
	hashes := make([]common.Hash, len(b.Transactions))
	for i, tx := range b.Transactions {
		hashes[i] = tx.Hash()
	}

	return hash.CalcMerkleRoot(hashes)
}

// RebuildMerkleRoot rebuilds the merkleroot of the block.
func (b *Block) RebuildMerkleRoot() {
	b.MerkleRoot = b.ComputeMerkleRoot()
}

// NewTrimmedFromReader returns a new block from trimmed data.
// This is commonly used to create a block from stored data.
// Blocks created from trimmed data will have their Trimmed field
// set to true.
func NewTrimmedFromReader(br *io.BinReader) (*Block, error) {
	block := &Block{
		Header:  Header{},
		Trimmed: true,
	}

	block.Header.DecodeBinary(br)
	lenHashes := br.ReadVarUint()
	if lenHashes > MaxTransactionsPerBlock {
		return nil, ErrMaxContentsPerBlock
	}
	if lenHashes > 0 {
		block.Transactions = make([]*transaction.Transaction, lenHashes)
		for i := 0; i < int(lenHashes); i++ {
			var hash common.Hash
			br.ReadBytes(hash[:])
			block.Transactions[i] = transaction.NewTrimmedTX(hash)
		}
	}

	return block, br.Err
}

// New creates a new blank block with proper state root setting.
func New() *Block {
	return &Block{
		Header: Header{},
	}
}

// EncodeTrimmed writes trimmed representation of the block data into w. Trimmed blocks
// do not store complete transactions, instead they only store their hashes.
func (b *Block) EncodeTrimmed(w *io.BinWriter) {
	b.Header.EncodeBinary(w)

	w.WriteVarUint(uint64(len(b.Transactions)))
	for _, tx := range b.Transactions {
		h := tx.Hash()
		w.WriteBytes(h.Bytes())
	}
}

// DecodeBinary decodes the block from the given BinReader, implementing
// Serializable interface.
func (b *Block) DecodeBinary(br *io.BinReader) {
	b.Header.DecodeBinary(br)
	contentsCount := br.ReadVarUint()
	if contentsCount > MaxTransactionsPerBlock {
		br.Err = ErrMaxContentsPerBlock
		return
	}
	txes := make([]*transaction.Transaction, contentsCount)
	for i := 0; i < int(contentsCount); i++ {
		tx := &transaction.Transaction{}
		tx.DecodeBinary(br)
		txes[i] = tx
	}
	b.Transactions = txes
	if br.Err != nil {
		return
	}
}

// EncodeBinary encodes the block to the given BinWriter, implementing
// Serializable interface.
func (b *Block) EncodeBinary(bw *io.BinWriter) {
	b.Header.EncodeBinary(bw)
	bw.WriteVarUint(uint64(len(b.Transactions)))
	for i := 0; i < len(b.Transactions); i++ {
		b.Transactions[i].EncodeBinary(bw)
	}
}

// MarshalJSON implements json.Marshaler interface.
func (b Block) MarshalJSON() ([]byte, error) {
	auxb, err := json.Marshal(auxBlockOut{
		Transactions: b.Transactions,
	})
	if err != nil {
		return nil, err
	}
	baseBytes, err := json.Marshal(b.Header)
	if err != nil {
		return nil, err
	}

	// Stitch them together.
	if baseBytes[len(baseBytes)-1] != '}' || auxb[0] != '{' {
		return nil, errors.New("can't merge internal jsons")
	}
	baseBytes[len(baseBytes)-1] = ','
	baseBytes = append(baseBytes, auxb[1:]...)
	return baseBytes, nil
}

// UnmarshalJSON implements json.Unmarshaler interface.
func (b *Block) UnmarshalJSON(data []byte) error {
	// As Base and auxb are at the same level in json,
	// do unmarshalling separately for both structs.
	auxb := new(auxBlockIn)
	err := json.Unmarshal(data, auxb)
	if err != nil {
		return err
	}
	err = json.Unmarshal(data, &b.Header)
	if err != nil {
		return err
	}
	if len(auxb.Transactions) != 0 {
		b.Transactions = make([]*transaction.Transaction, 0, len(auxb.Transactions))
		for _, txBytes := range auxb.Transactions {
			tx := &transaction.Transaction{}
			err = tx.UnmarshalJSON(txBytes)
			if err != nil {
				return err
			}
			b.Transactions = append(b.Transactions, tx)
		}
	}
	return nil
}

// GetExpectedBlockSize returns expected block size which should be equal to io.GetVarSize(b).
func (b *Block) GetExpectedBlockSize() int {
	var transactionsSize int
	for _, tx := range b.Transactions {
		transactionsSize += tx.Size()
	}
	return b.GetExpectedBlockSizeWithoutTransactions(len(b.Transactions)) + transactionsSize
}

// GetExpectedBlockSizeWithoutTransactions returns expected block size without transactions size.
func (b *Block) GetExpectedBlockSizeWithoutTransactions(txCount int) int {
	size := expectedHeaderSizeWithEmptyWitness - 1 - 1 + // 1 is for the zero-length (new(Header)).Script.Invocation/Verification
		io.GetVarSize(&b.Witness) +
		io.GetVarSize(txCount)
	return size
}
