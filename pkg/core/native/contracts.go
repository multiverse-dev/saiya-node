package native

import (
	"errors"
	"strings"

	"github.com/multiverse-dev/saiya/pkg/config"
	"github.com/multiverse-dev/saiya/pkg/core/state"
)

const (
	defaultNativeReadFee  = 1000
	defaultNativeWriteFee = 10000
)

var (
	ErrEmptyInput      = errors.New("empty input")
	ErrInvalidInput    = errors.New("invalid input")
	ErrInvalidMethodID = errors.New("invalid method id")
	ErrEmptyNodeList   = errors.New("node list is empty")
	ErrLargeNodeList   = errors.New("node list is too large")
	ErrInvalidRole     = errors.New("invalid role")
	ErrInvalidSender   = errors.New("sender check failed")
	ErrNoBlock         = errors.New("no persisting block in the context")
	ErrInitialize      = errors.New("initialize should only execute in genesis block")
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
	cs.SAI = NewSAI(cfg.InitialSAISupply)
	cs.Contracts = append(cs.Contracts, cs.SAI.NativeContract)
	cs.Ledger = NewLedger()
	cs.Contracts = append(cs.Contracts, cs.Ledger.NativeContract)
	cs.Management = NewManagement()
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

func checkCommittee(ic InteropContext) error {
	if ic.PersistingBlock() == nil {
		return ErrNoBlock
	}
	committeeAddress, err := ic.Natives().Designate.GetCommitteeAddress(ic.Dao(), ic.PersistingBlock().Index)
	if err != nil {
		return err
	}
	if ic.Sender() != committeeAddress {
		return ErrInvalidSender
	}
	return nil
}
