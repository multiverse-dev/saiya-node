package crypto

import "github.com/multiverse-dev/saiya/pkg/crypto/hash"

// VerifiableDecodable represents an object which can be verified and
// those hashable part can be encoded/decoded.
type VerifiableDecodable interface {
	hash.Hashable
	EncodeHashableFields() ([]byte, error)
	DecodeHashableFields([]byte) error
}
