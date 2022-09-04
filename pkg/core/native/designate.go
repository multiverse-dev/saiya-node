package native

import (
	"encoding/binary"
	"errors"
	"math"
	"sort"
	"sync"

	"github.com/ethereum/go-ethereum/common"
	"github.com/multiverse-dev/saiya/pkg/config"
	"github.com/multiverse-dev/saiya/pkg/core/block"
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
	DesignationAddress       common.Address = common.Address(common.BytesToAddress([]byte{nativeids.Designation}))
	ErrInvalidCommitteeCount                = errors.New("designated committee can't be less than validators count")
)

type Designate struct {
	state.NativeContract
	StandbyCommittee keys.PublicKeys
	ValidatorsCount  int
	cache            *DesignationCache
}

type roleData struct {
	nodes  keys.PublicKeys
	addr   common.Address
	height uint32
}

type DesignationCache struct {
	mutex            sync.Mutex
	rolesChangedFlag bool
	committee        roleData
	stateVals        roleData
}

func NewDesignate(cfg config.ProtocolConfiguration) *Designate {
	standbyCommittee, err := keys.NewPublicKeysFromStrings(cfg.StandbyCommittee)
	if err != nil {
		panic("invalid standy committee config")
	}
	standbyCommittee = standbyCommittee.Unique()
	sort.Sort(standbyCommittee)
	d := &Designate{
		NativeContract: state.NativeContract{
			Name: nativenames.Designation,
			Contract: state.Contract{
				Address:  DesignationAddress,
				CodeHash: hash.Keccak256(DesignationAddress[:]),
				Code:     DesignationAddress[:],
			},
		},
		StandbyCommittee: standbyCommittee,
		ValidatorsCount:  cfg.ValidatorsCount,
		cache:            &DesignationCache{},
	}
	designateAbi, contractCalls, err := constructAbi(d)
	if err != nil {
		panic(err)
	}
	d.Abi = *designateAbi
	d.ContractCalls = contractCalls
	return d
}

func (d *Designate) UpdateCache(s *dao.Simple) error {
	err := d.updateCachedRoleData(d.cache, s, noderoles.Committee)
	if err != nil {
		return err
	}
	return d.updateCachedRoleData(d.cache, s, noderoles.StateValidator)
}

func (d *Designate) PostPersist(s *dao.Simple, _ *block.Block) error {
	return d.UpdateCache(s)
}

func (d *Designate) updateCachedRoleData(cache *DesignationCache, s *dao.Simple, r noderoles.Role) error {
	d.cache.mutex.Lock()
	defer d.cache.mutex.Unlock()
	var v *roleData
	switch r {
	case noderoles.StateValidator:
		v = &cache.stateVals
	case noderoles.Committee:
		v = &cache.committee
	}
	nodeKeys, height, err := d.GetDesignatedByRoleFromStorage(s, r, math.MaxUint32)
	if err != nil {
		return err
	}
	v.nodes = nodeKeys
	addr, err := addressFromNodes(r, nodeKeys)
	if err != nil {
		return err
	}
	v.addr = addr
	v.height = height
	cache.rolesChangedFlag = true
	return nil
}

func addressFromNodes(r noderoles.Role, nodes keys.PublicKeys) (common.Address, error) {
	switch r {
	case noderoles.Committee:
		return createCommitteeAddress(nodes)
	case noderoles.StateValidator:
		if nodes.Len() == 0 {
			return common.Address{}, nil
		}
		script, err := nodes.CreateDefaultMultiSigRedeemScript()
		if err != nil {
			return common.Address{}, err
		}
		return hash.Hash160(script), nil
	default:
		return common.Address{}, errors.New("invalid role")
	}
}

func (d *Designate) ContractCall_initialize(ic InteropContext) error {
	if ic.PersistingBlock() == nil || ic.PersistingBlock().Index != 0 {
		return ErrInitialize
	}
	ic.Dao().PutStorageItem(d.Address, createRoleKey(noderoles.Committee, 0), d.StandbyCommittee.Bytes())
	log(ic, d.Address, d.StandbyCommittee.Bytes(), d.Abi.Events["initialize"].ID, common.BytesToHash([]byte{byte(noderoles.Committee)}))
	return nil
}

func (d *Designate) ContractCall_designateAsRole(ic InteropContext, role byte, rawPks []byte) error {
	r := noderoles.Role(role)
	pks := new(keys.PublicKeys)
	err := pks.DecodeBytes(rawPks)
	if err != nil {
		return err
	}
	err = d.designateAsRole(ic, r, *pks)
	if err == nil {
		log(ic, d.Address, rawPks, d.Abi.Events["designateAsRole"].ID, common.BytesToHash([]byte{role}))
	}
	return err
}

