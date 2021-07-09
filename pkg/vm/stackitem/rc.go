package stackitem

type (
	rc struct {
		count int
	}

	RC interface {
		AddRef() int
		RemoveRef() int
	}
)

// AddRef adds reference to the counter.
func (r *rc) AddRef() int {
	r.count++
	return r.count
}

// RemoveRef removes reference from the counter.
func (r *rc) RemoveRef() int {
	r.count--
	return r.count
}
