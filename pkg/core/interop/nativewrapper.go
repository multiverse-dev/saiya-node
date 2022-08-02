package interop

import "github.com/ethereum/go-ethereum/common"

type nativeWrapper struct {
	nativeContract NativeContract
	ic             *Context
}

func (w nativeWrapper) RequiredGas(input []byte) uint64 {
	return w.nativeContract.RequiredGas(w.ic, input)
}

func (w nativeWrapper) Run(caller common.Address, input []byte) ([]byte, error) {
	w.ic.caller = caller
	return w.nativeContract.Run(w.ic, input)
}