func (d *Designate) checkCommittee(ic InteropContext) error {
	if ic.PersistingBlock() == nil {
		return ErrNoBlock
	}
	committeeAddress, err := d.GetCommitteeAddress(ic.Dao(), ic.PersistingBlock().Index)
	if err != nil {
		return err
	}
	if ic.Sender() != committeeAddress {
		return ErrInvalidSender
	}
	return nil
}

func (d *Designate) designateAsRole(ic InteropContext, r noderoles.Role, keys keys.PublicKeys) error {
	if !noderoles.IsValid(r) {
		return ErrInvalidRole
	}
	err := d.checkCommittee(ic)
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
	if r == noderoles.Committee && ks.Len() < d.ValidatorsCount {
		return ErrInvalidCommitteeCount
	}
	index := ic.PersistingBlock().Index + 1
	if r == noderoles.Committee {
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

func createRoleKey(role noderoles.Role, index uint32) []byte {
	key := make([]byte, 5)
	key[0] = byte(role)
	binary.BigEndian.PutUint32(key[1:], index)
	return key
}

func (d *Designate) GetDesignatedByRole(s *dao.Simple, r noderoles.Role, index uint32) (keys.PublicKeys, uint32, error) {
	if !noderoles.IsValid(r) {
		return nil, 0, ErrInvalidRole
	}
	d.cache.mutex.Lock()
	defer d.cache.mutex.Unlock()
	var val roleData
	switch r {
	case noderoles.Committee:
		val = d.cache.committee
	case noderoles.StateValidator:
		val = d.cache.stateVals
	default:
		return nil, 0, ErrInvalidRole
	}
	if val.height <= index {
		return val.nodes.Copy(), val.height, nil
	}
	return d.GetDesignatedByRoleFromStorage(s, r, index)
}

func (d *Designate) GetDesignatedByRoleFromStorage(s *dao.Simple, r noderoles.Role, index uint32) (keys.PublicKeys, uint32, error) {
	kvs, err := s.GetStorageItemsWithPrefix(d.Address, []byte{byte(r)})
	if err != nil {
		return nil, 0, err
	}
	var (
		ks     = keys.PublicKeys{}
		resSi  state.StorageItem
		height uint32
	)
	for i := len(kvs) - 1; i >= 0; i-- {
		kv := kvs[i]
		if len(kv.Key) < 4 {
			continue
		}
		siInd := binary.BigEndian.Uint32(kv.Key)
		if siInd <= index {
			height = siInd
			resSi = kv.Item
			break
		}
	}
	if resSi != nil {
		err := ks.DecodeBytes(resSi)
		if err != nil {
			return nil, 0, err
		}
	}
	return ks, height, nil
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
	nodes, _, err := d.GetDesignatedByRole(s, noderoles.Committee, index)
	return nodes, err
}

func (d *Designate) GetCommitteeAddress(s *dao.Simple, index uint32) (common.Address, error) {
	committees, err := d.GetCommitteeMembers(s, index)
	if err != nil {
		return common.Address{}, err
	}
	return createCommitteeAddress(committees)
}

func (d *Designate) GetValidators(s *dao.Simple, index uint32) (keys.PublicKeys, error) {
	committees, err := d.GetCommitteeMembers(s, index)
	if err != nil {
		return keys.PublicKeys{}, err
	}
	if committees.Len() < d.ValidatorsCount {
		panic(ErrInvalidCommitteeCount)
	}
	return keys.PublicKeys(committees[:d.ValidatorsCount]), nil
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

func (d *Designate) GetSysAddresses(s *dao.Simple, index uint32) ([]common.Address, error) {
	accounts, err := d.GetCommitteeMembers(s, index)
	if err != nil {
		return nil, err
	}
	addrs := make([]common.Address, len(accounts))
	for i, account := range accounts {
		addrs[i] = account.Address()
	}
	if accounts.Len() > 1 {
		committeeAddress, err := createCommitteeAddress(accounts)
		if err != nil {
			return nil, err
		}
		addrs = append(addrs, committeeAddress)
	}
	script, err := keys.PublicKeys(accounts[:d.ValidatorsCount]).CreateDefaultMultiSigRedeemScript()
	if err != nil {
		return nil, err
	}
	addrs = append(addrs, hash.Hash160(script))
	return addrs, nil
}

func (d *Designate) RequiredGas(ic InteropContext, input []byte) uint64 {
	if len(input) < 4 {
		return 0
	}
	method, err := d.Abi.MethodById(input[:4])
	if err != nil {
		return 0
	}
	switch method.Name {
	case "initialize":
		return 0
	case "designateAsRole":
		return defaultNativeWriteFee
	default:
		return 0
	}
}

func (d *Designate) Run(ic InteropContext, input []byte) ([]byte, error) {
	return contractCall(d, &d.NativeContract, ic, input)
}
