package native

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/multiverse-dev/saiya/pkg/core/block"
	"github.com/multiverse-dev/saiya/pkg/core/dao"
)

type InteropContext interface {
	Log(*types.Log)
	Sender() common.Address
	Dao() *dao.Simple
	PersistingBlock() *block.Block
}
