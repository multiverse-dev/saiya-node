package wallet

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/multiverse-dev/saiya/pkg/encoding/address"
)

// Token represents imported token contract.
type Token struct {
	Name     string       `json:"name"`
	Hash     common.Address `json:"script_hash"`
	Decimals int64        `json:"decimals"`
	Symbol   string       `json:"symbol"`
	Standard string       `json:"standard"`
}

// NewToken returns new token contract info.
func NewToken(tokenHash common.Address, name, symbol string, decimals int64, standardName string) *Token {
	return &Token{
		Name:     name,
		Hash:     tokenHash,
		Decimals: decimals,
		Symbol:   symbol,
		Standard: standardName,
	}
}

// Address returns token address from hash.
func (t *Token) Address() string {
	return address.AddressToBase58(t.Hash)
}
