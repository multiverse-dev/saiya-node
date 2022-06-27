package native

import (
	"encoding/binary"
	"sort"

	"github.com/ethereum/go-ethereum/common"
	"github.com/multiverse-dev/saiya/pkg/config"
	"github.com/multiverse-dev/saiya/pkg/core/dao"
	"github.com/multiverse-dev/saiya/pkg/core/native/nativeids"
	"github.com/multiverse-dev/saiya/pkg/core/native/nativenames"
	"github.com/multiverse-dev/saiya/pkg/core/native/noderoles"
	"github.com/multiverse-dev/saiya/pkg/core/state"
	"github.com/multiverse-dev/saiya/pkg/crypto/hash"
	"github.com/multiverse-dev/saiya/pkg/crypto/keys"
)

const (
	MaxNodeCount = 21
)

var (
	DesignationAddress common.Address = common.Address(common.BytesToAddress([]byte{nativeids.Designation}))
)

type Designate struct {
	state.NativeContract
	StandbyCommittee keys.PublicKeys
	StandyValidators keys.PublicKeys
}

func NewDesignate(cfg config.ProtocolConfiguration) *Designate {
	standbyCommittee, err := keys.NewPublicKeysFromStrings(cfg.StandbyCommittee)
	if err != nil {
		panic("invalid standy committee config")
	}
	standbyCommittee = standbyCommittee.Unique()
	sort.Sort(standbyCommittee)
	standbyValidators, err := keys.NewPublicKeysFromStrings(cfg.StandbyValidators)
	if err != nil {
		panic("invalid standy validators config")
	}
	standbyValidators = standbyValidators.Unique()
	sort.Sort(standbyValidators)
	return &Designate{
		NativeContract: state.NativeContract{
			Name: nativenames.Designation,
			Contract: state.Contract{
				Address: DesignationAddress,
			},
		},
		StandbyCommittee: standbyCommittee,
		StandyValidators: standbyValidators,
	}
}

func (d *Designate) initialize(ic InteropContext) error {
	if ic.PersistingBlock() == nil || ic.PersistingBlock().Index != 0 {
		return ErrInitialize
	}
	ic.Dao().PutStorageItem(d.Address, createRoleKey(noderoles.Committee, 0), d.StandbyCommittee.Bytes())
	ic.Dao().PutStorageItem(d.Address, createRoleKey(noderoles.Validator, 0), d.StandyValidators.Bytes())
	return nil
}

func createRoleKey(role noderoles.Role, index uint32) []byte {
	key := make([]byte, 5)
	key[0] = byte(role)
	binary.BigEndian.PutUint32(key[1:], index)
	return key
}

func (d *Designate) GetDesignatedByRole(s *dao.Simple, r noderoles.Role, index uint32) (keys.PublicKeys, error) {
	if !noderoles.IsValid(r) {
		return nil, ErrInvalidRole
	}
	kvs, err := s.GetStorageItemsWithPrefix(d.Address, []byte{byte(r)})
	if err != nil {
		return nil, err
	}
	var (
		ks    = keys.PublicKeys{}
		resSi state.StorageItem
	)
	for i := len(kvs) - 1; i >= 0; i-- {
		kv := kvs[i]
		if len(kv.Key) < 4 {
			continue
		}
		siInd := binary.BigEndian.Uint32(kv.Key)
		if siInd <= index {
			resSi = kv.Item
			break
		}
	}
	if resSi != nil {
		err := ks.DecodeBytes(resSi)
		if err != nil {
			return nil, err
		}
	}
	return ks, nil
}

func createCommitteeAddress(keys keys.PublicKeys) (common.Address, error) {
	if keys.Len() == 0 {
		return common.Address{}, ErrEmptyNodeList
	}
	if keys.Len() == 1 {
		return keys[0].Address(), nil
	}
	script, err := keys.CreateMajorityMultiSigRedeemScript()
	if err != nil {
		return common.Address{}, err
	}
	return hash.Hash160(script), nil
}

func (d *Designate) GetCommitteeMembers(s *dao.Simple, index uint32) (keys.PublicKeys, error) {
	return d.GetDesignatedByRole(s, noderoles.Committee, index)
}

