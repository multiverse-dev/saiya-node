package interop

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/multiverse-dev/saiya/pkg/evm/vm"
)

func NewEVMTxContext(sender common.Address, gasPrice *big.Int) vm.TxContext {
	return vm.TxContext{
		Origin:   common.Address(sender),
		GasPrice: gasPrice,
	}
}
