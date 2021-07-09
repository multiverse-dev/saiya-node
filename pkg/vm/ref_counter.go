package vm

import (
	"github.com/nspcc-dev/neo-go/pkg/vm/stackitem"
)

// refCounter represents reference counter for the VM.
type refCounter struct {
	size int
}

func newRefCounter() *refCounter {
	return &refCounter{}
}

// Add adds an item to the reference counter.
func (r *refCounter) Add(item stackitem.Item) {
	r.size++

	typ := item.Type()
	if typ == stackitem.ArrayT || typ == stackitem.MapT || typ == stackitem.StructT {
		if item.(stackitem.RC).AddRef() > 1 {
			return
		}

		switch t := item.(type) {
		case *stackitem.Array, *stackitem.Struct:
			for _, it := range item.Value().([]stackitem.Item) {
				r.Add(it)
			}
		case *stackitem.Map:
			for i := range t.Value().([]stackitem.MapElement) {
				r.Add(t.Value().([]stackitem.MapElement)[i].Value)
			}
		}
	}
}

// Remove removes item from the reference counter.
func (r *refCounter) Remove(item stackitem.Item) {
	r.size--

	typ := item.Type()
	if typ == stackitem.ArrayT || typ == stackitem.MapT || typ == stackitem.StructT {
		if item.(stackitem.RC).RemoveRef() > 0 {
			return
		}

		switch t := item.(type) {
		case *stackitem.Array, *stackitem.Struct:
			for _, it := range item.Value().([]stackitem.Item) {
				r.Remove(it)
			}
		case *stackitem.Map:
			for i := range t.Value().([]stackitem.MapElement) {
				r.Remove(t.Value().([]stackitem.MapElement)[i].Value)
			}
		}
	}
}
