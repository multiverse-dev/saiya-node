package dbft

import (
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/multiverse-dev/saiya/pkg/dbft/block"
	"github.com/multiverse-dev/saiya/pkg/dbft/payload"
	"github.com/multiverse-dev/saiya/pkg/dbft/timer"
	"go.uber.org/zap"
)

type (
	// DBFT is a wrapper over Context containing service configuration and
	// some other parameters not directly related to dBFT's state machine.
	DBFT struct {
		Context
		Config

		*sync.Mutex
		cache      cache
		recovering bool
	}

	// Service is an interface for dBFT consensus.
	Service interface {
		Start()
		OnTransaction(block.Transaction)
		OnReceive(payload.ConsensusPayload)
		OnTimeout(timer.HV)
	}
)

var _ Service = (*DBFT)(nil)

// New returns new DBFT instance with provided options
// and nil if some of the options are missing or invalid.
func New(options ...Option) *DBFT {
	cfg := defaultConfig()

	for _, option := range options {
		option(cfg)
	}

	if err := checkConfig(cfg); err != nil {
		return nil
	}

	d := &DBFT{
		Mutex:  new(sync.Mutex),
		Config: *cfg,
		Context: Context{
			Config: cfg,
		},
	}

	return d
}

func (d *DBFT) addTransaction(tx block.Transaction) {
	d.Transactions[tx.Hash()] = tx
	if d.hasAllTransactions() {
		if d.IsPrimary() || d.Context.WatchOnly() {
			return
		}

		if !d.createAndCheckBlock() {
			return
		}

		d.extendTimer(2)
		d.sendPrepareResponse()
		d.checkPrepare()
	}
}

// Start initializes dBFT instance and starts protocol if node is primary.
func (d *DBFT) Start() {
	d.cache = newCache()
	d.InitializeConsensus(0)
	d.start()
}

// InitializeConsensus initializes dBFT instance.
func (d *DBFT) InitializeConsensus(view byte) {
	d.reset(view)

	var role string

	switch {
	case d.IsPrimary():
		role = "Primary"
	case d.Context.WatchOnly():
		role = "WatchOnly"
	default:
		role = "Backup"
	}

	var logMsg = "initializing dbft"
	if view > 0 {
		logMsg = "changing dbft view"
	}

	d.Logger.Info(logMsg,
		zap.Uint32("height", d.BlockIndex),
		zap.Uint("view", uint(view)),
		zap.Int("index", d.MyIndex),
		zap.String("role", role))

	if d.Context.WatchOnly() {
		return
	}

	var timeout time.Duration
	if d.IsPrimary() && !d.recovering {
		// Initializing to view 0 means we have just persisted previous block or are starting consensus first time.
		// In both cases we should wait full timeout value.
		// Having non-zero view means we have to start immediately.
		if view == 0 {
			timeout = d.SecondsPerBlock
		}
	} else {
		timeout = d.SecondsPerBlock << (d.ViewNumber + 1)
	}
	if d.lastBlockIndex+1 == d.BlockIndex {
		var ts = d.Timer.Now()
		var diff = ts.Sub(d.lastBlockTime)
		timeout -= diff
		if timeout < 0 {
			timeout = 0
		}
	}
	d.changeTimer(timeout)
}

// OnTransaction notifies service about receiving new transaction.
func (d *DBFT) OnTransaction(tx block.Transaction) {
	// d.Logger.Debug("OnTransaction",
	// 	zap.Bool("backup", d.IsBackup()),
	// 	zap.Bool("not_accepting", d.NotAcceptingPayloadsDueToViewChanging()),
	// 	zap.Bool("request_ok", d.RequestSentOrReceived()),
	// 	zap.Bool("response_sent", d.ResponseSent()),
	// 	zap.Bool("block_sent", d.BlockSent()))
	if !d.IsBackup() || d.NotAcceptingPayloadsDueToViewChanging() ||
		!d.RequestSentOrReceived() || d.ResponseSent() || d.BlockSent() ||
		len(d.MissingTransactions) == 0 {
		return
	}

	for i := range d.MissingTransactions {
		if tx.Hash() == d.MissingTransactions[i] {
			d.addTransaction(tx)
			// `addTransaction` checks for responses and commits. If this was the last transaction
			// Context could be initialized on a new height, clearing this field.
			if len(d.MissingTransactions) == 0 {
				return
			}
			theLastOne := len(d.MissingTransactions) - 1
			if i < theLastOne {
				d.MissingTransactions[i] = d.MissingTransactions[theLastOne]
			}
			d.MissingTransactions = d.MissingTransactions[:theLastOne]
			return
		}
	}
}

