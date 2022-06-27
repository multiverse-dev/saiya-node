package request

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/multiverse-dev/saiya/pkg/io"
)

// ExpandArrayIntoScript pushes all FuncParam parameters from the given array
// into the given buffer in reverse order.
func ExpandArrayIntoScript(script *io.BinWriter, slice []Param) error {

	return script.Err
}

// CreateFunctionInvocationScript creates a script to invoke given contract with
// given parameters.
func CreateFunctionInvocationScript(contract common.Address, method string, param *Param) ([]byte, error) {
	script := io.NewBufBinWriter()
	return script.Bytes(), nil
}
