package payload

import (
	"errors"

	"github.com/nspcc-dev/neo-go/pkg/config/netmode"
	"github.com/nspcc-dev/neo-go/pkg/core/transaction"
	"github.com/nspcc-dev/neo-go/pkg/io"
)

type Transactions struct {
	Network netmode.Magic
	Values  []*transaction.Transaction
}

// DecodeBinary implements Serializable interface.
func (p *Transactions) DecodeBinary(br *io.BinReader) {
	u := br.ReadVarUint()
	if u > MaxHashesCount {
		br.Err = errors.New("too big array")
		return
	}
	p.Values = make([]*transaction.Transaction, u)
	for i := range p.Values {
		p.Values[i] = &transaction.Transaction{Network: p.Network}
		p.Values[i].DecodeBinary(br)
	}
}

// EncodeBinary implements Serializable interface.
func (p *Transactions) EncodeBinary(bw *io.BinWriter) {
	bw.WriteArray(p.Values)
}
