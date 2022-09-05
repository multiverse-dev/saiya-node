package zk

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestVerif(t *testing.T) {
	proof := []byte{1}
	key := []byte{1}
	r := Verify(proof, key)
	assert.Equal(t, 0, r)
}
