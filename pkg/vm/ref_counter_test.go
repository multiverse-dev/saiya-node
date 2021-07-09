package vm

import (
	"math/big"
	"testing"

	"github.com/nspcc-dev/neo-go/pkg/vm/stackitem"
	"github.com/stretchr/testify/require"
)

func TestRefCounter_Add(t *testing.T) {
	r := newRefCounter()

	require.Equal(t, 0, r.size)

	r.Add(stackitem.Null{})
	require.Equal(t, 1, r.size)

	r.Add(stackitem.Null{})
	require.Equal(t, 2, r.size) // count scalar items twice

	arr := stackitem.NewArray([]stackitem.Item{stackitem.NewByteArray([]byte{1}), stackitem.NewBool(false)})
	r.Add(arr)
	require.Equal(t, 5, r.size) // array + 2 elements

	r.Add(arr)
	require.Equal(t, 6, r.size) // count only array

	r.Remove(arr)
	require.Equal(t, 5, r.size)

	r.Remove(arr)
	require.Equal(t, 2, r.size)
}

func BenchmarkRefCounter(b *testing.B) {
	b.Run("big array", func(b *testing.B) {
		b.Run("new", func(b *testing.B) {
			arr := stackitem.NewArray([]stackitem.Item{})
			for i := 0; i < 2040; i++ {
				arr = stackitem.NewArray([]stackitem.Item{arr})
			}
			v := New()
			benchmarkRefCounter(b, v, arr)
		})
		b.Run("repeat", func(b *testing.B) {
			arr := stackitem.NewArray([]stackitem.Item{})
			for i := 0; i < 2040; i++ {
				arr = stackitem.NewArray([]stackitem.Item{arr})
			}
			v := New()
			v.estack.PushVal(arr)
			benchmarkRefCounter(b, v, arr)
		})
	})
	b.Run("primitive", func(b *testing.B) {
		benchmarkRefCounter(b, New(), stackitem.NewBigInteger(big.NewInt(12345678)))
		b.Run("big stack", func(b *testing.B) {
			v := New()
			for i := 0; i < 100; i++ {
				v.estack.PushVal(stackitem.NewBigInteger(big.NewInt(13)))
			}
			benchmarkRefCounter(b, v, stackitem.NewBigInteger(big.NewInt(12345678)))
		})
	})
	b.Run("put map on a big stack", func(b *testing.B) {
		v := New()
		for i := 0; i < 100; i++ {
			v.estack.PushVal([]stackitem.Item{stackitem.NewBool(true)})
		}

		m := stackitem.NewMap()
		m.Add(stackitem.Make(1), stackitem.Make(2))
		m.Add(stackitem.Make(2), stackitem.Make([]stackitem.Item{}))
		m.Add(stackitem.Make(3), stackitem.Make([]stackitem.Item{stackitem.NewBool(false)}))
	})
}

func benchmarkRefCounter(b *testing.B, v *VM, item stackitem.Item) {
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		v.estack.PushVal(item)
		v.estack.Pop()
	}
}