// OnTimeout advances state machine as if timeout was fired.
func (d *DBFT) OnTimeout(hv timer.HV) {
	if d.Context.WatchOnly() {
		return
	}

	if hv.Height != d.BlockIndex || hv.View != d.ViewNumber {
		d.Logger.Debug("timeout: ignore old timer",
			zap.Uint32("height", hv.Height),
			zap.Uint("view", uint(hv.View)))

		return
	}

	d.Logger.Debug("timeout",
		zap.Uint32("height", hv.Height),
		zap.Uint("view", uint(hv.View)))

	if d.IsPrimary() && !d.RequestSentOrReceived() {
		d.sendPrepareRequest()
	} else if (d.IsPrimary() && d.RequestSentOrReceived()) || d.IsBackup() {
		if d.CommitSent() {
			d.Logger.Debug("send recovery to resend commit")
			d.sendRecoveryMessage()
			d.changeTimer(d.SecondsPerBlock << 1)
		} else {
			d.sendChangeView(payload.CVTimeout)
		}
	}
}

// OnReceive advances state machine in accordance with msg.
func (d *DBFT) OnReceive(msg payload.ConsensusPayload) {
	if int(msg.ValidatorIndex()) >= len(d.Validators) {
		d.Logger.Error("too big validator index", zap.Uint16("from", msg.ValidatorIndex()))
		return
	}

	if msg.Payload() == nil {
		d.Logger.DPanic("invalid message")
		return
	}

	d.Logger.Debug("received message",
		zap.Stringer("type", msg.Type()),
		zap.Uint16("from", msg.ValidatorIndex()),
		zap.Uint32("height", msg.Height()),
		zap.Uint("view", uint(msg.ViewNumber())),
		zap.Uint32("my_height", d.BlockIndex),
		zap.Uint("my_view", uint(d.ViewNumber)))

	if msg.Height() < d.BlockIndex {
		d.Logger.Debug("ignoring old height", zap.Uint32("height", msg.Height()))
		return
	} else if msg.Height() > d.BlockIndex || (msg.Height() == d.BlockIndex &&
		msg.ViewNumber() == d.ViewNumber+1 && msg.Type() != payload.RecoveryMessageType) {
		d.Logger.Debug("caching message from future",
			zap.Uint32("height", msg.Height()),
			zap.Uint("view", uint(msg.ViewNumber())),
			zap.Any("cache", d.cache.mail[msg.Height()]))
		d.cache.addMessage(msg)
		return
	} else if msg.ValidatorIndex() > uint16(d.N()) {
		return
	}

	hv := d.LastSeenMessage[msg.ValidatorIndex()]
	if hv == nil || hv.Height < msg.Height() || hv.View < msg.ViewNumber() {
		d.LastSeenMessage[msg.ValidatorIndex()] = &timer.HV{
			Height: msg.Height(),
			View:   msg.ViewNumber(),
		}
	}

	switch msg.Type() {
	case payload.ChangeViewType:
		d.onChangeView(msg)
	case payload.PrepareRequestType:
		d.onPrepareRequest(msg)
	case payload.PrepareResponseType:
		d.onPrepareResponse(msg)
	case payload.CommitType:
		d.onCommit(msg)
	case payload.RecoveryRequestType:
		d.onRecoveryRequest(msg)
	case payload.RecoveryMessageType:
		d.onRecoveryMessage(msg)
	default:
		d.Logger.DPanic("wrong message type")
	}
}

