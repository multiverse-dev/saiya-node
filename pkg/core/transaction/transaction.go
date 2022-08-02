package transaction

import (
	"encoding/json"
	"errors"
	"math"
	"math/big"
	"sync/atomic"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/multiverse-dev/saiya/pkg/crypto/hash"
	"github.com/multiverse-dev/saiya/pkg/io"
)

const (
	EthLegacyTxType     = byte(0)
	SaiyaTxType         = byte(1)
	SignatureLength     = 64
	MaxScriptLength     = math.MaxUint16
	MaxTransactionSize  = 102400
	EthLegacyBaseLength = 100
)

var (
	ErrUnsupportType = errors.New("unsupport tx type")
)

type Transaction struct {
	Type     byte
	LegacyTx *types.LegacyTx
	SaiyaTx  *SaiyaTx

	Trimmed bool
	EthSize int
	EthFrom common.Address
	hash    atomic.Value
	size    atomic.Value
}

func NewTrimmedTX(hash common.Hash) *Transaction {
	t := &Transaction{
		Trimmed: true,
	}
	t.hash.Store(hash)
	return t
}

func NewTx(t interface{}) *Transaction {
	tx := &Transaction{}
	switch v := t.(type) {
	case *SaiyaTx:
		tx.Type = SaiyaTxType
		tx.SaiyaTx = v
	case *types.LegacyTx:
		tx.Type = EthLegacyTxType
		tx.LegacyTx = v
	default:
		panic("unsupport tx")
	}
	return tx
}

func NewTransactionFromBytes(b []byte) (*Transaction, error) {
	tx := &Transaction{}
	err := io.FromByteArray(tx, b)
	if err != nil {
		return nil, err
	}
	return tx, err
}

func (t *Transaction) Nonce() uint64 {
	switch t.Type {
	case EthLegacyTxType:
		return t.LegacyTx.Nonce
	case SaiyaTxType:
		return t.SaiyaTx.Nonce
	default:
		panic(ErrUnsupportType)
	}
}

func (t *Transaction) To() *common.Address {
	switch t.Type {
	case EthLegacyTxType:
		return t.LegacyTx.To
	case SaiyaTxType:
		return t.SaiyaTx.To
	default:
		panic(ErrUnsupportType)
	}
}

func (t *Transaction) Gas() uint64 {
	switch t.Type {
	case EthLegacyTxType:
		return t.LegacyTx.Gas
	case SaiyaTxType:
		return t.SaiyaTx.Gas
	default:
		panic(ErrUnsupportType)
	}
}

func (t *Transaction) GasPrice() *big.Int {
	switch t.Type {
	case EthLegacyTxType:
		return t.LegacyTx.GasPrice
	case SaiyaTxType:
		return t.SaiyaTx.GasPrice
	default:
		panic(ErrUnsupportType)
	}
}

func (t Transaction) Cost() *big.Int {
	cost := big.NewInt(0).Mul(big.NewInt(int64(t.Gas())), t.GasPrice())
	return big.NewInt(0).Add(t.Value(), cost)
}

func (t *Transaction) Value() *big.Int {
	switch t.Type {
	case EthLegacyTxType:
		return t.LegacyTx.Value
	case SaiyaTxType:
		return t.SaiyaTx.Value
	default:
		panic(ErrUnsupportType)
	}
}

func (t *Transaction) Data() []byte {
	switch t.Type {
	case EthLegacyTxType:
		return t.LegacyTx.Data
	case SaiyaTxType:
		return t.SaiyaTx.Data
	default:
		panic(ErrUnsupportType)
	}
}

func (t *Transaction) Size() int {
	if size := t.size.Load(); size != nil {
		return size.(int)
	}
	var size int
	switch t.Type {
	case EthLegacyTxType:
		size = RlpSize(t.LegacyTx)
	case SaiyaTxType:
		size = t.SaiyaTx.Size()
	default:
		panic(ErrUnsupportType)
	}
	t.size.Store(size)
	return size
}

func (t *Transaction) From() common.Address {
	switch t.Type {
	case EthLegacyTxType:
		return t.EthFrom
	case SaiyaTxType:
		return t.SaiyaTx.From
	default:
		panic(ErrUnsupportType)
	}
}

