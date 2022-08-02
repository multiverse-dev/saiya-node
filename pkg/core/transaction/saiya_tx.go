package transaction

import (
	"encoding/json"
	"errors"
	"math/big"
	"math/rand"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/multiverse-dev/saiya/pkg/crypto/hash"
	"github.com/multiverse-dev/saiya/pkg/io"
	"golang.org/x/crypto/sha3"
)

// ErrInvalidWitnessNum returns when the number of witnesses does not match signers.
var (
	ErrNoSender = errors.New("no sender in trimmed tx")
)

type SaiyaTx struct {
	Nonce    uint64
	GasPrice *big.Int
	Gas      uint64
	From     common.Address
	To       *common.Address
	Value    *big.Int
	Data     []byte
	Witness  Witness

	len    int
	hash   common.Hash
	hashed bool

	Trimmed bool
}

// NewTrimmedTX returns a trimmed transaction with only its hash
// and Trimmed to true.

// New returns a new transaction to execute given script and pay given system
// fee.
func New(data []byte, gas uint64) *SaiyaTx {
	return &SaiyaTx{
		Nonce: rand.Uint64(),
		Data:  data,
	}
}

func NewSaiyaTxFromBytes(b []byte) (*SaiyaTx, error) {
	tx := &SaiyaTx{}
	err := io.FromByteArray(tx, b)
	if err != nil {
		return nil, err
	}
	return tx, err
}

// Hash returns the hash of the transaction.
func (t *SaiyaTx) Hash() common.Hash {
	if !t.hashed {
		if t.createHash() != nil {
			panic("failed to compute hash!")
		}
	}
	return t.hash
}

// decodeHashableFields decodes the fields that are used for signing the
// transaction, which are all fields except the scripts.
func (t *SaiyaTx) decodeHashableFields(br *io.BinReader, buf []byte) {
	var start, end int

	if buf != nil {
		start = len(buf) - br.Len()
	}
	t.Nonce = br.ReadU64LE()
	pricebs := br.ReadVarBytes()
	t.GasPrice = big.NewInt(0).SetBytes(pricebs)
	t.Gas = br.ReadU64LE()
	br.ReadBytes(t.From[:])
	tob := br.ReadVarBytes(common.AddressLength)
	if len(tob) == 0 {
		t.To = nil
	} else {
		to := common.BytesToAddress(tob)
		t.To = &to
	}
	valuebs := br.ReadVarBytes()
	t.Value = big.NewInt(0).SetBytes(valuebs)
	t.Data = br.ReadVarBytes(MaxScriptLength)
	if br.Err == nil {
		br.Err = t.isValid()
	}
	if buf != nil {
		end = len(buf) - br.Len()
		t.hash = hash.Keccak256(buf[start:end])
		t.hashed = true
	}
}

func (t *SaiyaTx) decodeBinaryNoSize(br *io.BinReader, buf []byte) {
	t.decodeHashableFields(br, buf)
	if br.Err != nil {
		return
	}
	t.Witness.DecodeBinary(br)

	// Create the hash of the transaction at decode, so we dont need
	// to do it anymore.
	if br.Err == nil && buf == nil {
		br.Err = t.createHash()
	}
}

// DecodeBinary implements Serializable interface.
func (t *SaiyaTx) DecodeBinary(br *io.BinReader) {
	t.decodeBinaryNoSize(br, nil)

	if br.Err == nil {
		_ = t.Size()
	}
}

// EncodeBinary implements Serializable interface.
func (t *SaiyaTx) EncodeBinary(bw *io.BinWriter) {
	t.encodeHashableFields(bw)
	t.Witness.EncodeBinary(bw)
}

// encodeHashableFields encodes the fields that are not used for
// signing the transaction, which are all fields except the scripts.
func (t *SaiyaTx) encodeHashableFields(bw *io.BinWriter) {
	bw.WriteU64LE(t.Nonce)
	if t.GasPrice == nil {
		bw.WriteVarUint(0)
	} else {
		bw.WriteVarBytes(t.GasPrice.Bytes())
	}
	bw.WriteU64LE(t.Gas)
	bw.WriteBytes(t.From.Bytes())
	if t.To == nil {
		bw.WriteVarUint(0)
	} else {
		bw.WriteVarBytes(t.To.Bytes())
	}
	if t.Value == nil {
		bw.WriteVarUint(0)
	} else {
		bw.WriteVarBytes(t.Value.Bytes())
	}
	bw.WriteVarBytes(t.Data)
}