// start performs initial operations and returns messages to be sent.
// It must be called after every height or view increment.
func (d *DBFT) start() {
	if !d.IsPrimary() {
		if msgs := d.cache.getHeight(d.BlockIndex); msgs != nil {
			for _, m := range msgs.prepare {
				d.OnReceive(m)
			}

			for _, m := range msgs.chViews {
				d.OnReceive(m)
			}

			for _, m := range msgs.commit {
				d.OnReceive(m)
			}
		}

		return
	}

	d.sendPrepareRequest()
}

func (d *DBFT) onPrepareRequest(msg payload.ConsensusPayload) {
	// ignore prepareRequest if we had already received it or
	// are in process of changing view
	if d.RequestSentOrReceived() { //|| (d.ViewChanging() && !d.MoreThanFNodesCommittedOrLost()) {
		d.Logger.Debug("ignoring PrepareRequest due to view changing",
			zap.Bool("sor", d.RequestSentOrReceived()),
			zap.Bool("viewChanging", d.ViewChanging()),
			zap.Bool("moreThanF", d.MoreThanFNodesCommittedOrLost()))

		return
	}

	if d.ViewNumber != msg.ViewNumber() {
		d.Logger.Debug("ignoring wrong view number", zap.Uint("view", uint(msg.ViewNumber())))
		return
	} else if uint(msg.ValidatorIndex()) != d.GetPrimaryIndex(d.ViewNumber) {
		d.Logger.Debug("ignoring PrepareRequest from wrong node", zap.Uint16("from", msg.ValidatorIndex()))
		return
	}

	if err := d.VerifyPrepareRequest(msg); err != nil {
		// We should change view if we receive signed PrepareRequest from the expected validator but it is invalid.
		d.Logger.Warn("invalid PrepareRequest", zap.Uint16("from", msg.ValidatorIndex()), zap.String("error", err.Error()))
		d.sendChangeView(payload.CVBlockRejectedByPolicy)
		return
	}

	d.extendTimer(2)

	p := msg.GetPrepareRequest()
	if len(p.TransactionHashes()) == 0 {
		d.Logger.Debug("received empty PrepareRequest")
	}

	d.Timestamp = p.Timestamp()
	d.Nonce = p.Nonce()
	d.NextConsensus = p.NextConsensus()
	d.TransactionHashes = p.TransactionHashes()

	d.Logger.Info("received PrepareRequest", zap.Uint16("validator", msg.ValidatorIndex()), zap.Int("tx", len(d.TransactionHashes)))
	d.processMissingTx()
	d.updateExistingPayloads(msg)
	d.PreparationPayloads[msg.ValidatorIndex()] = msg

	if !d.hasAllTransactions() || !d.createAndCheckBlock() || d.Context.WatchOnly() {
		return
	}

	d.sendPrepareResponse()
	d.checkPrepare()
}

func (d *DBFT) processMissingTx() {
	missing := make([]common.Hash, 0, len(d.TransactionHashes)/2)

	for _, h := range d.TransactionHashes {
		if _, ok := d.Transactions[h]; ok {
			continue
		}
		if tx := d.GetTx(h); tx == nil {
			missing = append(missing, h)
		} else {
			d.Transactions[h] = tx
		}
	}

	if len(missing) != 0 {
		d.MissingTransactions = missing
		d.Logger.Info("missing tx",
			zap.Int("count", len(missing)))
		d.RequestTx(missing...)
	}
}

// createAndCheckBlock is a prepareRequest-level helper that creates and checks
// the new proposed block, if it's fine it returns true, if something is wrong
// with it it send a changeView request and returns false. It's only valid to
// call it when all transactions for this block are already collected.
func (d *DBFT) createAndCheckBlock() bool {
	if d.NextConsensus != d.GetConsensusAddress(d.GetValidators(d.BlockIndex+1)...) {
		d.Logger.Error("invalid nextConsensus in proposed block")
		d.sendChangeView(payload.CVBlockRejectedByPolicy)
		return false
	}
	if b := d.Context.CreateBlock(); !d.VerifyBlock(b) {
		d.Logger.Warn("proposed block fails verification")
		d.sendChangeView(payload.CVTxInvalid)
		return false
	}
	return true
}

