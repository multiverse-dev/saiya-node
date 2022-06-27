package result

import "github.com/ethereum/go-ethereum/common"

// RelayResult ia a result of `sendrawtransaction` or `submitblock` RPC calls.
type RelayResult struct {
	Hash common.Hash `json:"hash"`
}
