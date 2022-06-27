package state

import (
	"bytes"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/multiverse-dev/saiya/pkg/core/storage"
	"github.com/multiverse-dev/saiya/pkg/encoding/bigint"
	"github.com/multiverse-dev/saiya/pkg/io"
)

// TokenTransferBatchSize is the maximum number of entries for TokenTransferLog.
const TokenTransferBatchSize = 128

// TokenTransferLog is a serialized log of token transfers.
type TokenTransferLog struct {
	Raw []byte
}

// ERC20Transfer represents a single NEP-17 Transfer event.
type ERC20Transfer struct {
	// Asset is a NEP-17 contract ID.
	ScriptHash common.Address
	// Address is the address of the sender.
	From common.Address
	// To is the address of the receiver.
	To common.Address
	// Amount is the amount of tokens transferred.
	// It is negative when tokens are sent and positive if they are received.
	Amount big.Int
	// Block is a number of block when the event occurred.
	Block uint32
	// Timestamp is the timestamp of the block where transfer occurred.
	Timestamp uint64
	// Tx is a hash the transaction.
	Tx common.Hash
}

// ERC721Transfer represents a single NEP-11 Transfer event.
type ERC721Transfer struct {
	ERC20Transfer

	// ID is a NEP-11 token ID.
	ID []byte
}

// TokenTransferInfo stores map of the contract IDs to the balance's last updated
// block trackers along with information about NEP-17 and NEP-11 transfer batch.
type TokenTransferInfo struct {
	LastUpdated map[common.Address]uint32
	// NextERC721Batch stores the index of the next NEP-17 transfer batch.
	NextERC721Batch uint32
	// NextERC20Batch stores the index of the next NEP-17 transfer batch.
	NextERC20Batch uint32
	// NextERC721NewestTimestamp stores the block timestamp of the first NEP-11 transfer in raw.
	NextERC721NewestTimestamp uint64
	// NextERC20NewestTimestamp stores the block timestamp of the first NEP-17 transfer in raw.
	NextERC20NewestTimestamp uint64
	// NewERC721Batch is true if batch with the `NextERC721Batch` index should be created.
	NewERC721Batch bool
	// NewERC20Batch is true if batch with the `NextERC20Batch` index should be created.
	NewERC20Batch bool
}

// NewTokenTransferInfo returns new TokenTransferInfo.
func NewTokenTransferInfo() *TokenTransferInfo {
	return &TokenTransferInfo{
		NewERC721Batch: true,
		NewERC20Batch:  true,
		LastUpdated:    make(map[common.Address]uint32),
	}
}

// DecodeBinary implements io.Serializable interface.
func (bs *TokenTransferInfo) DecodeBinary(r *io.BinReader) {
	bs.NextERC721Batch = r.ReadU32LE()
	bs.NextERC20Batch = r.ReadU32LE()
	bs.NextERC721NewestTimestamp = r.ReadU64LE()
	bs.NextERC20NewestTimestamp = r.ReadU64LE()
	bs.NewERC721Batch = r.ReadBool()
	bs.NewERC20Batch = r.ReadBool()
	lenBalances := r.ReadVarUint()
	m := make(map[common.Address]uint32, lenBalances)
	for i := 0; i < int(lenBalances); i++ {
		key := common.Address{}
		r.ReadBytes(key[:])
		m[key] = r.ReadU32LE()
	}
	bs.LastUpdated = m
}

// EncodeBinary implements io.Serializable interface.
func (bs *TokenTransferInfo) EncodeBinary(w *io.BinWriter) {
	w.WriteU32LE(bs.NextERC721Batch)
	w.WriteU32LE(bs.NextERC20Batch)
	w.WriteU64LE(bs.NextERC721NewestTimestamp)
	w.WriteU64LE(bs.NextERC20NewestTimestamp)
	w.WriteBool(bs.NewERC721Batch)
	w.WriteBool(bs.NewERC20Batch)
	w.WriteVarUint(uint64(len(bs.LastUpdated)))
	for k, v := range bs.LastUpdated {
		w.WriteBytes(k[:])
		w.WriteU32LE(v)
	}
}