func (d *DBFT) updateExistingPayloads(msg payload.ConsensusPayload) {
	for i, m := range d.PreparationPayloads {
		if m != nil && m.Type() == payload.PrepareResponseType {
			resp := m.GetPrepareResponse()
			if resp != nil && resp.PreparationHash() != msg.Hash() {
				d.PreparationPayloads[i] = nil
			}
		}
	}

	for i, m := range d.CommitPayloads {
		if m != nil && m.ViewNumber() == d.ViewNumber {
			if header := d.MakeHeader(); header != nil {
				pub := d.Validators[m.ValidatorIndex()]
				if header.Verify(pub, m.GetCommit().Signature()) != nil {
					d.CommitPayloads[i] = nil
					d.Logger.Warn("can't validate commit signature")
				}
			}
		}
	}
}

func (d *DBFT) onPrepareResponse(msg payload.ConsensusPayload) {
	if d.ViewNumber != msg.ViewNumber() {
		d.Logger.Debug("ignoring wrong view number", zap.Uint("view", uint(msg.ViewNumber())))
		return
	} else if uint(msg.ValidatorIndex()) == d.GetPrimaryIndex(d.ViewNumber) {
		d.Logger.Debug("ignoring PrepareResponse from primary node", zap.Uint16("from", msg.ValidatorIndex()))
		return
	}

	// ignore PrepareResponse if in process of changing view
	m := d.PreparationPayloads[msg.ValidatorIndex()]
	if m != nil || d.ViewChanging() && !d.MoreThanFNodesCommittedOrLost() {
		d.Logger.Debug("ignoring PrepareResponse because of view changing")
		return
	}

	if err := d.VerifyPrepareResponse(msg); err != nil {
		d.Logger.Warn("invalid PrepareResponse", zap.Uint16("from", msg.ValidatorIndex()), zap.String("error", err.Error()))
		return
	}
	d.Logger.Info("received PrepareResponse", zap.Uint16("validator", msg.ValidatorIndex()))
	d.PreparationPayloads[msg.ValidatorIndex()] = msg

	if m = d.PreparationPayloads[d.GetPrimaryIndex(d.ViewNumber)]; m != nil {
		req := m.GetPrepareRequest()
		if req == nil {
			d.Logger.DPanic("unexpected nil prepare request")
			return
		}

		prepHash := msg.GetPrepareResponse().PreparationHash()
		if h := m.Hash(); prepHash != h {
			d.PreparationPayloads[msg.ValidatorIndex()] = nil
			d.Logger.Debug("hash mismatch",
				zap.Stringer("primary", h),
				zap.Stringer("received", prepHash))

			return
		}
	}

	d.extendTimer(2)

	if !d.Context.WatchOnly() && !d.CommitSent() && d.RequestSentOrReceived() {
		d.checkPrepare()
	}
}

func (d *DBFT) onChangeView(msg payload.ConsensusPayload) {
	p := msg.GetChangeView()

	if p.NewViewNumber() <= d.ViewNumber {
		d.Logger.Debug("ignoring old ChangeView", zap.Uint("new_view", uint(p.NewViewNumber())))
		d.onRecoveryRequest(msg)

		return
	}

	if d.CommitSent() {
		d.Logger.Debug("ignoring ChangeView: commit sent")
		d.sendRecoveryMessage()
		return
	}

	m := d.ChangeViewPayloads[msg.ValidatorIndex()]
	if m != nil && p.NewViewNumber() < m.GetChangeView().NewViewNumber() {
		return
	}

	d.Logger.Info("received ChangeView",
		zap.Uint("validator", uint(msg.ValidatorIndex())),
		zap.Stringer("reason", p.Reason()),
		zap.Uint("new view", uint(p.NewViewNumber())),
	)

	d.ChangeViewPayloads[msg.ValidatorIndex()] = msg
	d.checkChangeView(p.NewViewNumber())
}

