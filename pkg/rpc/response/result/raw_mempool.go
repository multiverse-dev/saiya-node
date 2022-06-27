package result

import "github.com/ethereum/go-ethereum/common"

// RawMempool represents a result of getrawmempool RPC call.
type RawMempool struct {
	Height     uint32         `json:"height"`
	Verified   []common.Hash `json:"verified"`
	Unverified []common.Hash `json:"unverified"`
}
