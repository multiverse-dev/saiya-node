package native

import "github.com/ethereum/go-ethereum/common"

func makeAddressKey(prefix byte, h common.Address) []byte {
	k := make([]byte, common.AddressLength+1)
	k[0] = prefix
	copy(k[1:], h.Bytes())
	return k
}

func makeHashKey(prefix byte, h common.Hash) []byte {
	k := make([]byte, common.HashLength+1)
	k[0] = prefix
	copy(k[1:], h.Bytes())
	return k
}

func MakeContractKey(h common.Address) []byte {
	return makeAddressKey(prefixContract, h)
}