func (d *Designate) GetCommitteeAddress(s *dao.Simple, index uint32) (common.Address, error) {
	committees, err := d.GetCommitteeMembers(s, index)
	if err != nil {
		return common.Address{}, err
	}
	return createCommitteeAddress(committees)
}

func (d *Designate) GetValidators(s *dao.Simple, index uint32) (keys.PublicKeys, error) {
	return d.GetDesignatedByRole(s, noderoles.Validator, index)
}

func (d *Designate) GetValidatorAddress(s *dao.Simple, index uint32) (common.Address, error) {
	validators, err := d.GetValidators(s, index)
	if err != nil {
		return common.Address{}, err
	}
	script, err := validators.CreateDefaultMultiSigRedeemScript()
	if err != nil {
		return common.Address{}, err
	}
	return hash.Hash160(script), nil
}

func (d *Designate) GetSysAccounts(s *dao.Simple, index uint32) (keys.PublicKeys, error) {
	committee, err := d.GetCommitteeMembers(s, index)
	if err != nil {
		return nil, err
	}
	validators, err := d.GetValidators(s, index)
	if err != nil {
		return nil, err
	}
	return append(committee, validators...), nil
}

func (d *Designate) GetSysAddresses(s *dao.Simple, index uint32) ([]common.Address, error) {
	accounts, err := d.GetSysAccounts(s, index)
	if err != nil {
		return nil, err
	}
	addrs := make([]common.Address, len(accounts))
	for i, account := range accounts {
		addrs[i] = account.Address()
	}
	return addrs, nil
}

func (d *Designate) GetAdminAccounts(s *dao.Simple, index uint32) (keys.PublicKeys, error) {
	return d.GetCommitteeMembers(s, index)
}

func (d *Designate) GetAdminAddresses(s *dao.Simple, index uint32) ([]common.Address, error) {
	accounts, err := d.GetCommitteeMembers(s, index)
	if err != nil {
		return nil, err
	}
	addrs := make([]common.Address, len(accounts)+1)
	for i, account := range accounts {
		addrs[i] = account.Address()
	}
	committeeAddress, err := createCommitteeAddress(accounts)
	if err != nil {
		return nil, err
	}
	addrs[len(addrs)-1] = committeeAddress
	return addrs, nil
}

func (d *Designate) designateAsRole(ic InteropContext, r noderoles.Role, keys keys.PublicKeys) error {
	if !noderoles.IsValid(r) {
		return ErrInvalidRole
	}
	err := checkCommittee(ic)
	if err != nil {
		return err
	}
	ks := keys.Unique()
	if ks.Len() == 0 {
		return ErrEmptyNodeList
	}
	if ks.Len() > MaxNodeCount {
		return ErrLargeNodeList
	}
	index := ic.PersistingBlock().Index + 1
	if r == noderoles.Validator {
		index += 1
	}
	committee, err := d.GetCommitteeAddress(ic.Dao(), index)
	if err != nil {
		return err
	}
	if ic.Sender() != committee {
		return ErrInvalidSender
	}
	sort.Sort(ks)
	ic.Dao().PutStorageItem(d.Address, createRoleKey(r, index), ks.Bytes())
	return nil
}

func (g *Designate) designateRawAsRole(ic InteropContext, input []byte) error {
	if len(input) < 2 {
		return ErrInvalidInput
	}
	r := noderoles.Role(input[0])
	pks := keys.PublicKeys{}
	err := pks.DecodeBytes(input[1:])
	if err != nil {
		return err
	}
	return g.designateAsRole(ic, r, pks)
}

func (g *Designate) RequiredGas(ic InteropContext, input []byte) uint64 {
	if len(input) < 1 {
		return 0
	}
	switch input[0] {
	case 0x00:
		return 0
	case 0x01:
		return defaultNativeWriteFee
	default:
		return 0
	}
}

func (g *Designate) Run(ic InteropContext, input []byte) ([]byte, error) {
	if len(input) < 1 {
		return nil, ErrEmptyInput
	}
	switch input[0] {
	case 0x00:
		return []byte{}, g.initialize(ic)
	case 0x01:
		return []byte{}, g.designateRawAsRole(ic, input[1:])
	default:
		return nil, ErrInvalidMethodID
	}
}
