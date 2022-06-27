package request

type (
	// Params represents the JSON-RPC params.
	Params []Param
)

// Value returns the param struct for the given
// index if it exists.
func (p Params) Value(index int) *Param {
	if len(p) > index {
		return &p[index]
	}

	return nil
}

func (p Params) String() string {
	str := "["
	for i, p := range p {
		if i > 0 {
			str += ", "
		}
		str += p.String()
	}
	str += "]"
	return str
}
