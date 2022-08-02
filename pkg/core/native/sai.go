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
	"github.com/multiverse-dev/saiya/pkg/crypto/hash"
	"github.com/multiverse-dev/saiya/pkg/dbft/block"
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
	cs            *Contracts
	symbol        string
	decimals      int64
	initialSupply uint64
}

func NewSAI(cs *Contracts, init uint64) *SAI {
	g := &SAI{
		NativeContract: state.NativeContract{
			Name: nativenames.SAI,
			Contract: state.Contract{
				Address:  SAIAddress,
				CodeHash: hash.Keccak256(SAIAddress[:]),
				Code:     SAIAddress[:],
			},
		},
		cs:            cs,
		initialSupply: init,
	}

	g.symbol = "SAI"
	g.decimals = SAIDecimal
	gasAbi, contractCalls, err := constructAbi(g)
	if err != nil {
		panic(err)
	}
	g.Abi = *gasAbi
	g.ContractCalls = contractCalls
	return g
}

func makeAccountKey(h common.Address) []byte {
	return makeAddressKey(prefixAccount, h)
}

func (g *SAI) ContractCall_initialize(ic InteropContext) error {
	if ic.PersistingBlock() == nil || ic.PersistingBlock().Index != 0 {
		return ErrInitialize
	}
	validators, err := g.cs.Designate.GetValidators(ic.Dao(), 0)
	if err != nil {
		return err
	}
	wei := big.NewInt(1).Exp(big.NewInt(10), big.NewInt(SAIDecimal), nil)
	var addr common.Address
	if validators.Len() == 1 {
		addr = validators[0].Address()
	} else {
		script, err := validators.CreateDefaultMultiSigRedeemScript()
		if err != nil {
			return err
		}
		addr = hash.Hash160(script)
	}
	total := big.NewInt(1).Mul(big.NewInt(int64(g.initialSupply)), wei)
	err = g.addTokens(ic.Dao(), addr, total)
	if err == nil {
		log(ic, g.Address, total.Bytes(), g.Abi.Events["initialize"].ID)
	}
	return err
}

func (g *SAI) OnPersist(d *dao.Simple, block *block.Block) {

}

func (g *SAI) increaseBalance(gs *GasState, amount *big.Int) error {
	if amount.Sign() == -1 && gs.Balance.CmpAbs(amount) == -1 {
		return errors.New("insufficient funds")
	}
	gs.Balance.Add(gs.Balance, amount)
	return nil
}

func (g *SAI) getTotalSupply(d *dao.Simple) *big.Int {
	si := d.GetStorageItem(g.Address, totalSupplyKey)

	if si == nil {
		return nil
	}
	return big.NewInt(0).SetBytes(si)
}

func (g *SAI) saveTotalSupply(d *dao.Simple, supply *big.Int) {
	d.PutStorageItem(g.Address, totalSupplyKey, supply.Bytes())
}

func (g *SAI) getGasState(d *dao.Simple, key []byte) (*GasState, error) {
	si := d.GetStorageItem(g.Address, key)
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

func (g *SAI) putGasState(d *dao.Simple, key []byte, gs *GasState) error {
	data, err := io.ToByteArray(gs)
	if err != nil {
		return err
	}
	d.PutStorageItem(g.Address, key, data)
	return nil
}

func (g *SAI) addTokens(d *dao.Simple, h common.Address, amount *big.Int) error {
	if amount.Sign() == 0 {
		return nil
	}
	key := makeAccountKey(h)
	gs, err := g.getGasState(d, key)
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
		d.DeleteStorageItem(g.Address, key)
	} else {
		err = g.putGasState(d, key, ngs)
		if err != nil {
			return err
		}
	}
	supply := g.getTotalSupply(d)
	if supply == nil {
		supply = big.NewInt(0)
	}
	supply.Add(supply, amount)
	g.saveTotalSupply(d, supply)
	return nil
}

func (g *SAI) AddBalance(d *dao.Simple, h common.Address, amount *big.Int) {
	g.addTokens(d, h, amount)
}

func (g *SAI) SubBalance(d *dao.Simple, h common.Address, amount *big.Int) {
	neg := big.NewInt(0)
	neg.Neg(amount)
	g.addTokens(d, h, neg)
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
	if len(input) < 4 {
		return 0
	}
	method, err := g.Abi.MethodById(input[:4])
	if err != nil {
		return 0
	}
	switch method.Name {
	case "initialize":
		return 0
	default:
		return 0
	}
}

func (g *SAI) Run(ic InteropContext, input []byte) ([]byte, error) {
	return contractCall(g, &g.NativeContract, ic, input)
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
