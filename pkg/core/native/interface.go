package native

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/multiverse-dev/saiya/pkg/core/block"
	"github.com/multiverse-dev/saiya/pkg/core/dao"
)

type InteropContext interface {
	Sender() common.Address
	Natives() *Contracts
	Dao() *dao.Simple
	PersistingBlock() *block.Block
}