// Append appends single transfer to a log.
func (lg *TokenTransferLog) Append(tr io.Serializable) error {
	// The first entry, set up counter.
	if len(lg.Raw) == 0 {
		lg.Raw = append(lg.Raw, 0)
	}

	b := bytes.NewBuffer(lg.Raw)
	w := io.NewBinWriterFromIO(b)

	tr.EncodeBinary(w)
	if w.Err != nil {
		return w.Err
	}
	lg.Raw = b.Bytes()
	lg.Raw[0]++
	return nil
}

// ForEachERC721 iterates over transfer log returning on first error.
func (lg *TokenTransferLog) ForEachERC721(f func(*ERC721Transfer) (bool, error)) (bool, error) {
	if lg == nil || len(lg.Raw) == 0 {
		return true, nil
	}
	transfers := make([]ERC721Transfer, lg.Size())
	r := io.NewBinReaderFromBuf(lg.Raw[1:])
	for i := 0; i < lg.Size(); i++ {
		transfers[i].DecodeBinary(r)
	}
	if r.Err != nil {
		return false, r.Err
	}
	for i := len(transfers) - 1; i >= 0; i-- {
		cont, err := f(&transfers[i])
		if err != nil || !cont {
			return false, err
		}
	}
	return true, nil
}

// ForEachERC20 iterates over transfer log returning on first error.
func (lg *TokenTransferLog) ForEachERC20(f func(*ERC20Transfer) (bool, error)) (bool, error) {
	if lg == nil || len(lg.Raw) == 0 {
		return true, nil
	}
	transfers := make([]ERC20Transfer, lg.Size())
	r := io.NewBinReaderFromBuf(lg.Raw[1:])
	for i := 0; i < lg.Size(); i++ {
		transfers[i].DecodeBinary(r)
	}
	if r.Err != nil {
		return false, r.Err
	}
	for i := len(transfers) - 1; i >= 0; i-- {
		cont, err := f(&transfers[i])
		if err != nil || !cont {
			return false, err
		}
	}
	return true, nil
}

// Size returns an amount of transfer written in log.
func (lg *TokenTransferLog) Size() int {
	if len(lg.Raw) == 0 {
		return 0
	}
	return int(lg.Raw[0])
}

// EncodeBinary implements io.Serializable interface.
func (t *ERC20Transfer) EncodeBinary(w *io.BinWriter) {
	w.WriteBytes(t.ScriptHash[:])
	w.WriteBytes(t.Tx[:])
	w.WriteBytes(t.From[:])
	w.WriteBytes(t.To[:])
	w.WriteU32LE(t.Block)
	w.WriteU64LE(t.Timestamp)
	amount := bigint.ToBytes(&t.Amount)
	w.WriteVarBytes(amount)
}

// DecodeBinary implements io.Serializable interface.
func (t *ERC20Transfer) DecodeBinary(r *io.BinReader) {
	r.ReadBytes(t.ScriptHash[:])
	r.ReadBytes(t.Tx[:])
	r.ReadBytes(t.From[:])
	r.ReadBytes(t.To[:])
	t.Block = r.ReadU32LE()
	t.Timestamp = r.ReadU64LE()
	amount := r.ReadVarBytes(bigint.MaxBytesLen)
	t.Amount = *bigint.FromBytes(amount)
}

// EncodeBinary implements io.Serializable interface.
func (t *ERC721Transfer) EncodeBinary(w *io.BinWriter) {
	t.ERC20Transfer.EncodeBinary(w)
	w.WriteVarBytes(t.ID)
}

// DecodeBinary implements io.Serializable interface.
func (t *ERC721Transfer) DecodeBinary(r *io.BinReader) {
	t.ERC20Transfer.DecodeBinary(r)
	t.ID = r.ReadVarBytes(storage.MaxStorageKeyLen)
}
