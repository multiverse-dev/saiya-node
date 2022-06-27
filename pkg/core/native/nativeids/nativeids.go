package nativeids

const (
	Policy      byte = 0xE0
	GAS         byte = 0xE1
	Management  byte = 0xE2
	Ledger      byte = 0xE3
	Designation byte = 0xE4
)

// IsValid checks that name is a valid native contract's name.
func IsValid(id byte) bool {
	return id >= Policy && id <= Designation
}
