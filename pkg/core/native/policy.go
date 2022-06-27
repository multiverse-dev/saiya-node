package native

import (
	"encoding/binary"
	"errors"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/multiverse-dev/saiya/pkg/core/dao"
	"github.com/multiverse-dev/saiya/pkg/core/native/nativeids"
	"github.com/multiverse-dev/saiya/pkg/core/native/nativenames"
	"github.com/multiverse-dev/saiya/pkg/core/state"
)

const (
	DefaultFeePerByte uint64 = 1000
	DefaultGasPrice   int64  = 10000000000 //10GWei

	PrefixFeePerByte    byte = 0x01
	PrefixGasPrice      byte = 0x08
	PrefixBlockedAcount byte = 0x11
)

var (
	PolicyAddress      = common.Address(common.BytesToAddress([]byte{nativeids.Policy}))
	ErrAccountBlocked  = errors.New("account blocked")
	ErrContractBlocked = errors.New("contract blocked")
)

type Policy struct {
	state.NativeContract
}

func NewPolicy(cs *Contracts) *Policy {
	return &Policy{
		NativeContract: state.NativeContract{
			Name: nativenames.Policy,
			Contract: state.Contract{
				Address: PolicyAddress,
				Code:    []byte{},
			},
		},
	}
}

func createBlockKey(address common.Address) []byte {
	return makeAddressKey(PrefixBlockedAcount, address)
}

func (p *Policy) initialize(ic InteropContext) error {
	if ic.PersistingBlock() == nil || ic.PersistingBlock().Index != 0 {
		return ErrInitialize
	}
	p.setFeePerByte(ic, DefaultFeePerByte)
	return nil
}

func (p *Policy) setFeePerByte(ic InteropContext, fee uint64) error {
	err := checkCommittee(ic)
	if err != nil {
		return err
	}
	item := make([]byte, 8)
	binary.BigEndian.PutUint64(item, fee)
	ic.Dao().PutStorageItem(p.Address, []byte{PrefixFeePerByte}, item)
	return nil
}

func (p *Policy) GetFeePerByteInternal(s *dao.Simple) uint64 {
	item := s.GetStorageItem(p.Address, []byte{PrefixFeePerByte})
	if item == nil {
		return DefaultFeePerByte
	}
	return binary.BigEndian.Uint64(item)
}

func (p *Policy) setGasPrice(ic InteropContext, gasPrice *big.Int) error {
	err := checkCommittee(ic)
	if err != nil {
		return err
	}
	ic.Dao().PutStorageItem(p.Address, []byte{PrefixGasPrice}, gasPrice.Bytes())
	return nil
}

func (p *Policy) GetGasPriceInternal(s *dao.Simple) *big.Int {
	item := s.GetStorageItem(p.Address, []byte{PrefixGasPrice})
	if item == nil {
		return big.NewInt(DefaultGasPrice)
	}
	return big.NewInt(0).SetBytes(item)
}

func (p *Policy) checkSystem(ic InteropContext, address common.Address) error {
	if ic.PersistingBlock() == nil {
		return ErrNoBlock
	}
	addrs, err := ic.Natives().Designate.GetSysAddresses(ic.Dao(), ic.PersistingBlock().Index)
	if err != nil {
		return err
	}
	for _, account := range addrs {
		if address == account {
			return errors.New("system account")
		}
	}
	return nil
}

func (p *Policy) checkAdmin(ic InteropContext, address common.Address) error {
	if ic.PersistingBlock() == nil {
		return ErrNoBlock
	}
	addrs, err := ic.Natives().Designate.GetAdminAddresses(ic.Dao(), ic.PersistingBlock().Index)
	if err != nil {
		return err
	}
	for _, account := range addrs {
		if address == account {
			return errors.New("admin account")
		}
	}
	return nil
}

func (p *Policy) blockAddress(ic InteropContext, address common.Address) error {
	err := checkCommittee(ic)
	if err != nil {
		return err
	}
	if err := p.checkSystem(ic, address); err != nil {
		return err
	}
	key := createBlockKey(address)
	item := ic.Dao().GetStorageItem(p.Address, key)
	if item != nil {
		return errors.New("already blocked")
	}
	ic.Dao().PutStorageItem(p.Address, key, []byte{1})
	return nil
}

func (p *Policy) IsBlocked(d *dao.Simple, address common.Address) bool {
	key := createBlockKey(address)
	item := d.GetStorageItem(p.Address, key)
	return item != nil
}

func (p *Policy) unblockAddress(ic InteropContext, address common.Address) error {
	err := checkCommittee(ic)
	if err != nil {
		return err
	}
	if err := p.checkSystem(ic, address); err != nil {
		return err
	}
	key := createBlockKey(address)
	item := ic.Dao().GetStorageItem(p.Address, key)
	if item != nil {
		return errors.New("account isn't blocked")
	}
	ic.Dao().DeleteStorageItem(p.Address, key)
	return nil
}

func (p *Policy) RequiredGas(ic InteropContext, input []byte) uint64 {
	if len(input) < 1 {
		return 0
	}
	switch input[0] {
	case 0x00:
		return 0
	case 0x01, 0x02, 0x03:
		return defaultNativeWriteFee
	default:
		return 0
	}
}

func (p *Policy) Run(ic InteropContext, input []byte) ([]byte, error) {
	if len(input) < 1 {
		return nil, ErrEmptyInput
	}
	switch input[0] {
	case 0x00:
		return []byte{}, p.initialize(ic)
	case PrefixFeePerByte:
		fee := binary.BigEndian.Uint64(input[1:])
		p.setFeePerByte(ic, fee)
		return []byte{}, nil
	case PrefixGasPrice:
		gasPrice := big.NewInt(0).SetBytes(input[1:])
		p.setGasPrice(ic, gasPrice)
		return []byte{}, nil
	case PrefixBlockedAcount:
		address := common.BytesToAddress(input[1:])
		return []byte{}, p.blockAddress(ic, address)
	case PrefixBlockedAcount + 1:
		address := common.BytesToAddress(input[1:])
		return []byte{}, p.unblockAddress(ic, address)
	default:
		return nil, ErrInvalidMethodID
	}
}