// EncodeHashableFields returns serialized transaction's fields which are hashed.
func (t *SaiyaTx) EncodeHashableFields() ([]byte, error) {
	bw := io.NewBufBinWriter()
	t.encodeHashableFields(bw.BinWriter)
	if bw.Err != nil {
		return nil, bw.Err
	}
	return bw.Bytes(), nil
}

// createHash creates the hash of the transaction.
func (t *SaiyaTx) createHash() error {
	shaHash := sha3.NewLegacyKeccak256()
	bw := io.NewBinWriterFromIO(shaHash)
	t.encodeHashableFields(bw)
	if bw.Err != nil {
		return bw.Err
	}

	shaHash.Sum(t.hash[:0])
	t.hashed = true
	return nil
}

// DecodeHashableFields decodes a part of transaction which should be hashed.
func (t *SaiyaTx) DecodeHashableFields(buf []byte) error {
	r := io.NewBinReaderFromBuf(buf)
	t.decodeHashableFields(r, buf)
	if r.Err != nil {
		return r.Err
	}
	// Ensure all the data was read.
	if r.Len() != 0 {
		return errors.New("additional data after the signed part")
	}
	return nil
}

// Bytes converts the transaction to []byte.
func (t *SaiyaTx) Bytes() ([]byte, error) {
	buf := io.NewBufBinWriter()
	t.EncodeBinary(buf.BinWriter)
	if buf.Err != nil {
		return nil, buf.Err
	}
	return buf.Bytes(), nil
}

// Size returns size of the serialized transaction.
func (t *SaiyaTx) Size() int {
	if t.len == 0 {
		t.len = io.GetVarSize(t)
	}
	return t.len
}

// transactionJSON is a wrapper for SaiyaTx and
// used for correct marhalling of transaction.Data.
type saiyaTxJson struct {
	TxID     common.Hash     `json:"hash"`
	Size     hexutil.Uint    `json:"size"`
	Nonce    hexutil.Uint64  `json:"nonce"`
	GasPrice hexutil.Big     `json:"gasPrice"`
	Gas      hexutil.Uint64  `json:"gas"`
	From     common.Address  `json:"from"`
	To       *common.Address `json:"to,omitempty"`
	Value    hexutil.Big     `json:"value"`
	Data     hexutil.Bytes   `json:"data"`
	Witness  Witness         `json:"witness"`
}

// MarshalJSON implements json.Marshaler interface.
func (t *SaiyaTx) MarshalJSON() ([]byte, error) {
	tx := saiyaTxJson{
		TxID:     t.Hash(),
		Size:     hexutil.Uint(t.Size()),
		Nonce:    hexutil.Uint64(t.Nonce),
		GasPrice: hexutil.Big(*t.GasPrice),
		Gas:      hexutil.Uint64(t.Gas),
		From:     t.From,
		To:       t.To,
		Value:    hexutil.Big(*t.Value),
		Data:     t.Data,
		Witness:  t.Witness,
	}
	return json.Marshal(tx)
}

// UnmarshalJSON implements json.Unmarshaler interface.
func (t *SaiyaTx) UnmarshalJSON(data []byte) error {
	tx := new(saiyaTxJson)
	if err := json.Unmarshal(data, tx); err != nil {
		return err
	}
	t.Nonce = uint64(tx.Nonce)
	t.GasPrice = (*big.Int)(&tx.GasPrice)
	t.Gas = uint64(tx.Gas)
	t.From = tx.From
	t.To = tx.To
	t.Value = (*big.Int)(&tx.Value)
	t.Data = tx.Data
	t.Witness = tx.Witness
	return t.isValid()
}

// Various errors for transaction validation.
var (
	ErrNegativeValue  = errors.New("negative value")
	ErrWitnessUnmatch = errors.New("witness not match from")
)

// isValid checks whether decoded/unmarshalled transaction has all fields valid.
func (t *SaiyaTx) isValid() error {
	if t.Value.Sign() < 0 {
		return ErrNegativeValue
	}
	return nil
}
