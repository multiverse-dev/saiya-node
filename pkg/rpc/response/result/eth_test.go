package result

import (
	"encoding/json"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/stretchr/testify/assert"
)

func TestJsonRequestTx(t *testing.T) {
	to := common.HexToAddress("0xd46E8dD67C5d32be8058Bb8Eb970870F07244567")
	data, _ := hexutil.Decode("0xd46e8dd67c5d32be8d46e8dd67c5d32be8058bb8eb970870f072445675058bb8eb970870f072445675")
	tx := TransactionObject{
		Data:     data,
		From:     common.HexToAddress("0xb60E8dD61C5d32be8058BB8eb970870F07233155"),
		Gas:      30400,
		GasPrice: big.NewInt(10000000000000),
		To:       &to,
		Value:    big.NewInt(2441406250),
	}
	data, err := json.Marshal(tx)
	assert.NoError(t, err)
	j := `{"data":"0xd46e8dd67c5d32be8d46e8dd67c5d32be8058bb8eb970870f072445675058bb8eb970870f072445675","from":"0xb60e8dd61c5d32be8058bb8eb970870f07233155","gas":"0x76c0","gasPrice":"0x9184e72a000","to":"0xd46e8dd67c5d32be8058bb8eb970870f07244567","value":"0x9184e72a"}`
	assert.Equal(t, j, string(data))
}

func TestTxObjectUnmarshal(t *testing.T) {
	j := `{"data": "0xd46e8dd67c5d32be8d46e8dd67c5d32be8058bb8eb970870f072445675058bb8eb970870f072445675","from": "0xb60e8dd61c5d32be8058bb8eb970870f07233155","gas": "0x76c0","gasPrice": "0x9184e72a000","to":"0xd46e8dd67c5d32be8058bb8eb970870f07244567","value": "0x9184e72a"}`
	tx := TransactionObject{}
	err := json.Unmarshal([]byte(j), &tx)
	assert.NoError(t, err)
	data, _ := hexutil.Decode("0xd46e8dd67c5d32be8d46e8dd67c5d32be8058bb8eb970870f072445675058bb8eb970870f072445675")
	assert.Equal(t, data, tx.Data)
	from := common.HexToAddress("0xb60e8dd61c5d32be8058bb8eb970870f07233155")
	assert.Equal(t, from, tx.From)
	gas, _ := hexutil.DecodeUint64("0x76c0")
	assert.Equal(t, gas, tx.Gas)
	gp, _ := hexutil.DecodeUint64("0x9184e72a000")
	assert.Equal(t, gp, tx.GasPrice)
	to := common.HexToAddress("0xd46e8dd67c5d32be8058bb8eb970870f07244567")
	assert.Equal(t, to, tx.To)
	value, _ := hexutil.DecodeUint64("0x9184e72a")
	assert.Equal(t, value, tx.Value)
	t.Logf("\n%+v\n", tx)
}
