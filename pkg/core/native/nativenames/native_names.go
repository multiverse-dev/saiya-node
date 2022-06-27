package nativenames

const (
	Management  = "ContractManagement"
	Ledger      = "LedgerContract"
	SAI         = "SAIYAToken"
	Policy      = "PolicyContract"
	Designation = "RoleManagement"
)

// IsValid checks that name is a valid native contract's name.
func IsValid(name string) bool {
	return name == Management ||
		name == Ledger ||
		name == SAI ||
		name == Policy ||
		name == Designation
}
