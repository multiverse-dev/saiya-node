package result

import (
	"encoding/json"
	"errors"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
)

// UnclaimedGas response wrapper.
type UnclaimedGas struct {
	Address   common.Address
	Unclaimed big.Int
}

// unclaimedGas is an auxiliary struct for JSON marhsalling.
type unclaimedGas struct {
	Address   string `json:"address"`
	Unclaimed string `json:"unclaimed"`
}

// MarshalJSON implements json.Marshaler interface.
func (g UnclaimedGas) MarshalJSON() ([]byte, error) {
	gas := &unclaimedGas{
		Address:   g.Address.String(),
		Unclaimed: g.Unclaimed.String(),
	}
	return json.Marshal(gas)
}

// UnmarshalJSON implements json.Unmarshaler interface.
func (g *UnclaimedGas) UnmarshalJSON(data []byte) error {
	gas := new(unclaimedGas)
	if err := json.Unmarshal(data, gas); err != nil {
		return err
	}
	uncl, ok := new(big.Int).SetString(gas.Unclaimed, 10)
	if !ok {
		return errors.New("failed to convert unclaimed gas")
	}
	g.Unclaimed = *uncl
	addr := common.HexToAddress(gas.Address)
	g.Address = addr
	return nil
}
