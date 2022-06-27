package paramcontext

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/multiverse-dev/saiya/pkg/core/transaction"
	"github.com/multiverse-dev/saiya/pkg/wallet"
)

// validUntilBlockIncrement is the number of extra blocks to add to an exported transaction.
const validUntilBlockIncrement = 50

// InitAndSave creates incompletely signed transaction which can used
// as input to `multisig sign`.
func InitAndSave(chainId uint64, tx *transaction.Transaction, acc *wallet.Account, filename string) error {
	// avoid fast transaction expiration
	return Save(tx, filename)
}

// Read reads parameter context from file.
func Read(filename string) (*transaction.Transaction, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("can't read input file: %w", err)
	}

	tx := new(transaction.Transaction)
	if err := json.Unmarshal(data, tx); err != nil {
		return nil, fmt.Errorf("can't parse transaction: %w", err)
	}
	return tx, nil
}

// Save writes parameter context to file.
func Save(tx *transaction.Transaction, filename string) error {
	if data, err := json.Marshal(tx); err != nil {
		return fmt.Errorf("can't marshal transaction: %w", err)
	} else if err := os.WriteFile(filename, data, 0644); err != nil {
		return fmt.Errorf("can't write transaction to file: %w", err)
	}
	return nil
}
