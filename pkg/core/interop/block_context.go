package interop

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/multiverse-dev/saiya/pkg/config"
	"github.com/multiverse-dev/saiya/pkg/core/block"
	"github.com/multiverse-dev/saiya/pkg/evm/vm"
)

func NewEVMBlockContext(block *block.Block,
	bc Chain,
	protocolSettings config.ProtocolConfiguration) (bctx vm.BlockContext) {
	validators, err := bc.GetCurrentValidators()
	if err != nil {
		panic(err)
	}
	var coinbase common.Address
	if len(validators) != 0 {
		coinbase = validators[block.PrimaryIndex].Address()
	} else if block.Index != 0 {
		panic("missing validators")
	}
	random := common.BigToHash(big.NewInt(int64(block.Nonce)))
	bctx = vm.BlockContext{
		CanTransfer: func(sdb vm.StateDB, from common.Address, amount *big.Int) bool {
			// block and restrict
			return sdb.GetBalance(from).Cmp(amount) > 0
		},
		Transfer: func(sdb vm.StateDB, from common.Address, to common.Address, amount *big.Int) {
			fromAmount := big.NewInt(0).Neg(amount)
			sdb.AddBalance(from, fromAmount)
			sdb.AddBalance(to, amount)
		},
		Coinbase:    coinbase,
		GasLimit:    uint64(protocolSettings.MaxBlockSystemFee),
		BlockNumber: big.NewInt(int64(block.Index)),
		Time:        big.NewInt(int64(block.Timestamp)),
		Difficulty:  big.NewInt(0),
		BaseFee:     big.NewInt(0),
		Random:      &random,
	}
	return
}
