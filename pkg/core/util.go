package core

import (
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/multiverse-dev/saiya/pkg/config"
	"github.com/multiverse-dev/saiya/pkg/core/block"
	"github.com/multiverse-dev/saiya/pkg/core/native"
	"github.com/multiverse-dev/saiya/pkg/core/transaction"
	"github.com/multiverse-dev/saiya/pkg/crypto/hash"
	"github.com/multiverse-dev/saiya/pkg/crypto/keys"
)

// createGenesisBlock creates a genesis block based on the given configuration.
func createGenesisBlock(cfg config.ProtocolConfiguration) (*block.Block, error) {
	base := block.Header{
		Version:   0,
		PrevHash:  common.Hash{},
		Timestamp: uint64(time.Date(2016, 7, 15, 15, 8, 21, 0, time.UTC).Unix()) * 1000, // Milliseconds.
		Nonce:     2083236893,
		Index:     0,
		Witness: transaction.Witness{
			VerificationScript: []byte{},
			InvocationScript:   []byte{},
		},
	}
	initData := []byte{0x00}
	b := &block.Block{
		Header: base,
		Transactions: []*transaction.Transaction{
			transaction.NewTx(&types.LegacyTx{
				To:    &native.DesignationAddress,
				Data:  initData,
				Value: big.NewInt(0),
			}),
			transaction.NewTx(&types.LegacyTx{
				To:    &native.PolicyAddress,
				Data:  initData,
				Value: big.NewInt(0),
			}),
			transaction.NewTx(&types.LegacyTx{
				To:    &native.GASAddress,
				Data:  initData,
				Value: big.NewInt(0),
			}),
			transaction.NewTx(&types.LegacyTx{
				To:    &native.ManagementAddress,
				Data:  initData,
				Value: big.NewInt(0),
			}),
		},
	}
	b.RebuildMerkleRoot()

	return b, nil
}

func getConsensusAddress(validators []*keys.PublicKey) (val common.Address, err error) {
	raw, err := keys.PublicKeys(validators).CreateDefaultMultiSigRedeemScript()
	if err != nil {
		return val, err
	}
	return hash.Hash160(raw), nil
}

// headerSliceReverse reverses the given slice of *Header.
func headerSliceReverse(dest []*block.Header) {
	for i, j := 0, len(dest)-1; i < j; i, j = i+1, j-1 {
		dest[i], dest[j] = dest[j], dest[i]
	}
}
