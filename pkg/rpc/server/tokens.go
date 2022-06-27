package server

import (
	"github.com/multiverse-dev/saiya/pkg/rpc/response/result"
)

// tokenTransfers is a generic type used to represent NEP-11 and NEP-17 transfers.
type tokenTransfers struct {
	Sent     []interface{} `json:"sent"`
	Received []interface{} `json:"received"`
	Address  string        `json:"address"`
}

// ERC20TransferToERC721 adds an ID to provided NEP-17 transfer and returns a new
// NEP-11 structure.
func ERC20TransferToERC721(t17 *result.ERC20Transfer, id string) result.ERC721Transfer {
	return result.ERC721Transfer{
		Timestamp:   t17.Timestamp,
		Asset:       t17.Asset,
		Address:     t17.Address,
		ID:          id,
		Amount:      t17.Amount,
		Index:       t17.Index,
		NotifyIndex: t17.NotifyIndex,
		TxHash:      t17.TxHash,
	}
}
