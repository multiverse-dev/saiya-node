package blockchainer

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/multiverse-dev/saiya/pkg/config"
	"github.com/multiverse-dev/saiya/pkg/core/block"
	"github.com/multiverse-dev/saiya/pkg/core/filters"
	"github.com/multiverse-dev/saiya/pkg/core/interop"
	"github.com/multiverse-dev/saiya/pkg/core/mempool"
	"github.com/multiverse-dev/saiya/pkg/core/native"
	"github.com/multiverse-dev/saiya/pkg/core/state"
	"github.com/multiverse-dev/saiya/pkg/core/transaction"
	"github.com/multiverse-dev/saiya/pkg/crypto/hash"
	"github.com/multiverse-dev/saiya/pkg/crypto/keys"
)

// Blockchainer is an interface that abstract the implementation
// of the blockchain.
type Blockchainer interface {
	ApplyPolicyToTxSet([]*transaction.Transaction) []*transaction.Transaction
	AddBlock(block *block.Block) error
	AddHeaders(...*block.Header) error
	BlockHeight() uint32
	GetConfig() config.ProtocolConfiguration
	Close()
	Contracts() *native.Contracts
	IsTxStillRelevant(t *transaction.Transaction, txpool *mempool.Pool, isPartialTx bool) bool
	HeaderHeight() uint32
	GetBlock(hash common.Hash, full bool) (*block.Block, error)
	GetCommittee() (keys.PublicKeys, error)
	GetCommitteeAddress() (common.Address, error)
	GetContractState(hash common.Address) *state.Contract
	IsBlocked(common.Address) bool
	GetHeaderHash(int) common.Hash
	GetHeader(hash common.Hash) (*block.Header, error)
	CurrentHeaderHash() common.Hash
	CurrentBlockHash() common.Hash
	HasBlock(common.Hash) bool
	HasTransaction(common.Hash) bool
	IsExtensibleAllowed(common.Address) bool
	GetReceipt(common.Hash) (*types.Receipt, error)
	GetNativeContractScriptHash(string) (common.Address, error)
	GetNatives() []state.NativeContract
	GetValidators(uint32) ([]*keys.PublicKey, error)
	GetCurrentValidators() ([]*keys.PublicKey, error)
	GetStateModule() StateRoot
	GetStorageItem(hash common.Address, key []byte) state.StorageItem
	GetStorageItems(hash common.Address) ([]state.StorageItemWithKey, error)
	GetTestVM(tx *transaction.Transaction, b *block.Block) (*interop.Context, error)
	GetTransaction(common.Hash) (*transaction.Transaction, uint32, error)
	mempool.Feer // fee interface
	ManagementContractAddress() common.Address
	PoolTx(t *transaction.Transaction, pools ...*mempool.Pool) error
	PoolTxWithData(t *transaction.Transaction, data interface{}, mp *mempool.Pool, feer mempool.Feer, verificationFunction func(t *transaction.Transaction, data interface{}) error) error
	SubscribeForBlocks(ch chan<- *block.Block)
	SubscribeForExecutions(ch chan<- *types.Receipt)
	SubscribeForNotifications(ch chan<- *types.Log)
	SubscribeForTransactions(ch chan<- *transaction.Transaction)
	VerifyTx(*transaction.Transaction) error
	VerifyWitness(common.Address, hash.Hashable, *transaction.Witness) error
	GetMemPool() *mempool.Pool
	UnsubscribeFromBlocks(ch chan<- *block.Block)
	UnsubscribeFromExecutions(ch chan<- *types.Receipt)
	UnsubscribeFromNotifications(ch chan<- *types.Log)
	UnsubscribeFromTransactions(ch chan<- *transaction.Transaction)
	GetFeePerByte() uint64
	GetGasPrice() *big.Int
	GetNonce(addr common.Address) uint64
	GetLogs(filter *filters.LogFilter) ([]*types.Log, error)
}
