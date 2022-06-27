package result

import (
	"github.com/ethereum/go-ethereum/common"
)

// ERC721Balances is a result for the getERC721balances RPC call.
type ERC721Balances struct {
	Balances []ERC721AssetBalance `json:"balance"`
	Address  string               `json:"address"`
}

// ERC721Balance is a structure holding balance of a NEP-11 asset.
type ERC721AssetBalance struct {
	Asset  common.Address         `json:"assethash"`
	Tokens []ERC721TokenBalance `json:"tokens"`
}

// ERC721TokenBalance represents balance of a single NFT.
type ERC721TokenBalance struct {
	ID          string `json:"tokenid"`
	Amount      string `json:"amount"`
	LastUpdated uint32 `json:"lastupdatedblock"`
}

// ERC20Balances is a result for the getERC20balances RPC call.
type ERC20Balances struct {
	Balances []ERC20Balance `json:"balance"`
	Address  string         `json:"address"`
}

// ERC20Balance represents balance for the single token contract.
type ERC20Balance struct {
	Asset       common.Address `json:"assethash"`
	Amount      string       `json:"amount"`
	LastUpdated uint32       `json:"lastupdatedblock"`
}

// ERC721Transfers is a result for the getERC721transfers RPC.
type ERC721Transfers struct {
	Sent     []ERC721Transfer `json:"sent"`
	Received []ERC721Transfer `json:"received"`
	Address  string           `json:"address"`
}

// ERC721Transfer represents single NEP-11 transfer event.
type ERC721Transfer struct {
	Timestamp   uint64       `json:"timestamp"`
	Asset       common.Address `json:"assethash"`
	Address     string       `json:"transferaddress,omitempty"`
	ID          string       `json:"tokenid"`
	Amount      string       `json:"amount"`
	Index       uint32       `json:"blockindex"`
	NotifyIndex uint32       `json:"transfernotifyindex"`
	TxHash      common.Hash `json:"txhash"`
}

// ERC20Transfers is a result for the getERC20transfers RPC.
type ERC20Transfers struct {
	Sent     []ERC20Transfer `json:"sent"`
	Received []ERC20Transfer `json:"received"`
	Address  string          `json:"address"`
}

// ERC20Transfer represents single ERC20 transfer event.
type ERC20Transfer struct {
	Timestamp   uint64       `json:"timestamp"`
	Asset       common.Address `json:"assethash"`
	Address     string       `json:"transferaddress,omitempty"`
	Amount      string       `json:"amount"`
	Index       uint32       `json:"blockindex"`
	NotifyIndex uint32       `json:"transfernotifyindex"`
	TxHash      common.Hash `json:"txhash"`
}

// KnownERC721Properties contains a list of well-known NEP-11 token property names.
var KnownERC721Properties = map[string]bool{
	"description": true,
	"image":       true,
	"name":        true,
	"tokenURI":    true,
}
