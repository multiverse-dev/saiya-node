package dao

import (
	"encoding/json"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/multiverse-dev/saiya/pkg/core/storage"
	"github.com/multiverse-dev/saiya/pkg/core/transaction"
	"github.com/stretchr/testify/assert"
)

func TestStoreTx(t *testing.T) {
	d := NewSimple(storage.NewMemoryStore())
	tx := transaction.NewTx(&types.LegacyTx{})
	receipt := &types.Receipt{}
	err := d.StoreAsTransaction(tx, 0, receipt)
	assert.NoError(t, err)
	_, err = d.GetReceipt(tx.Hash())
	assert.NoError(t, err)
	txx, h, err := d.GetTransaction(tx.Hash())
	assert.NoError(t, err)
	assert.Equal(t, uint32(0), h)
	assert.True(t, txx.Hash() == tx.Hash())
}

func TestReceipt(t *testing.T) {
	j := `{"root":"0x","status":"0x0","cumulativeGasUsed":"0x186a0","logsBloom":"0x00000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000","logs":[],"transactionHash":"0x9f27c2aba7114513ec76cc4455feeed833ef064411b3f504199c0c5b80da44d4","contractAddress":"0x0000000000000000000000000000000000000000","gasUsed":"0x186a0","blockHash":"0x430079bc144da1d744f44d6ef830843d50ea610a5f71ed07f55545c4cdcda63d","blockNumber":"0x3","transactionIndex":"0x0"}`
	r := &types.Receipt{}
	err := json.Unmarshal([]byte(j), r)
	assert.NoError(t, err)
	assert.Equal(t, common.HexToHash("0x9f27c2aba7114513ec76cc4455feeed833ef064411b3f504199c0c5b80da44d4"), r.TxHash)
	b, err := rlp.EncodeToBytes(r)
	assert.NoError(t, err)
	rr := &types.Receipt{}
	err = rlp.DecodeBytes(b, rr)
	assert.NoError(t, err)
	assert.Equal(t, common.HexToHash("0x9f27c2aba7114513ec76cc4455feeed833ef064411b3f504199c0c5b80da44d4"), rr.TxHash)
}

func TestHeaderHashes(t *testing.T) {
	d := NewSimple(storage.NewMemoryStore())
	hashes := []common.Hash{
		common.HexToHash("0x430079bc144da1d744f44d6ef830843d50ea610a5f71ed07f55545c4cdcda63d"),
		common.Hash{},
	}
	err := d.StoreHeaderHashes(hashes, 1)
	assert.NoError(t, err)
	hs, err := d.GetHeaderHashes()
	assert.NoError(t, err)
	assert.Equal(t, 2, len(hs))
}
