package native

import (
	"errors"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/multiverse-dev/saiya/pkg/core/dao"
	"github.com/multiverse-dev/saiya/pkg/core/native/nativeids"
	"github.com/multiverse-dev/saiya/pkg/core/native/nativenames"
	"github.com/multiverse-dev/saiya/pkg/core/state"
	"github.com/multiverse-dev/saiya/pkg/io"
)

const (
	prefixAccount = 20
	SAIDecimal    = 18
)

var (
	SAIAddress     common.Address = common.Address(common.BytesToAddress([]byte{nativeids.SAI}))
	totalSupplyKey                = []byte{11}
)

type SAI struct {
	state.NativeContract
	symbol        string
	decimals      int64
	initialSupply uint64
}

func NewSAI(init uint64) *SAI {
	g := &SAI{
		NativeContract: state.NativeContract{
			Name: nativenames.SAI,
			Contract: state.Contract{
				Address: SAIAddress,
				Code:    []byte{},
			},
		},
		initialSupply: init,
	}

	g.symbol = "SAI"
	g.decimals = SAIDecimal

	return g
}

func makeAccountKey(h common.Address) []byte {
	return makeAddressKey(prefixAccount, h)
}

func (g *SAI) initialize(ic InteropContext) error {
	if ic.PersistingBlock() == nil || ic.PersistingBlock().Index != 0 {
		return ErrInitialize
	}
	addr, err := ic.Natives().Designate.GetCommitteeAddress(ic.Dao(), 0)
	if err != nil {
		return err
	}
	wei := big.NewInt(1).Exp(big.NewInt(10), big.NewInt(SAIDecimal), nil)
	total := big.NewInt(1).Mul(big.NewInt(int64(g.initialSupply)), wei)
	return g.addTokens(ic.Dao(), addr, total)
}

func (g *SAI) increaseBalance(gs *GasState, amount *big.Int) error {
	if amount.Sign() == -1 && gs.Balance.CmpAbs(amount) == -1 {
		return errors.New("insufficient funds")
	}
	gs.Balance.Add(gs.Balance, amount)
	return nil
}

func (g *SAI) getTotalSupply(s *dao.Simple) *big.Int {
	si := s.GetStorageItem(g.Address, totalSupplyKey)
	if si == nil {
		return nil
	}
	return big.NewInt(0).SetBytes(si)
}

func (g *SAI) saveTotalSupply(s *dao.Simple, supply *big.Int) {
	s.PutStorageItem(g.Address, totalSupplyKey, supply.Bytes())
}

func (g *SAI) getGasState(s *dao.Simple, key []byte) (*GasState, error) {
	si := s.GetStorageItem(g.Address, key)
	if si == nil {
		return nil, nil
	}
	gs := &GasState{}
	err := io.FromByteArray(gs, si)
	if err != nil {
		return nil, err
	}
	return gs, nil
}

func (g *SAI) putGasState(s *dao.Simple, key []byte, gs *GasState) error {
	data, err := io.ToByteArray(gs)
	if err != nil {
		return err
	}
	s.PutStorageItem(g.Address, key, data)
	return nil
}

func (g *SAI) addTokens(s *dao.Simple, h common.Address, amount *big.Int) error {
	if amount.Sign() == 0 {
		return nil
	}
	key := makeAccountKey(h)
	gs, err := g.getGasState(s, key)
	if err != nil {
		return err
	}
	ngs := gs
	if ngs == nil {
		ngs = &GasState{
			Balance: big.NewInt(0),
		}
	}
	if err := g.increaseBalance(ngs, amount); err != nil {
		return err
	}
	if gs != nil && ngs.Balance.Sign() == 0 {
		s.DeleteStorageItem(g.Address, key)
	} else {
		err = g.putGasState(s, key, ngs)
		if err != nil {
			return err
		}
	}
	supply := g.getTotalSupply(s)
	if supply == nil {
		supply = big.NewInt(0)
	}
	supply.Add(supply, amount)
	g.saveTotalSupply(s, supply)
	return nil
}

func (g *SAI) AddBalance(s *dao.Simple, h common.Address, amount *big.Int) {
	g.addTokens(s, h, amount)
}

func (g *SAI) SubBalance(s *dao.Simple, h common.Address, amount *big.Int) {
	neg := big.NewInt(0)
	neg.Neg(amount)
	g.addTokens(s, h, neg)
}

func (g *SAI) balanceFromBytes(si *state.StorageItem) (*big.Int, error) {
	acc := new(GasState)
	err := io.FromByteArray(acc, *si)
	if err != nil {
		return nil, err
	}
	return acc.Balance, err
}

func (g *SAI) GetBalance(d *dao.Simple, h common.Address) *big.Int {
	key := makeAccountKey(h)
	si := d.GetStorageItem(g.Address, key)
	if si == nil {
		return big.NewInt(0)
	}
	balance, err := g.balanceFromBytes(&si)
	if err != nil {
		panic(fmt.Errorf("can not deserialize balance state: %w", err))
	}
	return balance
}

func (g *SAI) RequiredGas(ic InteropContext, input []byte) uint64 {
	if len(input) < 1 {
		return 0
	}
	switch input[0] {
	case 0x00:
		return 0
	default:
		return 0
	}
}

func (g *SAI) Run(ic InteropContext, input []byte) ([]byte, error) {
	if len(input) < 1 {
		return nil, ErrEmptyInput
	}
	switch input[0] {
	case 0x00:
		return nil, g.initialize(ic)
	default:
		return nil, ErrInvalidMethodID
	}
}

type GasState struct {
	Balance *big.Int
}

func (g *GasState) EncodeBinary(bw *io.BinWriter) {
	bw.WriteVarBytes(g.Balance.Bytes())
}

func (g *GasState) DecodeBinary(br *io.BinReader) {
	g.Balance = big.NewInt(0).SetBytes(br.ReadVarBytes())
}
