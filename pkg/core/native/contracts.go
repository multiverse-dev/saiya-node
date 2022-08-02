package native

import (
	"errors"
	"fmt"
	"reflect"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/multiverse-dev/saiya/pkg/config"
	"github.com/multiverse-dev/saiya/pkg/core/dao"
	"github.com/multiverse-dev/saiya/pkg/core/state"
	"github.com/multiverse-dev/saiya/pkg/dbft/block"
)

const (
	defaultNativeReadFee     = 200
	defaultNativeWriteFee    = 5000
	contractMethodNamePrefix = "ContractCall_"
)

var (
	ErrEmptyInput                = errors.New("empty input")
	ErrInvalidInput              = errors.New("invalid input")
	ErrInvalidMethodID           = errors.New("invalid method id")
	ErrEmptyNodeList             = errors.New("node list is empty")
	ErrLargeNodeList             = errors.New("node list is too large")
	ErrInvalidRole               = errors.New("invalid role")
	ErrInvalidSender             = errors.New("sender check failed")
	ErrNoBlock                   = errors.New("no persisting block in the context")
	ErrInitialize                = errors.New("initialize should only execute in genesis block")
	ErrInvalidContractCallInputs = errors.New("need at least 1 for InteropContext in contract call")
	ErrInvalidContractCallReturn = errors.New("invalid return value in contract call")
)

type Contracts struct {
	SAI        *SAI
	Ledger     *Ledger
	Designate  *Designate
	Management *Management
	Policy     *Policy
	Contracts  []state.NativeContract
}

func NewContracts(cfg config.ProtocolConfiguration) *Contracts {
	cs := &Contracts{
		Contracts: make([]state.NativeContract, 0, 4),
	}
	cs.SAI = NewSAI(cs, cfg.InitialSAISupply)
	cs.Contracts = append(cs.Contracts, cs.SAI.NativeContract)
	cs.Ledger = NewLedger()
	cs.Contracts = append(cs.Contracts, cs.Ledger.NativeContract)
	cs.Management = NewManagement(cs)
	cs.Contracts = append(cs.Contracts, cs.Management.NativeContract)
	cs.Designate = NewDesignate(cfg)
	cs.Contracts = append(cs.Contracts, cs.Designate.NativeContract)
	cs.Policy = NewPolicy(cs)
	cs.Contracts = append(cs.Contracts, cs.Policy.NativeContract)
	return cs
}

func (cs *Contracts) ByName(name string) *state.NativeContract {
	name = strings.ToLower(name)
	for _, ctr := range cs.Contracts {
		if strings.ToLower(ctr.Name) == name {
			return &ctr
		}
	}
	return nil
}

func (cs *Contracts) OnPersist(d *dao.Simple, block *block.Block) {
	cs.SAI.OnPersist(d, block)
}

func convertType(in reflect.Type) (abi.Type, error) {
	switch in.Kind() {
	case reflect.Array, reflect.Slice:
		switch in.Elem().Kind() {
		case reflect.Uint8:
			if in.Size() == common.AddressLength && in.Name() == "Address" {
				return abi.NewType("address", "address", nil)
			}
			if in.Size() == common.HashLength && in.Name() == "Hash" {
				return abi.NewType("hash", "hash", nil)
			}
			return abi.NewType("bytes", "bytes", nil)
		default:
			return abi.Type{}, fmt.Errorf("invalid array element type: %s", in.Elem().Name())
		}
	case reflect.Uint8:
		return abi.NewType("uint8", "uint8", nil)
	case reflect.Uint64:
		return abi.NewType("uint64", "uint64", nil)
	default:
		return abi.Type{}, fmt.Errorf("invalid type: %s", in.Name())
	}
}

