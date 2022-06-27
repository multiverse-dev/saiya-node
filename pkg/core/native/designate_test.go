package native

import (
	"encoding/binary"
	"encoding/hex"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/multiverse-dev/saiya/pkg/config"
	"github.com/multiverse-dev/saiya/pkg/core/block"
	"github.com/multiverse-dev/saiya/pkg/core/dao"
	"github.com/multiverse-dev/saiya/pkg/core/native/noderoles"
	"github.com/multiverse-dev/saiya/pkg/core/storage"
	"github.com/multiverse-dev/saiya/pkg/crypto/keys"
	"github.com/stretchr/testify/assert"
)

func TestEndian(t *testing.T) {
	var index uint32 = 1
	b := make([]byte, 4)
	binary.LittleEndian.PutUint32(b, index)
	t.Log(hex.EncodeToString(b))
}

type interopContext struct {
	D     *dao.Simple
	S     common.Address
	Index uint32
}

func (ic interopContext) Sender() common.Address {
	return ic.S
}

func (ic interopContext) Natives() *Contracts {
	return &Contracts{}
}

func (ic interopContext) Dao() *dao.Simple {
	return ic.D
}

func (ic interopContext) PersistingBlock() *block.Block {
	return &block.Block{
		Header: block.Header{
			Index: ic.Index,
		},
	}
}

func TestCommitteeRole(t *testing.T) {
	dao := dao.NewSimple(storage.NewMemoryStore())
	des := NewDesignate(config.ProtocolConfiguration{
		StandbyCommittee: []string{
			"023c4d39a3fd2150407a9d4654430cdce0464eccaaf739eea79d63e2862f989ee6",
		},
	})
	k1, err := keys.NewPublicKeyFromString("023c4d39a3fd2150407a9d4654430cdce0464eccaaf739eea79d63e2862f989ee6")
	assert.NoError(t, err)
	ic := interopContext{
		D: dao,
	}
	err = des.initialize(ic)
	assert.NoError(t, err)
	ks, err := des.GetDesignatedByRole(dao, noderoles.Committee, 1)
	assert.NoError(t, err)
	assert.Equal(t, 1, ks.Len())
	assert.Equal(t, "023c4d39a3fd2150407a9d4654430cdce0464eccaaf739eea79d63e2862f989ee6", hex.EncodeToString(ks[0].Bytes()))
	// - change one committee -
	k2, err := keys.NewPublicKeyFromString("0218cbadb9db833a6b7432a920b6bdb6b822eb2df0d59cfc5d9d590d5dfd97fef4")
	assert.NoError(t, err)
	s, err := des.GetCommitteeAddress(dao, 1)
	assert.NoError(t, err)
	ic.S = s
	ic.Index = 1
	err = des.designateAsRole(ic, noderoles.Committee, keys.PublicKeys{k2})
	assert.NoError(t, err)
	ks, err = des.GetDesignatedByRole(dao, noderoles.Committee, 1)
	assert.NoError(t, err)
	assert.Equal(t, "023c4d39a3fd2150407a9d4654430cdce0464eccaaf739eea79d63e2862f989ee6", hex.EncodeToString(ks[0].Bytes()))
	ks, err = des.GetDesignatedByRole(dao, noderoles.Committee, 2)
	assert.NoError(t, err)
	assert.Equal(t, "0218cbadb9db833a6b7432a920b6bdb6b822eb2df0d59cfc5d9d590d5dfd97fef4", hex.EncodeToString(ks[0].Bytes()))
	// - - - - - - - - - - - - -
	// - change committee to 2 from 1 -
	s, err = des.GetCommitteeAddress(dao, 101)
	assert.NoError(t, err)
	ic.S = s
	ic.Index = 2
	err = des.designateAsRole(ic, noderoles.Committee, keys.PublicKeys{k1, k2})
	assert.NoError(t, err)
	ks, err = des.GetDesignatedByRole(dao, noderoles.Committee, 2)
	assert.NoError(t, err)
	assert.Equal(t, 1, ks.Len())
	ks, err = des.GetDesignatedByRole(dao, noderoles.Committee, 3)
	assert.NoError(t, err)
	assert.Equal(t, 2, ks.Len())
}

func TestValidatorRole(t *testing.T) {
	dao := dao.NewSimple(storage.NewMemoryStore())
	des := NewDesignate(config.ProtocolConfiguration{
		StandbyValidators: []string{
			"023c4d39a3fd2150407a9d4654430cdce0464eccaaf739eea79d63e2862f989ee6",
		},
		StandbyCommittee: []string{
			"023c4d39a3fd2150407a9d4654430cdce0464eccaaf739eea79d63e2862f989ee6",
		},
	})
	k1, err := keys.NewPublicKeyFromString("023c4d39a3fd2150407a9d4654430cdce0464eccaaf739eea79d63e2862f989ee6")
	assert.NoError(t, err)
	ic := interopContext{
		D: dao,
	}
	err = des.initialize(ic)
	assert.NoError(t, err)
	ks, err := des.GetDesignatedByRole(dao, noderoles.Validator, 1)
	assert.NoError(t, err)
	assert.Equal(t, 1, ks.Len())
	assert.Equal(t, "023c4d39a3fd2150407a9d4654430cdce0464eccaaf739eea79d63e2862f989ee6", hex.EncodeToString(ks[0].Bytes()))
	// - change one validator -
	k2, err := keys.NewPublicKeyFromString("0218cbadb9db833a6b7432a920b6bdb6b822eb2df0d59cfc5d9d590d5dfd97fef4")
	assert.NoError(t, err)
	s, err := des.GetCommitteeAddress(dao, 1)
	assert.NoError(t, err)
	ic.S = s
	ic.Index = 1
	err = des.designateAsRole(ic, noderoles.Validator, keys.PublicKeys{k2})
	assert.NoError(t, err)
	ks, err = des.GetDesignatedByRole(dao, noderoles.Validator, 2)
	assert.NoError(t, err)
	assert.Equal(t, "023c4d39a3fd2150407a9d4654430cdce0464eccaaf739eea79d63e2862f989ee6", hex.EncodeToString(ks[0].Bytes()))
	ks, err = des.GetDesignatedByRole(dao, noderoles.Validator, 3)
	assert.NoError(t, err)
	assert.Equal(t, "0218cbadb9db833a6b7432a920b6bdb6b822eb2df0d59cfc5d9d590d5dfd97fef4", hex.EncodeToString(ks[0].Bytes()))
	// - - - - - - - - - - - - -
	// - change committee to 2 from 1 -
	s, err = des.GetCommitteeAddress(dao, 3)
	assert.NoError(t, err)
	ic.S = s
	ic.Index = 3
	err = des.designateAsRole(ic, noderoles.Validator, keys.PublicKeys{k1, k2})
	assert.NoError(t, err)
	ks, err = des.GetDesignatedByRole(dao, noderoles.Validator, 4)
	assert.NoError(t, err)
	assert.Equal(t, 1, ks.Len())
	ks, err = des.GetDesignatedByRole(dao, noderoles.Validator, 5)
	assert.NoError(t, err)
	assert.Equal(t, 2, ks.Len())
}
