package transaction

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/multiverse-dev/saiya/pkg/io"
	"github.com/stretchr/testify/assert"
)

func TestDecode(t *testing.T) {
	hex := "0x0100000000000000043b9aca00785d020000000000c20f7284c317f0b1d747aff294dbbd38a2347aad1400000000000000000000000000000000000000e40003010100406bd84ddf171651c38132d4bda4d393cc9b7e17f350360bc6bddbc0fc39eca564a9cea57f4c4461ecbbb0e1b385f14908792c3dcaab3f6bafb65ccbc506b6956a2200028c43c918440067b4b71b601871752a0b549092ff9aa9a7bc46f5244684f30ea5"
	tx := &SaiyaTx{}
	b, err := hexutil.Decode(hex)
	assert.NoError(t, err)
	err = io.FromByteArray(tx, b)
	assert.NoError(t, err)
}

func TestEncodeDecode(t *testing.T) {
	txx := &SaiyaTx{
		Nonce:    0,
		GasPrice: big.NewInt(1),
		Gas:      1,
		From:     common.HexToAddress("0xc20f7284c317f0b1d747aff294dbbd38a2347aad"),
		To:       &common.Address{},
		Value:    big.NewInt(0),
		Data:     []byte{1},
		Witness: Witness{
			VerificationScript: []byte{2},
			InvocationScript:   []byte{3},
		},
	}
	h1 := txx.Hash()
	b, err := io.ToByteArray(txx)
	assert.NoError(t, err)
	tx := &SaiyaTx{}
	err = io.FromByteArray(tx, b)
	assert.NoError(t, err)
	assert.Equal(t, h1, tx.Hash())
}
/*
0000000000000000  							8
043b9aca00									5
b018050000000000							8
b568f662934f30cd294ca0c055113344750e190b	20
1400000000000000000000000000000000000000e4	21
00											1
2401010103092c7fc564d67a2c589a4229c54f19358629529bc191daf1642d0a95989e3a83 		37
220003092c7fc564d67a2c589a4229c54f19358629529bc191daf1642d0a95989e3a83
4063a2227268a8027b4faba6d54095f799ce4eb1897a6ccf4e5607aab681ba7a07958f16575b6c6906ea7f11909b21727da9d8bb08393f13486ae518ea886f0cc3


0000000000000000
043b9aca00
0000000000000000
b568f662934f30cd294ca0c055113344750e190b
1400000000000000000000000000000000000000e4
00
2401010103092c7fc564d67a2c589a4229c54f19358629529bc191daf1642d0a95989e3a83
220003092c7fc564d67a2c589a4229c54f19358629529bc191daf1642d0a95989e3a83
00
*/
