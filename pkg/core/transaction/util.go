package transaction

import (
	"encoding/json"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/multiverse-dev/saiya/pkg/io"
)

type writeCounter int

func (c *writeCounter) Write(b []byte) (int, error) {
	*c += writeCounter(len(b))
	return len(b), nil
}

func RlpSize(v interface{}) int {
	c := writeCounter(0)
	rlp.Encode(&c, v)
	return int(c)
}

func CalculateNetworkFee(tx *Transaction, feePerByte uint64) uint64 {
	switch tx.Type {
	case EthLegacyTxType:
		t := tx.LegacyTx
		size := EthLegacyBaseLength + len(t.Data)
		return uint64(size) * feePerByte
	case SaiyaTxType:
		t := tx.SaiyaTx
		size := 8 +
			io.GetVarSize(t.GasPrice.Bytes()) +
			8 +
			common.AddressLength +
			io.GetVarSize(t.Value.Bytes()) +
			io.GetVarSize(t.Data) +
			1 //from
		if t.To != nil {
			size += common.AddressLength
		}
		size += io.GetVarSize(t.Witness.VerificationScript)
		if t.Witness.VerificationScript[0] == 0 {
			size += SignatureLength + 1
		} else {
			size += 1 + int(t.Witness.VerificationScript[0])*(SignatureLength+1)
		}
		return uint64(size) * feePerByte
	default:
		return 0
	}
}

type legacyTxJson struct {
	Nonce    hexutil.Uint64  `json:"nonce"`
	GasPrice hexutil.Big     `json:"gasPrice"`
	Gas      hexutil.Uint64  `json:"gas"`
	To       *common.Address `json:"to,omitempty"`
	Value    hexutil.Big     `json:"value"`
	Data     hexutil.Bytes   `json:"data"`
	V        hexutil.Big     `json:"V"`
	R        hexutil.Big     `json:"R"`
	S        hexutil.Big     `json:"S"`
}

func marshlJSON(tx *types.LegacyTx) ([]byte, error) {
	t := &legacyTxJson{
		Nonce:    hexutil.Uint64(tx.Nonce),
		GasPrice: hexutil.Big(*tx.GasPrice),
		Gas:      hexutil.Uint64(tx.Gas),
		To:       tx.To,
		Value:    hexutil.Big(*tx.Value),
		Data:     hexutil.Bytes(tx.Data),
		V:        hexutil.Big(*tx.V),
		R:        hexutil.Big(*tx.R),
		S:        hexutil.Big(*tx.S),
	}
	return json.Marshal(t)
}

func unmarshalJSON(b []byte, tx *types.LegacyTx) error {
	t := new(legacyTxJson)
	err := json.Unmarshal(b, t)
	if err != nil {
		return err
	}
	tx.Nonce = uint64(t.Nonce)
	tx.GasPrice = (*big.Int)(&t.GasPrice)
	tx.Gas = uint64(t.Gas)
	tx.To = t.To
	tx.Value = (*big.Int)(&t.Value)
	tx.Data = []byte(t.Data)
	tx.V = (*big.Int)(&t.V)
	tx.R = (*big.Int)(&t.R)
	tx.S = (*big.Int)(&t.S)
	return nil
}
