package netmode

import "strconv"

const (
	MainNet uint64 = 0x334f454e
	TestNet uint64 = 0x3454334e
	PrivNet uint64 = 56753
)

// String implements the stringer interface.
func String(chainId uint64) string {
	switch chainId {
	case PrivNet:
		return "privnet"
	case TestNet:
		return "testnet"
	case MainNet:
		return "mainnet"
	default:
		return "net 0x" + strconv.FormatUint(chainId, 16)
	}
}
