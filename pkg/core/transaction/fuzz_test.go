package transaction

import (
	"math/rand"
	"testing"

	"github.com/stretchr/testify/require"
)

func FuzzReader(f *testing.F) {
	for i := 0; i < 100; i++ {
		seed := make([]byte, rand.Uint32()%1000)
		rand.Read(seed)
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, value []byte) {
		require.NotPanics(t, func() {
			_, _ = NewTransactionFromBytes(value)
		})
	})
}
