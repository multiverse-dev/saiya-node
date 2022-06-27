package filters

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLogFilterJson(t *testing.T) {
	j := `{
		"topics": ["0x000000000000000000000000a94f5374fce5edbc8e2a8697c15331677e6ebf0b"]
	  }`
	lf := &LogFilter{}
	err := json.Unmarshal([]byte(j), lf)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(lf.Topics))
}
