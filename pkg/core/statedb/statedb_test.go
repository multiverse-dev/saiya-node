package statedb

import (
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/multiverse-dev/saiya/pkg/config"
	"github.com/multiverse-dev/saiya/pkg/core/dao"
	"github.com/multiverse-dev/saiya/pkg/core/native"
	"github.com/multiverse-dev/saiya/pkg/core/storage"
	"github.com/stretchr/testify/assert"
)

type testNativeContracts struct {
	cs *native.Contracts
}

func newTestNativeContracts() *testNativeContracts {
	return &testNativeContracts{
		cs: native.NewContracts(config.ProtocolConfiguration{
			InitialSAISupply: 100,
		}),
	}
}

func (t *testNativeContracts) Contracts() *native.Contracts {
	return t.cs
}

func TestSnapshot(t *testing.T) {
	addr := common.BytesToAddress([]byte{0x01})
	h := common.BytesToHash([]byte{0x01})
	v1 := common.BytesToHash([]byte{0x01})
	v2 := common.BytesToHash([]byte{0x02})
	ve := common.Hash{}
	cs := newTestNativeContracts()
	ms := storage.NewMemoryStore()
	mc := storage.NewMemCachedStore(ms)
	d := dao.NewSimple(mc)
	sd := NewStateDB(d, cs)
	g := sd.GetState(addr, h)
	assert.Equal(t, ve, g)
	snapshot1 := sd.Snapshot()
	sd.SetState(addr, h, v1)
	g = sd.GetState(addr, h)
	assert.Equal(t, v1, g)
	snapshot2 := sd.Snapshot()
	sd.SetState(addr, h, v2)
	g = sd.GetState(addr, h)
	assert.Equal(t, v2, g)
	sd.RevertToSnapshot(snapshot2)
	g = sd.GetState(addr, h)
	assert.Equal(t, v1, g)
	sd.RevertToSnapshot(snapshot1)
	g = sd.GetState(addr, h)
	assert.Equal(t, ve, g)
}
