package config

import (
	"errors"
)

// ProtocolConfiguration represents the protocol config.
type (
	ProtocolConfiguration struct {
		// GarbageCollectionPeriod sets the number of blocks to wait before
		// starting the next MPT garbage collection cycle when RemoveUntraceableBlocks
		// option is used.
		GarbageCollectionPeriod uint32 `yaml:"GarbageCollectionPeriod"`

		ChainID     uint64 `yaml:"ChainID"`
		MemPoolSize int    `yaml:"MemPoolSize"`

		// InitialSaiSupply is the amount of Sai generated in the genesis block.
		InitialSaiSupply uint64 `yaml:"InitialSaiSupply"`
		// KeepOnlyLatestState specifies if MPT should only store latest state.
		// If true, DB size will be smaller, but older roots won't be accessible.
		// This value should remain the same for the same database.
		KeepOnlyLatestState bool `yaml:"KeepOnlyLatestState"`
		// RemoveUntraceableBlocks specifies if old data should be removed.
		RemoveUntraceableBlocks bool `yaml:"RemoveUntraceableBlocks"`
		// MaxBlockSize is the maximum block size in bytes.
		MaxBlockSize uint32 `yaml:"MaxBlockSize"`
		// MaxBlockSystemFee is the maximum overall system fee per block.
		MaxBlockGas uint64 `yaml:"MaxBlockGas"`
		// MaxTraceableBlocks is the length of the chain accessible to smart contracts.
		MaxTraceableBlocks uint32 `yaml:"MaxTraceableBlocks"`
		// MaxTransactionsPerBlock is the maximum amount of transactions per block.
		MaxTransactionsPerBlock uint16 `yaml:"MaxTransactionsPerBlock"`
		// SaveStorageBatch enables storage batch saving before every persist.
		SaveStorageBatch bool     `yaml:"SaveStorageBatch"`
		SecondsPerBlock  int      `yaml:"SecondsPerBlock"`
		SeedList         []string `yaml:"SeedList"`
		ValidatorsCount  int      `yaml:"ValidatorsCount"`
		StandbyCommittee []string `yaml:"StandbyCommittee"`
		// Whether to verify received blocks.
		VerifyBlocks bool `yaml:"VerifyBlocks"`
		// Whether to verify transactions in received blocks.
		VerifyTransactions bool `yaml:"VerifyTransactions"`
	}
)

// Validate checks ProtocolConfiguration for internal consistency and returns
// error if anything inappropriate found. Other methods can rely on protocol
// validity after this.
func (p *ProtocolConfiguration) Validate() error {
	if len(p.StandbyCommittee) == 0 {
		return errors.New("StandbyCommittee can't be empty")
	}
	if p.ValidatorsCount <= 0 {
		return errors.New("ValidatorsCount can't be 0")
	}
	if len(p.StandbyCommittee) < p.ValidatorsCount {
		return errors.New("validators count can't exceed the size of StandbyCommittee")
	}

	return nil
}

// GetNumOfCNs returns the number of validators for the given height.
// It implies valid configuration file.
func (p *ProtocolConfiguration) GetNumOfCNs(height uint32) int {
	return p.ValidatorsCount
}
