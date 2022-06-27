package block

import (
	"sort"

	"github.com/ethereum/go-ethereum/common"
	"github.com/multiverse-dev/saiya/pkg/crypto/hash"
	"github.com/multiverse-dev/saiya/pkg/crypto/keys"
	"github.com/multiverse-dev/saiya/pkg/io"
)

type Witnesses struct {
	M          uint8
	PublicKeys keys.PublicKeys
	Signatures [][]byte
	address    *common.Address
}

func (s *Witnesses) DecodeBinary(br *io.BinReader) {
	s.M = br.ReadB()
	s.PublicKeys.DecodeBytes(br.ReadVarBytes())
	n := br.ReadVarUint()
	for i := uint64(0); i < n; i++ {
		s.Signatures[i] = br.ReadVarBytes()
	}
}

func (s *Witnesses) EncodeBinary(bw *io.BinWriter) {
	bw.WriteB(s.M)
	bw.WriteVarBytes(s.PublicKeys.Bytes())
	bw.WriteVarUint(uint64(len(s.Signatures)))
	for _, p := range s.Signatures {
		bw.WriteVarBytes(p)
	}
}

func (s *Witnesses) IsValid() bool {
	if s.M == 0 || s.M > uint8(s.PublicKeys.Len()) {
		return false
	}
	if s.PublicKeys.Len() != s.PublicKeys.Unique().Len() {
		return false
	}
	if len(s.Signatures) != int(s.M) {
		return false
	}
	return true
}

func (s *Witnesses) Address() common.Address {
	if s.address == nil {
		sort.Sort(s.PublicKeys)
		addr := hash.Hash160(append([]byte{s.M}, s.PublicKeys.Bytes()...))
		s.address = &addr
	}
	return *s.address
}

func (s *Witnesses) VerifyHashable(chainId uint64, hh hash.Hashable) bool {
	if !s.IsValid() {
		return false
	}
	var i uint8 = 0
	for _, sig := range s.Signatures {
		for _, pk := range s.PublicKeys {
			if pk.VerifyHashable(sig, chainId, hh) {
				i++
				break
			}
		}
	}
	return i == s.M
}
