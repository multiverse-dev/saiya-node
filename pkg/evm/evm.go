package evm

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/params"
	"github.com/multiverse-dev/saiya/pkg/config"
	"github.com/multiverse-dev/saiya/pkg/evm/vm"
)

const TestGas uint64 = 2000000000

func NewEVM(bctx vm.BlockContext,
	tctx vm.TxContext,
	sdb vm.StateDB,
	protocolSettings config.ProtocolConfiguration,
	extraPrecompiles map[common.Address]vm.PrecompiledContract) *vm.EVM {
	evm := vm.NewEVM(bctx, tctx, sdb, &params.ChainConfig{
		ChainID:             big.NewInt(int64(protocolSettings.ChainID)),
		HomesteadBlock:      big.NewInt(0),
		DAOForkBlock:        nil,
		DAOForkSupport:      true,
		EIP150Block:         big.NewInt(0),
		EIP155Block:         big.NewInt(0),
		EIP158Block:         big.NewInt(0),
		ByzantiumBlock:      big.NewInt(0),
		ConstantinopleBlock: big.NewInt(0),
		PetersburgBlock:     big.NewInt(0),
		IstanbulBlock:       big.NewInt(0),
		MuirGlacierBlock:    big.NewInt(0),
		BerlinBlock:         big.NewInt(0),
		LondonBlock:         big.NewInt(0),
		Ethash:              new(params.EthashConfig),
	}, vm.Config{}, extraPrecompiles)
	return evm
}