func (d *DBFT) onCommit(msg payload.ConsensusPayload) {
	if d.ViewNumber == msg.ViewNumber() {
		d.Logger.Info("received Commit", zap.Uint("validator", uint(msg.ValidatorIndex())))
		d.extendTimer(4)
		header := d.MakeHeader()
		if header == nil {
			d.CommitPayloads[msg.ValidatorIndex()] = msg
		} else {
			pub := d.Validators[msg.ValidatorIndex()]
			if header.Verify(pub, msg.GetCommit().Signature()) == nil {
				d.CommitPayloads[msg.ValidatorIndex()] = msg
				d.checkCommit()
			} else {
				d.Logger.Warn("invalid commit signature",
					zap.Uint("validator", uint(msg.ValidatorIndex())),
				)
			}
		}

		return
	}

	d.Logger.Info("received commit for different view",
		zap.Uint("validator", uint(msg.ValidatorIndex())),
		zap.Uint("view", uint(msg.ViewNumber())),
	)
	d.CommitPayloads[msg.ValidatorIndex()] = msg
}

func (d *DBFT) onRecoveryRequest(msg payload.ConsensusPayload) {
	if !d.CommitSent() {
		// Limit recoveries to be sent from no more than F nodes
		// TODO replace loop with a single if
		shouldSend := false

		for i := 1; i <= d.F(); i++ {
			ind := (int(msg.ValidatorIndex()) + i) % len(d.Validators)
			if ind == d.MyIndex {
				shouldSend = true
				break
			}
		}

		if !shouldSend {
			return
		}
	}

	d.sendRecoveryMessage()
}

func (d *DBFT) onRecoveryMessage(msg payload.ConsensusPayload) {
	d.Logger.Debug("recovery message received", zap.Any("dump", msg))

	var (
		validPrepResp, validChViews, validCommits int
		validPrepReq, totalPrepReq                int
	)

	recovery := msg.GetRecoveryMessage()
	total := len(d.Validators)

	// isRecovering is always set to false again after OnRecoveryMessageReceived
	d.recovering = true

	defer func() {
		d.Logger.Sugar().Debugf("recovering finished cv=%d/%d preq=%d/%d presp=%d/%d co=%d/%d",
			validChViews, total,
			validPrepReq, totalPrepReq,
			validPrepResp, total,
			validCommits, total)
		d.recovering = false
	}()

	if msg.ViewNumber() > d.ViewNumber {
		if d.CommitSent() {
			return
		}

		for _, m := range recovery.GetChangeViews(msg, d.Validators) {
			validChViews++
			d.OnReceive(m)
		}
	}

	if msg.ViewNumber() == d.ViewNumber && !(d.ViewChanging() && !d.MoreThanFNodesCommittedOrLost()) && !d.CommitSent() {
		if !d.RequestSentOrReceived() {
			prepReq := recovery.GetPrepareRequest(msg, d.Validators, uint16(d.PrimaryIndex))
			if prepReq != nil {
				totalPrepReq, validPrepReq = 1, 1
				d.OnReceive(prepReq)
			} else if d.IsPrimary() {
				d.sendPrepareRequest()
			}
		}

		for _, m := range recovery.GetPrepareResponses(msg, d.Validators) {
			validPrepResp++
			d.OnReceive(m)
		}
	}

	if msg.ViewNumber() <= d.ViewNumber {
		// Ensure we know about all commits from lower view numbers.
		for _, m := range recovery.GetCommits(msg, d.Validators) {
			validCommits++
			d.OnReceive(m)
		}
	}
}

func (d *DBFT) changeTimer(delay time.Duration) {
	d.Logger.Debug("reset timer",
		zap.Uint32("h", d.BlockIndex),
		zap.Int("v", int(d.ViewNumber)),
		zap.Duration("delay", delay))
	d.Timer.Reset(timer.HV{Height: d.BlockIndex, View: d.ViewNumber}, delay)
}

func (d *DBFT) extendTimer(count time.Duration) {
	if !d.CommitSent() && !d.ViewChanging() {
		d.Timer.Extend(count * d.SecondsPerBlock / time.Duration(d.M()))
	}
}
