package noderoles

// Role represents type of participant.
type Role byte

// Role enumeration.
const (
	Validator      Role = 0
	Committee      Role = 1
	StateValidator Role = 2
)

func IsValid(r Role) bool {
	return r == Committee || r == Validator || r == StateValidator
}