func constructAbi(any interface{}) (*abi.ABI, map[string]reflect.Value, error) {
	const paramNames = "abcdefghijklmnopqrstuvwxyz"
	a := new(abi.ABI)
	a.Methods = make(map[string]abi.Method)
	a.Events = make(map[string]abi.Event)
	a.Errors = make(map[string]abi.Error)
	contractCalls := make(map[string]reflect.Value)
	ty := reflect.TypeOf(any)
	for i := 0; i < ty.NumMethod(); i++ {
		method := ty.Method(i)
		if strings.HasPrefix(method.Name, contractMethodNamePrefix) {
			name := method.Name[len(contractMethodNamePrefix):]
			if len(name) == 0 {
				continue
			}
			numIn := method.Type.NumIn() - 2
			if numIn < 0 {
				return nil, contractCalls, ErrInvalidContractCallInputs
			}
			inputs := make([]abi.Argument, numIn)
			for i := 0; i < numIn; i++ {
				in := method.Type.In(i + 2)
				ty, err := convertType(in)
				if err != nil {
					return nil, contractCalls, fmt.Errorf("can't convert %d input of %s: %w", i, name, err)
				}
				inputs[i] = abi.Argument{
					Name: string(paramNames[i]),
					Type: ty,
				}
			}
			numOut := method.Type.NumOut()
			if numOut < 1 || numOut > 2 {
				return nil, contractCalls, ErrInvalidContractCallReturn
			}
			outE := method.Type.Out(numOut - 1)
			if !outE.Implements(reflect.TypeOf(new(error)).Elem()) {
				return nil, contractCalls, ErrInvalidContractCallReturn
			}
			outputs := abi.Arguments{}
			if numOut > 1 {
				b := method.Type.Out(0)
				if (b.Kind() != reflect.Array && b.Kind() != reflect.Slice) || b.Elem().Kind() != reflect.Int8 {
					return nil, contractCalls, ErrInvalidContractCallReturn
				}
				r, _ := abi.NewType("bytes", "bytes", nil)
				outputs = abi.Arguments{
					{
						Name: "result",
						Type: r,
					},
				}
			}
			contractCalls[name] = method.Func
			if len(inputs) > 0 {
				a.Events[name] = abi.NewEvent(name, name, false, inputs)
			}
			a.Methods[name] = abi.NewMethod(name, name, abi.Function, "nonpayable", false, false, inputs, outputs)
		}
	}
	return a, contractCalls, nil
}

func contractCall(contract interface{}, nativeContract *state.NativeContract, ic InteropContext, input []byte) ([]byte, error) {
	if len(input) < 4 {
		return nil, errors.New("invalid input")
	}
	method, err := nativeContract.Abi.MethodById(input[:4])
	if err != nil {
		return nil, err
	}
	if method == nil {
		return nil, errors.New("method not found")
	}
	args, err := method.Inputs.Unpack(input[4:])
	if err != nil {
		return nil, err
	}
	vals := make([]reflect.Value, len(args))
	for i, arg := range args {
		vals[i] = reflect.ValueOf(arg)
	}
	rs := nativeContract.ContractCalls[method.Name].Call(append([]reflect.Value{reflect.ValueOf(contract), reflect.ValueOf(ic)}, vals...))
	if len(rs) < 1 || len(rs) > 2 {
		return nil, ErrInvalidContractCallReturn
	}
	if !rs[len(rs)-1].IsNil() {
		e, ok := rs[len(rs)-1].Interface().(error)
		if !ok {
			return nil, ErrInvalidContractCallReturn
		}
		return nil, e
	}
	if len(rs) == 1 {
		return nil, nil
	}
	r, ok := rs[0].Interface().([]byte)
	if !ok {
		return nil, ErrInvalidContractCallReturn
	}
	return r, nil
}

func log(ic InteropContext, address common.Address, data []byte, topics ...common.Hash) {
	ic.Log(&types.Log{
		Address:     address,
		Topics:      topics,
		Data:        data,
		BlockNumber: uint64(ic.PersistingBlock().Index),
	})
}
