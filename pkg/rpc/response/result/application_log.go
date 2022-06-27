package result

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

// ApplicationLog represent the results of the script executions for block or transaction.
type ApplicationLog struct {
	Container common.Hash
	Logs      []types.Log
}

// NewApplicationLog creates ApplicationLog from a set of several application execution results
// including only the results with the specified trigger.
func NewApplicationLog(hash common.Hash, aer *types.Receipt) ApplicationLog {
	result := ApplicationLog{
		Container: hash,
		Logs:      make([]types.Log, len(aer.Logs)),
	}
	for _, log := range aer.Logs {
		result.Logs = append(result.Logs, *log)
	}
	return result
}