func (t *Transaction) Hash() common.Hash {
	if hash := t.hash.Load(); hash != nil {
		return hash.(common.Hash)
	}
	var h common.Hash
	if t.Type == EthLegacyTxType {
		h = hash.RlpHash(t.LegacyTx)
	} else {
		h = t.SaiyaTx.Hash()
	}
	t.hash.Store(h)
	return h
}

func (t Transaction) SignHash(chainId uint64) common.Hash {
	if t.Type == EthLegacyTxType {
		signer := types.NewEIP2930Signer(big.NewInt(int64(chainId)))
		return signer.Hash(types.NewTx(t.LegacyTx))
	} else {
		return t.Hash()
	}
}

func (t *Transaction) Bytes() ([]byte, error) {
	return io.ToByteArray(t)
}

func (t Transaction) FeePerByte() uint64 {
	return t.Gas() / uint64(t.Size())
}

func (t *Transaction) EncodeBinary(w *io.BinWriter) {
	w.WriteB(t.Type)
	switch t.Type {
	case EthLegacyTxType:
		err := rlp.Encode(w, t.LegacyTx)
		w.Err = err
	case SaiyaTxType:
		t.SaiyaTx.EncodeBinary(w)
	default:
		w.Err = ErrUnsupportType
	}
}

func (t *Transaction) DecodeBinary(r *io.BinReader) {
	t.Type = r.ReadB()
	switch t.Type {
	case EthLegacyTxType:
		inner := new(types.LegacyTx)
		err := rlp.Decode(r, inner)
		r.Err = err
		t.LegacyTx = inner
	case SaiyaTxType:
		inner := new(SaiyaTx)
		inner.DecodeBinary(r)
		t.SaiyaTx = inner
	default:
		r.Err = ErrUnsupportType
	}
}

func (t *Transaction) Verify(chainId uint64) error {
	switch t.Type {
	case EthLegacyTxType:
		signer := types.NewEIP2930Signer(big.NewInt(int64(chainId)))
		from, err := signer.Sender(types.NewTx(t.LegacyTx))
		if err != nil {
			return err
		}
		t.EthFrom = from
		return nil
	case SaiyaTxType:
		return t.SaiyaTx.Witness.VerifyHashable(chainId, t.SaiyaTx)
	default:
		return ErrUnsupportType
	}
}

func (t *Transaction) WithSignature(chainId uint64, sig []byte) error {
	switch t.Type {
	case EthLegacyTxType:
		signer := types.NewEIP2930Signer(big.NewInt(int64(chainId)))
		r, s, v, err := signer.SignatureValues(types.NewTx(t.LegacyTx), sig)
		if err != nil {
			return err
		}
		t.LegacyTx.V, t.LegacyTx.R, t.LegacyTx.S = v, r, s
		return nil
	default:
		return ErrUnsupportType
	}
}

func (t *Transaction) WithWitness(witness Witness) error {
	if t.Type != SaiyaTxType {
		return ErrUnsupportType
	}
	t.SaiyaTx.Witness = witness
	return nil
}

func (t *Transaction) UnmarshalJSON(b []byte) error {
	if t.Type == EthLegacyTxType {
		tx := new(types.LegacyTx)
		err := unmarshalJSON(b, tx)
		if err != nil {
			return err
		}
		t.LegacyTx = tx
		return nil
	} else if t.Type == SaiyaTxType {
		tx := new(SaiyaTx)
		err := json.Unmarshal(b, tx)
		if err != nil {
			return err
		}
		t.SaiyaTx = tx
		return nil
	} else {
		return ErrUnsupportType
	}
}

func (t *Transaction) MarshalJSON() ([]byte, error) {
	if t.Trimmed {
		return json.Marshal(t.Hash())
	}
	switch t.Type {
	case EthLegacyTxType:
		return marshlJSON(t.LegacyTx)
	case SaiyaTxType:
		return json.Marshal(t.SaiyaTx)
	default:
		return nil, ErrUnsupportType
	}
}

var (
	ErrInvalidTxType = errors.New("invalid tx type")
)

func (t Transaction) IsValid() error {
	switch t.Type {
	case EthLegacyTxType:
		if t.LegacyTx.Value.Sign() < 0 {
			return ErrNegativeValue
		}
		return nil
	case SaiyaTxType:
		return t.SaiyaTx.isValid()
	default:
		return ErrInvalidTxType
	}
}
