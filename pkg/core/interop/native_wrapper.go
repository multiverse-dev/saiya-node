package interop

type nativeWrapper struct {
	nativeContract NativeContract
	ic             *Context
}

func (w nativeWrapper) RequiredGas(input []byte) uint64 {
	return w.nativeContract.RequiredGas(w.ic, input)
}

func (w nativeWrapper) Run(input []byte) ([]byte, error) {
	return w.nativeContract.Run(w.ic, input)
}
