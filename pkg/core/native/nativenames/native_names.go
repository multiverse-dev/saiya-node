package nativenames

const (
	Management  = "ContractManagement"
	Ledger      = "LedgerContract"
	Sai         = "SaiToken"
	Policy      = "PolicyContract"
	Designation = "RoleManagement"
)

// IsValid checks that name is a valid native contract's name.
func IsValid(name string) bool {
	return name == Management ||
		name == Ledger ||
		name == Sai ||
		name == Policy ||
		name == Designation
}
