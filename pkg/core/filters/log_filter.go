package filters

import (
	"encoding/json"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
)

type LogFilter struct {
	FromBlock uint32
	ToBlock   uint32
	Blockhash common.Hash
	Address   common.Address
	Topics    []common.Hash
}

func (f *LogFilter) Match(l *types.Log) bool {
	if f.Blockhash != (common.Hash{}) && l.BlockHash != f.Blockhash {
		return false
	}
	if f.Blockhash == (common.Hash{}) && (l.BlockNumber < uint64(f.FromBlock) || l.BlockNumber >= uint64(f.ToBlock)) {
		return false
	}
	if l.Address != f.Address {
		return false
	}
	for _, topic := range f.Topics {
		for _, t := range l.Topics {
			if topic == t {
				return true
			}
		}
	}
	return false
}

type logFilterJSON struct {
	FromBlock string         `json:"fromBlock"`
	ToBlock   string         `json:"toBlock"`
	Blockhash common.Hash    `json:"blockHash"`
	Address   common.Address `json:"address"`
	Topics    []common.Hash  `json:"topics"`
}

func (f LogFilter) MarshalJSON() ([]byte, error) {
	lf := logFilterJSON{
		FromBlock: hexutil.EncodeUint64(uint64(f.FromBlock)),
		ToBlock:   hexutil.EncodeUint64(uint64(f.ToBlock)),
		Blockhash: f.Blockhash,
		Address:   f.Address,
		Topics:    f.Topics,
	}
	return json.Marshal(lf)
}

func (f *LogFilter) UnmarshalJSON(b []byte) error {
	lf := &logFilterJSON{}
	err := json.Unmarshal(b, lf)
	if err != nil {
		return err
	}
	fromBlock, err := hexutil.DecodeUint64(lf.FromBlock)
	if err != nil {
		return err
	}
	f.FromBlock = uint32(fromBlock)
	toBlock, err := hexutil.DecodeUint64(lf.ToBlock)
	if err != nil {
		return err
	}
	f.ToBlock = uint32(toBlock)
	f.Blockhash = lf.Blockhash
	f.Address = lf.Address
	f.Topics = lf.Topics
	return nil
}
