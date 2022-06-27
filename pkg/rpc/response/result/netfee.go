package result

// NetworkFee represents a result of calculatenetworkfee RPC call.
type NetworkFee struct {
	Value uint64 `json:"networkfee,string"`
}
