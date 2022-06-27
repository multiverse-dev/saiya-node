package payload

//go:generate stringer -type=ChangeViewReason -linecomment

// ChangeViewReason represents a view change reason code.
type ChangeViewReason byte

const (
	CVTimeout               ChangeViewReason = 0x0  // Timeout
	CVChangeAgreement       ChangeViewReason = 0x1  // ChangeAgreement
	CVTxNotFound            ChangeViewReason = 0x2  // TxNotFound
	CVTxRejectedByPolicy    ChangeViewReason = 0x3  // TxRejectedByPolicy
	CVTxInvalid             ChangeViewReason = 0x4  // TxInvalid
	CVBlockRejectedByPolicy ChangeViewReason = 0x5  // BlockRejectedByPolicy
	CVUnknown               ChangeViewReason = 0xff // Unknown
)
