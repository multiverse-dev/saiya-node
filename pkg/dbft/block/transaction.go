package block

import "github.com/ethereum/go-ethereum/common"

// Transaction is a generic transaction interface.
type Transaction interface {
	// Hash must return cryptographic hash of the transaction.
	// Transactions which have equal hashes are considered equal.
	Hash() common.Hash
}
