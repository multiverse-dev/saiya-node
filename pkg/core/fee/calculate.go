package fee

import (
	"github.com/multiverse-dev/saiya/pkg/io"
)

// ECDSAVerifyPrice is a gas price of a single verification.
const ECDSAVerifyPrice = 1 << 15

// Calculate returns network fee for transaction.
func Calculate(base int64, script []byte) (int64, int) {
	var (
		netFee int64
		size   int
	)
	size += 67 + io.GetVarSize(script)

	return netFee, size
}
