package wallet

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestUnmarshalJSON(t *testing.T) {
	j := `{"chainId":"0x2c6","tx":{"hash":"0x476dba117809db284348117e4aae749a900c8747698953ce26f4c32d8fbc9819","size":"0xb0","nonce":"0x1","gasPrice":"0x2540be400","gas":"0x4b44c","from":"0x3876b7cb56310266748eca29db9b95a52c4241b8","to":"0x657bb17b66a3a515fec705ae1e4a8d4af251009f","value":"0xde0b6b3a7640000","data":"0x","witness":{"verification":"0x0203022c304cda3491abaf91c50fc3735da5146319fca912bb839724a9baecc41139ad036b1897607f394a4c26a36227517307f8cd3e3c7784a4802c3521d18735dbd0f7027623ecda6016ae033556d4169e75ca19a2dd678d15d78237f5d67f207a07a5b3","invocation":"0x"}},"m":"0x2","parameters":{"036b1897607f394a4c26a36227517307f8cd3e3c7784a4802c3521d18735dbd0f7":"0xe8567109a4abdb43d915640366d916003d51f801f55bff6a87db44aecf7f9fab05bdae2febfdfc2f4a56f93339d42c95b5370810314fc82d35bee2c546c026e6"}}`
	sc := new(SignContext)
	err := json.Unmarshal([]byte(j), sc)
	assert.NoError(t, err)
}
