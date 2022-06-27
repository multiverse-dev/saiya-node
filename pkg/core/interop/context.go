package interop

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/multiverse-dev/saiya/pkg/config"
	"github.com/multiverse-dev/saiya/pkg/core/block"
	"github.com/multiverse-dev/saiya/pkg/core/dao"
	"github.com/multiverse-dev/saiya/pkg/core/native"
	"github.com/multiverse-dev/saiya/pkg/core/statedb"
	"github.com/multiverse-dev/saiya/pkg/core/transaction"
	"github.com/multiverse-dev/saiya/pkg/crypto/keys"
	"github.com/multiverse-dev/saiya/pkg/evm"
	"github.com/multiverse-dev/saiya/pkg/evm/vm"
)

type NativeContract interface {
	RequiredGas(ic native.InteropContext, input []byte) uint64
	Run(ic native.InteropContext, input []byte) ([]byte, error)
}

type Chain interface {
	GetConfig() config.ProtocolConfiguration
	Contracts() *native.Contracts
	GetCurrentValidators() ([]*keys.PublicKey, error)
}

// Context represents context in which interops are executed.
type Context struct {
	Chain Chain
	Block *block.Block
	Tx    *transaction.Transaction
	VM    *vm.EVM
	bctx  vm.BlockContext
	sdb   *statedb.StateDB
}

func NewContext(block *block.Block, tx *transaction.Transaction, sdb *statedb.StateDB, chain Chain) (*Context, error) {
	ctx := &Context{
		Chain: chain,
		Block: block,
		Tx:    tx,
		sdb:   sdb,
	}
	ctx.bctx = NewEVMBlockContext(block, chain, chain.GetConfig())
	txContext := NewEVMTxContext(tx.From(), big.NewInt(1))
	ctx.VM = evm.NewEVM(ctx.bctx,
		txContext, sdb, chain.GetConfig(),
		map[common.Address]vm.PrecompiledContract{
			native.DesignationAddress: nativeWrapper{
				nativeContract: chain.Contracts().Designate,
				ic:             ctx,
			},
			native.PolicyAddress: nativeWrapper{
				nativeContract: chain.Contracts().Policy,
				ic:             ctx,
			},
			native.GASAddress: nativeWrapper{
				nativeContract: chain.Contracts().GAS,
				ic:             ctx,
			},
		})
	return ctx, nil
}

func (c Context) Sender() common.Address {
	return c.Tx.From()
}

func (c Context) Natives() *native.Contracts {
	return c.Chain.Contracts()
}

func (c Context) Dao() *dao.Simple {
	return c.sdb.CurrentStore().Simple
}

func (c Context) PersistingBlock() *block.Block {
	return c.Block
}

func (c Context) Coinbase() common.Address {
	return c.bctx.Coinbase
}

func (c Context) Address() common.Address {
	return c.Tx.From()
}
