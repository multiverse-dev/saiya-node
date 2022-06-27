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

		// InitialSAIYASupply is the amount of SAIYA generated in the genesis block.
		InitialSAISupply uint64 `yaml:"InitialSAIYASupply"`
		// KeepOnlyLatestState specifies if MPT should only store latest state.
		// If true, DB size will be smaller, but older roots won't be accessible.
		// This value should remain the same for the same database.
		KeepOnlyLatestState bool `yaml:"KeepOnlyLatestState"`
		// RemoveUntraceableBlocks specifies if old data should be removed.
		RemoveUntraceableBlocks bool `yaml:"RemoveUntraceableBlocks"`
		// MaxBlockSize is the maximum block size in bytes.
		MaxBlockSize uint32 `yaml:"MaxBlockSize"`
		// MaxBlockSystemFee is the maximum overall system fee per block.
		MaxBlockSystemFee uint64 `yaml:"MaxBlockSystemFee"`
		// MaxTraceableBlocks is the length of the chain accessible to smart contracts.
		MaxTraceableBlocks uint32 `yaml:"MaxTraceableBlocks"`
		// MaxTransactionsPerBlock is the maximum amount of transactions per block.
		MaxTransactionsPerBlock uint16 `yaml:"MaxTransactionsPerBlock"`
		// MaxValidUntilBlockIncrement is the upper increment size of blockchain height in blocks
		// exceeding that a transaction should fail validation. It is set to estimated daily number
		// of blocks with 15s interval.
		MaxValidUntilBlockIncrement uint32 `yaml:"MaxValidUntilBlockIncrement"`
		// SaveStorageBatch enables storage batch saving before every persist.
		SaveStorageBatch  bool     `yaml:"SaveStorageBatch"`
		SecondsPerBlock   int      `yaml:"SecondsPerBlock"`
		SeedList          []string `yaml:"SeedList"`
		StandbyCommittee  []string `yaml:"StandbyCommittee"`
		StandbyValidators []string `yaml:"StandbyValidators"`

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
	if len(p.StandbyValidators) == 0 {
		return errors.New("StandyValidators can't be empty")
	}
	return nil
}

// GetNumOfCNs returns the number of validators for the given height.
// It implies valid configuration file.
func (p *ProtocolConfiguration) GetNumOfCNs(height uint32) int {
	return len(p.StandbyValidators)
}
