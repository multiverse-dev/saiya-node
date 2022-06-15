package client

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"
	"testing"

	"github.com/nspcc-dev/neo-go/pkg/crypto/keys"
	"github.com/nspcc-dev/neo-go/pkg/io"
	"github.com/nspcc-dev/neo-go/pkg/rpc/request"
	"github.com/nspcc-dev/neo-go/pkg/smartcontract"
	"github.com/nspcc-dev/neo-go/pkg/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestClient_ExpendArrayIntoScriptCompat checks that result of expandArrayIntoScript is the same as
// for request.ExpandArrayIntoScript.
func TestClient_ExpendArrayIntoScriptCompat(t *testing.T) {
	priv, err := keys.NewPrivateKey()
	require.NoError(t, err)
	testCases := [][]smartcontract.Parameter{
		{{Type: smartcontract.BoolType, Value: true}},
		{{Type: smartcontract.IntegerType, Value: big.NewInt(123)}},
		{{Type: smartcontract.ByteArrayType, Value: []byte{1, 2, 3}}},
		{{Type: smartcontract.StringType, Value: "123"}},
		{{Type: smartcontract.Hash160Type, Value: util.Uint160{1, 2, 3}}},
		{{Type: smartcontract.Hash256Type, Value: util.Uint256{1, 2, 3}}},
		{{Type: smartcontract.PublicKeyType, Value: priv.PublicKey().Bytes()}},
		{{Type: smartcontract.SignatureType, Value: []byte{1, 2, 3}}},
		{{Type: smartcontract.ArrayType, Value: []smartcontract.Parameter{{Type: smartcontract.ByteArrayType, Value: []byte{1, 2, 3}}, {Type: smartcontract.BoolType, Value: false}}}},
	}

	buf := io.NewBufBinWriter()
	for i, params := range testCases {
		// Perform exactly the same action as RPC client/server do during `invokefunction` call handling
		// to be able to convert a set of smartcontract.Parameter to System.Contract.Call parameters.
		jBytes, err := json.Marshal(params)
		require.NoError(t, err)
		var slice request.Params
		err = json.Unmarshal(jBytes, &slice)
		require.NoError(t, err)
		require.NoError(t, request.ExpandArrayIntoScript(buf.BinWriter, slice))
		require.NoError(t, buf.Err)
		val := buf.Bytes()
		expected := make([]byte, len(val))
		copy(expected, val)
		buf.Reset()
		require.NoError(t, expandArrayIntoScript(buf.BinWriter, params))
		require.NoError(t, buf.Err)
		actual := buf.Bytes()
		assert.True(t, bytes.Equal(expected, actual), fmt.Sprintf("%d: expected %s, got %s", i, hex.EncodeToString(expected), hex.EncodeToString(actual)))
		buf.Reset()
	}
}
