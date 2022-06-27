package blockchainer

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/multiverse-dev/saiya/pkg/core/state"
	"github.com/multiverse-dev/saiya/pkg/core/storage"
)

// StateRoot represents local state root module.
type StateRoot interface {
	CurrentLocalHeight() uint32
	CurrentLocalStateRoot() common.Hash
	CurrentValidatedHeight() uint32
	FindStates(root common.Hash, prefix, start []byte, max int) ([]storage.KeyValue, error)
	GetState(root common.Hash, key []byte) ([]byte, error)
	GetStateProof(root common.Hash, key []byte) ([][]byte, error)
	GetStateRoot(height uint32) (*state.MPTRoot, error)
}
