package noderoles

// Role represents type of participant.
type Role byte

// Role enumeration.
const (
	Committee      Role = 0
	StateValidator Role = 1
)

func IsValid(r Role) bool {
	return r == Committee || r == StateValidator
}
