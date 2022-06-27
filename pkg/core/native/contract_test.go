package native

import (
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/multiverse-dev/saiya/pkg/core/dao"
	"github.com/multiverse-dev/saiya/pkg/core/storage"
	"github.com/stretchr/testify/assert"
)

func TestKey(t *testing.T) {
	d := dao.NewSimple(storage.NewMemoryStore())
	ledger := NewLedger()
	management := NewManagement()
	t.Logf("%s, %s\n", LedgerAddress, ManagementAddress)
	ledger.SetNonce(d, common.Address{}, 1)
	code := management.GetCode(d, common.Address{})
	assert.Nil(t, code)
}
