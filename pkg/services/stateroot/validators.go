package stateroot

import (
	"time"

	"github.com/multiverse-dev/saiya/pkg/core/state"
	"github.com/multiverse-dev/saiya/pkg/core/transaction"
	"github.com/multiverse-dev/saiya/pkg/io"
	"github.com/multiverse-dev/saiya/pkg/network/payload"
	"github.com/multiverse-dev/saiya/pkg/wallet"
	"go.uber.org/zap"
)

const (
	voteValidEndInc      = 10
	firstVoteResendDelay = 3 * time.Second
)

// Start runs service instance in a separate goroutine.
func (s *service) Start() {
	s.log.Info("starting state validation service")
	s.chain.SubscribeForBlocks(s.blockCh)
	go s.run()
}

func (s *service) run() {
runloop:
	for {
		select {
		case b := <-s.blockCh:
			r, err := s.GetStateRoot(b.Index)
			if err != nil {
				s.log.Error("can't get state root for new block", zap.Error(err))
			} else if err := s.signAndSend(r); err != nil {
				s.log.Error("can't sign or send state root", zap.Error(err))
			}
			s.srMtx.Lock()
			delete(s.incompleteRoots, b.Index-voteValidEndInc)
			s.srMtx.Unlock()
		case <-s.done:
			break runloop
		}
	}
drainloop:
	for {
		select {
		case <-s.blockCh:
		default:
			break drainloop
		}
	}
}

// Shutdown stops the service.
func (s *service) Shutdown() {
	s.chain.UnsubscribeFromBlocks(s.blockCh)
	close(s.done)
}

func (s *service) signAndSend(r *state.MPTRoot) error {
	if !s.MainCfg.Enabled {
		return nil
	}

	myIndex, acc := s.getAccount()
	if acc == nil {
		return nil
	}

	sig := acc.PrivateKey().SignHashable(s.ChainID, r)
	incRoot := s.getIncompleteRoot(r.Index, myIndex)
	incRoot.Lock()
	defer incRoot.Unlock()
	incRoot.root = r
	incRoot.addSignature(acc.PrivateKey().PublicKey(), sig)
	incRoot.reverify(s.ChainID)
	s.trySendRoot(incRoot, acc)

	msg := NewMessage(VoteT, &Vote{
		ValidatorIndex: int32(myIndex),
		Height:         r.Index,
		Signature:      sig,
	})

	w := io.NewBufBinWriter()
	msg.EncodeBinary(w.BinWriter)
	if w.Err != nil {
		return w.Err
	}
	e := &payload.Extensible{
		Category:        Category,
		ValidBlockStart: r.Index,
		ValidBlockEnd:   r.Index + voteValidEndInc,
		Sender:          acc.Address,
		Data:            w.Bytes(),
		Witness:         transaction.Witness{},
	}
	sig = acc.PrivateKey().SignHashable(s.ChainID, e)
	e.Witness.VerificationScript = acc.PrivateKey().PublicKey().CreateVerificationScript()
	e.Witness.InvocationScript = sig
	incRoot.myVote = e
	incRoot.retries = -1
	s.sendVote(incRoot)
	return nil
}

// sendVote attempts to send vote if it's still valid and if stateroot message
// was not sent yet. It must be called with ir locked.
func (s *service) sendVote(ir *incompleteRoot) {
	if ir.isSent || ir.retries >= s.maxRetries ||
		s.chain.HeaderHeight() >= ir.myVote.ValidBlockEnd {
		return
	}
	s.relayExtensible(ir.myVote)
	delay := firstVoteResendDelay
	if ir.retries > 0 {
		delay = s.timePerBlock << ir.retries
	}
	_ = time.AfterFunc(delay, func() {
		ir.Lock()
		s.sendVote(ir)
		ir.Unlock()
	})
	ir.retries++
}

// getAccount returns current index and account for the node running this service.
func (s *service) getAccount() (byte, *wallet.Account) {
	s.accMtx.RLock()
	defer s.accMtx.RUnlock()
	return s.myIndex, s.acc
}
