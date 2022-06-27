package result

type (
	// Version model used for reporting server version
	// info.
	Version struct {
		// Magic contains network magic.
		// Deprecated: use Protocol.StateRootInHeader instead
		ChainID   uint64   `json:"network"`
		TCPPort   uint16   `json:"tcpport"`
		WSPort    uint16   `json:"wsport,omitempty"`
		Nonce     uint32   `json:"nonce"`
		UserAgent string   `json:"useragent"`
		Protocol  Protocol `json:"protocol"`
	}

	// Protocol represents network-dependent parameters.
	Protocol struct {
		AddressVersion              byte   `json:"addressversion"`
		ChainID                     uint64 `json:"network"`
		MillisecondsPerBlock        int    `json:"msperblock"`
		MaxTraceableBlocks          uint32 `json:"maxtraceableblocks"`
		MaxValidUntilBlockIncrement uint32 `json:"maxvaliduntilblockincrement"`
		MaxTransactionsPerBlock     uint16 `json:"maxtransactionsperblock"`
		MemoryPoolMaxTransactions   int    `json:"memorypoolmaxtransactions"`
		ValidatorsCount             byte   `json:"validatorscount"`
		InitialGasDistribution      uint64 `json:"initialgasdistribution"`
		// StateRootInHeader is true if state root is contained in block header.
	}
)
