package consensus

import (
	"errors"
	"fmt"
	"sort"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/multiverse-dev/saiya/pkg/config"
	coreb "github.com/multiverse-dev/saiya/pkg/core/block"
	"github.com/multiverse-dev/saiya/pkg/core/blockchainer"
	"github.com/multiverse-dev/saiya/pkg/core/mempool"
	"github.com/multiverse-dev/saiya/pkg/core/transaction"
	cc "github.com/multiverse-dev/saiya/pkg/crypto"
	"github.com/multiverse-dev/saiya/pkg/crypto/hash"
	"github.com/multiverse-dev/saiya/pkg/crypto/keys"
	"github.com/multiverse-dev/saiya/pkg/dbft"
	"github.com/multiverse-dev/saiya/pkg/dbft/block"
	"github.com/multiverse-dev/saiya/pkg/dbft/payload"
	npayload "github.com/multiverse-dev/saiya/pkg/network/payload"
	"github.com/multiverse-dev/saiya/pkg/wallet"
	"go.uber.org/atomic"
	"go.uber.org/zap"
)

// cacheMaxCapacity is the default cache capacity taken
const cacheMaxCapacity = 100

const defaultTimePerBlock = 15 * time.Second

// Number of nanoseconds in millisecond.
const nsInMs = 1000000

// Category is message category for extensible payloads.
const Category = "dBFT"

// Ledger is the interface to Blockchain sufficient for Service.
type Ledger interface {
	AddBlock(block *coreb.Block) error
	ApplyPolicyToTxSet([]*transaction.Transaction) []*transaction.Transaction
	GetConfig() config.ProtocolConfiguration
	GetMemPool() *mempool.Pool
	GetStateModule() blockchainer.StateRoot
	GetTransaction(common.Hash) (*transaction.Transaction, uint32, error)
	GetValidators(uint32) ([]*keys.PublicKey, error)
	PoolTx(t *transaction.Transaction, pools ...*mempool.Pool) error
	SubscribeForBlocks(ch chan<- *coreb.Block)
	UnsubscribeFromBlocks(ch chan<- *coreb.Block)
	BlockHeight() uint32
	CurrentBlockHash() common.Hash
	GetBlock(hash common.Hash, full bool) (*coreb.Block, error)
	GetHeaderHash(int) common.Hash
	mempool.Feer
}

// Service represents consensus instance.
type Service interface {
	// Start initializes dBFT and starts event loop for consensus service.
	// It must be called only when sufficient amount of peers are connected.
	Start()
	// Shutdown stops dBFT event loop.
	Shutdown()

	// OnPayload is a callback to notify Service about new received payload.
	OnPayload(p *npayload.Extensible) error
	// OnTransaction is a callback to notify Service about new received transaction.
	OnTransaction(tx *transaction.Transaction)
}

type service struct {
	Config

	log *zap.Logger
	// txx is a fifo cache which stores miner transactions.
	txx  *relayCache
	dbft *dbft.DBFT
	// messages and transactions are channels needed to process
	// everything in single thread.
	messages     chan Payload
	transactions chan *transaction.Transaction
	// blockEvents is used to pass a new block event to the consensus
	// process.
	blockEvents  chan *coreb.Block
	lastProposal []common.Hash
	wallet       *wallet.Wallet
	// started is a flag set with Start method that runs an event handling
	// goroutine.
	started  *atomic.Bool
	quit     chan struct{}
	finished chan struct{}
	// lastTimestamp contains timestamp for the last processed block.
	// We can't rely on timestamp from dbft context because it is changed
	// before block is accepted, so in case of change view it will contain
	// updated value.
	lastTimestamp uint64
}

// Config is a configuration for consensus services.
type Config struct {
	// Logger is a logger instance.
	Logger *zap.Logger
	// Broadcast is a callback which is called to notify server
	// about new consensus payload to sent.
	Broadcast func(p *npayload.Extensible)
	// Chain is a Ledger instance.
	Chain Ledger
	// ProtocolConfiguration contains protocol settings.
	ProtocolConfiguration config.ProtocolConfiguration
	// RequestTx is a callback to which will be called
	// when a node lacks transactions present in a block.
	RequestTx func(h ...common.Hash)
	// TimePerBlock minimal time that should pass before next block is accepted.
	TimePerBlock time.Duration
	// Wallet is a local-node wallet configuration.
	Wallet *config.Wallet
}

// NewService returns new consensus.Service instance.
func NewService(cfg Config) (Service, error) {
	if cfg.TimePerBlock <= 0 {
		cfg.TimePerBlock = defaultTimePerBlock
	}

	if cfg.Logger == nil {
		return nil, errors.New("empty logger")
	}

	srv := &service{
		Config: cfg,

		log:      cfg.Logger,
		txx:      newFIFOCache(cacheMaxCapacity),
		messages: make(chan Payload, 100),

		transactions: make(chan *transaction.Transaction, 100),
		blockEvents:  make(chan *coreb.Block, 1),
		started:      atomic.NewBool(false),
		quit:         make(chan struct{}),
		finished:     make(chan struct{}),
	}

	var err error

	if srv.wallet, err = wallet.NewWalletFromFile(cfg.Wallet.Path); err != nil {
		return nil, err
	}

	// Check that wallet password is correct for at least one account.
	var ok bool
	for _, acc := range srv.wallet.Accounts {
		err := acc.Decrypt(srv.Config.Wallet.Password, srv.wallet.Scrypt)
		if err == nil {
			ok = true
			break
		}
	}
	if !ok {
		return nil, errors.New("no account with provided password was found")
	}

	defer srv.wallet.Close()

	srv.dbft = dbft.New(
		dbft.WithLogger(srv.log),
		dbft.WithSecondsPerBlock(cfg.TimePerBlock),
		dbft.WithGetKeyPair(srv.getKeyPair),
		dbft.WithRequestTx(cfg.RequestTx),
		dbft.WithGetTx(srv.getTx),
		dbft.WithGetVerified(srv.getVerifiedTx),
		dbft.WithBroadcast(srv.broadcast),
		dbft.WithProcessBlock(srv.processBlock),
		dbft.WithVerifyBlock(srv.verifyBlock),
		dbft.WithGetBlock(srv.getBlock),
		dbft.WithWatchOnly(func() bool { return false }),
		dbft.WithNewBlockFromContext(srv.newBlockFromContext),
		dbft.WithCurrentHeight(cfg.Chain.BlockHeight),
		dbft.WithCurrentBlockHash(cfg.Chain.CurrentBlockHash),
		dbft.WithGetValidators(srv.getValidators),
		dbft.WithGetConsensusAddress(srv.getConsensusAddress),

		dbft.WithNewConsensusPayload(srv.newPayload),
		dbft.WithNewPrepareRequest(srv.newPrepareRequest),
		dbft.WithNewPrepareResponse(func() payload.PrepareResponse { return new(prepareResponse) }),
		dbft.WithNewChangeView(func() payload.ChangeView { return new(changeView) }),
		dbft.WithNewCommit(func() payload.Commit { return new(commit) }),
		dbft.WithNewRecoveryRequest(func() payload.RecoveryRequest { return new(recoveryRequest) }),
		dbft.WithNewRecoveryMessage(func() payload.RecoveryMessage {
			return &recoveryMessage{}
		}),
		dbft.WithVerifyPrepareRequest(srv.verifyRequest),
		dbft.WithVerifyPrepareResponse(func(_ payload.ConsensusPayload) error { return nil }),
	)

	if srv.dbft == nil {
		return nil, errors.New("can't initialize dBFT")
	}

	return srv, nil
}

var (
	_ block.Transaction = (*transaction.Transaction)(nil)
	_ block.Block       = (*consensusBlock)(nil)
)

// NewPayload creates new consensus payload for the provided network.
func NewPayload(m uint64) *Payload {
	return &Payload{
		Extensible: npayload.Extensible{
			Category: Category,
		},
		message: message{},
		chainId: m,
	}
}

func (s *service) newPayload(c *dbft.Context, t payload.MessageType, msg interface{}) payload.ConsensusPayload {
	cp := NewPayload(s.ProtocolConfiguration.ChainID)
	cp.SetHeight(c.BlockIndex)
	cp.SetValidatorIndex(uint16(c.MyIndex))
	cp.SetViewNumber(c.ViewNumber)
	cp.SetType(t)
	if pr, ok := msg.(*prepareRequest); ok {
		pr.SetPrevHash(s.dbft.PrevHash)
		pr.SetVersion(s.dbft.Version)
	}
	cp.SetPayload(msg)

	cp.Extensible.ValidBlockStart = 0
	cp.Extensible.ValidBlockEnd = c.BlockIndex
	cp.Extensible.Sender = c.Validators[c.MyIndex].Address()

	return cp
}

func (s *service) newPrepareRequest() payload.PrepareRequest {
	r := new(prepareRequest)
	return r
}

func (s *service) Start() {
	if s.started.CAS(false, true) {
		s.log.Info("starting consensus service")
		s.dbft.Start()
		s.Chain.SubscribeForBlocks(s.blockEvents)
		go s.eventLoop()
	}
}

// Shutdown implements Service interface.
func (s *service) Shutdown() {
	if s.started.Load() {
		close(s.quit)
		<-s.finished
	}
}

func (s *service) eventLoop() {
events:
	for {
		select {
		case <-s.quit:
			s.dbft.Timer.Stop()
			break events
		case <-s.dbft.Timer.C():
			hv := s.dbft.Timer.HV()
			s.log.Debug("timer fired",
				zap.Uint32("height", hv.Height),
				zap.Uint("view", uint(hv.View)))
			s.dbft.OnTimeout(hv)
		case msg := <-s.messages:
			fields := []zap.Field{
				zap.Uint8("from", msg.message.ValidatorIndex),
				zap.Stringer("type", msg.Type()),
			}

			if msg.Type() == payload.RecoveryMessageType {
				rec := msg.GetRecoveryMessage().(*recoveryMessage)
				if rec.preparationHash == nil {
					req := rec.GetPrepareRequest(&msg, s.dbft.Validators, uint16(s.dbft.PrimaryIndex))
					if req != nil {
						h := req.Hash()
						rec.preparationHash = &h
					}
				}

				fields = append(fields,
					zap.Int("#preparation", len(rec.preparationPayloads)),
					zap.Int("#commit", len(rec.commitPayloads)),
					zap.Int("#changeview", len(rec.changeViewPayloads)),
					zap.Bool("#request", rec.prepareRequest != nil),
					zap.Bool("#hash", rec.preparationHash != nil))
			}

			s.log.Debug("received message", fields...)
			s.dbft.OnReceive(&msg)
		case tx := <-s.transactions:
			s.dbft.OnTransaction(tx)
		case b := <-s.blockEvents:
			s.handleChainBlock(b)
		}
		// Always process block event if there is any, we can add one above.
		select {
		case b := <-s.blockEvents:
			s.handleChainBlock(b)
		default:
		}
	}
	close(s.finished)
}

func (s *service) handleChainBlock(b *coreb.Block) {
	// We can get our own block here, so check for index.
	if b.Index >= s.dbft.BlockIndex {
		s.log.Debug("new block in the chain",
			zap.Uint32("dbft index", s.dbft.BlockIndex),
			zap.Uint32("chain index", s.Chain.BlockHeight()))
		s.postBlock(b)
		s.dbft.InitializeConsensus(0)
	}
}

func (s *service) validatePayload(p *Payload) bool {
	validators := s.getValidators(p.BlockIndex)
	if int(p.message.ValidatorIndex) >= len(validators) {
		return false
	}

	pub := validators[p.message.ValidatorIndex]
	h := pub.Address()
	return p.Sender == h
}

func (s *service) getKeyPair(pubs []*keys.PublicKey) (int, *keys.PrivateKey, *keys.PublicKey) {
	for i := range pubs {
		sh := pubs[i].Address()
		acc := s.wallet.GetAccount(sh)
		if acc == nil {
			continue
		}

		key := acc.PrivateKey()
		if acc.PrivateKey() == nil {
			err := acc.Decrypt(s.Config.Wallet.Password, s.wallet.Scrypt)
			if err != nil {
				s.log.Fatal("can't unlock account", zap.String("address", sh.String()))
				break
			}
			key = acc.PrivateKey()
		}

		return i, key, key.PublicKey()
	}

	return -1, nil, nil
}

func (s *service) payloadFromExtensible(ep *npayload.Extensible) *Payload {
	return &Payload{
		Extensible: *ep,
		message:    message{},
	}
}

// OnPayload handles Payload receive.
func (s *service) OnPayload(cp *npayload.Extensible) error {
	log := s.log.With(zap.Stringer("hash", cp.Hash()))
	p := s.payloadFromExtensible(cp)
	// decode payload data into message
	if err := p.decodeData(); err != nil {
		log.Info("can't decode payload data", zap.Error(err))
		return nil
	}

	if !s.validatePayload(p) {
		log.Info("can't validate payload")
		return nil
	}

	if s.dbft == nil || !s.started.Load() {
		log.Debug("dbft is inactive or not started yet")
		return nil
	}

	s.messages <- *p
	return nil
}

func (s *service) OnTransaction(tx *transaction.Transaction) {
	if s.dbft != nil {
		s.transactions <- tx
	}
}

func (s *service) broadcast(p payload.ConsensusPayload) {
	if err := p.(*Payload).Sign(s.dbft.Priv); err != nil {
		s.log.Warn("can't sign consensus payload", zap.Error(err))
	}

	ep := &p.(*Payload).Extensible
	s.Config.Broadcast(ep)
}

func (s *service) getTx(h common.Hash) block.Transaction {
	if tx := s.txx.Get(h); tx != nil {
		return tx.(*transaction.Transaction)
	}

	tx, _, _ := s.Config.Chain.GetTransaction(h)

	// this is needed because in case of absent tx dBFT expects to
	// get nil interface, not a nil pointer to any concrete type
	if tx != nil {
		return tx
	}

	return nil
}

func (s *service) verifyBlock(b block.Block) bool {
	coreb := &b.(*consensusBlock).Block

	if s.Chain.BlockHeight() >= coreb.Index {
		s.log.Warn("proposed block has already outdated")
		return false
	}
	if s.lastTimestamp >= coreb.Timestamp {
		s.log.Warn("proposed block has small timestamp",
			zap.Uint64("ts", coreb.Timestamp),
			zap.Uint64("last", s.lastTimestamp))
		return false
	}

	size := coreb.GetExpectedBlockSize()
	if size > int(s.ProtocolConfiguration.MaxBlockSize) {
		s.log.Warn("proposed block size exceeds config MaxBlockSize",
			zap.Uint32("max size allowed", s.ProtocolConfiguration.MaxBlockSize),
			zap.Int("block size", size))
		return false
	}

	var gas uint64
	var pool = mempool.New(len(coreb.Transactions), 0, false)
	var mainPool = s.Chain.GetMemPool()
	for _, tx := range coreb.Transactions {
		var err error

		gas += tx.Gas()
		if mainPool.ContainsKey(tx.Hash()) {
			err = pool.Add(tx, s.Chain)
			if err == nil {
				continue
			}
		} else {
			err = s.Chain.PoolTx(tx, pool)
		}
		if err != nil {
			s.log.Warn("invalid transaction in proposed block",
				zap.Stringer("hash", tx.Hash()),
				zap.Error(err))
			return false
		}
		if s.Chain.BlockHeight() >= coreb.Index {
			s.log.Warn("proposed block has already outdated")
			return false
		}
	}

	maxBlockGas := s.ProtocolConfiguration.MaxBlockGas
	if gas > maxBlockGas {
		s.log.Warn("proposed block system fee exceeds config MaxBlockSystemFee",
			zap.Int("max system fee allowed", int(maxBlockGas)),
			zap.Int("block system fee", int(gas)))
		return false
	}

	return true
}

var (
	errInvalidPrevHash          = errors.New("invalid PrevHash")
	errInvalidVersion           = errors.New("invalid Version")
	errInvalidTransactionsCount = errors.New("invalid transactions count")
)

func (s *service) verifyRequest(p payload.ConsensusPayload) error {
	req := p.GetPrepareRequest().(*prepareRequest)
	if req.prevHash != s.dbft.PrevHash {
		return errInvalidPrevHash
	}
	if req.version != s.dbft.Version {
		return errInvalidVersion
	}
	if len(req.TransactionHashes()) > int(s.ProtocolConfiguration.MaxTransactionsPerBlock) {
		return fmt.Errorf("%w: max = %d, got %d", errInvalidTransactionsCount, s.ProtocolConfiguration.MaxTransactionsPerBlock, len(req.TransactionHashes()))
	}
	// Save lastProposal for getVerified().
	s.lastProposal = req.transactionHashes

	return nil
}

func (s *service) processBlock(b block.Block) {
	bb := &b.(*consensusBlock).Block
	bb.Witness = s.getBlockWitness(bb)

	if err := s.Chain.AddBlock(bb); err != nil {
		// The block might already be added via the regular network
		// interaction.
		if _, errget := s.Chain.GetBlock(bb.Hash(), false); errget != nil {
			s.log.Warn("error on add block", zap.Error(err))
		}
	}
	s.postBlock(bb)
}

func (s *service) postBlock(b *coreb.Block) {
	if s.lastTimestamp < b.Timestamp {
		s.lastTimestamp = b.Timestamp
	}
	s.lastProposal = nil
}

func (s *service) getBlockWitness(b *coreb.Block) transaction.Witness {
	m := s.dbft.Context.M()
	dctx := s.dbft.Context
	pubs := dctx.Validators
	sigs := make([][]byte, m)
	sort.Sort(keys.PublicKeys(pubs))
	j := 0
	for i := range pubs {
		if p := dctx.CommitPayloads[i]; p != nil && p.ViewNumber() == dctx.ViewNumber {
			sigs[j] = p.GetCommit().Signature()
			j++
			if j == m {
				break
			}
		}
	}
	verif, _ := keys.PublicKeys(pubs).CreateDefaultMultiSigRedeemScript()
	return transaction.Witness{
		VerificationScript: verif,
		InvocationScript:   cc.CreateMultiInvocationScript(sigs),
	}
}

func (s *service) getBlock(h common.Hash) block.Block {
	b, err := s.Chain.GetBlock(h, false)
	if err != nil {
		return nil
	}

	return &consensusBlock{chainId: s.ProtocolConfiguration.ChainID, Block: *b}
}

func (s *service) getVerifiedTx() []block.Transaction {
	pool := s.Config.Chain.GetMemPool()

	var txx []*transaction.Transaction

	if s.dbft.ViewNumber > 0 && len(s.lastProposal) > 0 {
		txx = make([]*transaction.Transaction, 0, len(s.lastProposal))
		for i := range s.lastProposal {
			if tx, ok := pool.TryGetValue(s.lastProposal[i]); ok {
				txx = append(txx, tx)
			}
		}

		if len(txx) < len(s.lastProposal)/2 {
			txx = pool.GetVerifiedTransactions()
		}
	} else {
		txx = pool.GetVerifiedTransactions()
	}

	if len(txx) > 0 {
		txx = s.Config.Chain.ApplyPolicyToTxSet(txx)
	}

	res := make([]block.Transaction, len(txx))
	for i := range txx {
		res[i] = txx[i]
	}

	return res
}

func (s *service) getValidators(index uint32) []*keys.PublicKey {
	var (
		pKeys []*keys.PublicKey
		err   error
	)

	pKeys, err = s.Chain.GetValidators(index)
	if err != nil {
		s.log.Error("error while trying to get validators", zap.Error(err))
	}
	return pKeys
}

func (s *service) getConsensusAddress(validators ...*keys.PublicKey) common.Address {
	script, err := keys.PublicKeys(validators).CreateDefaultMultiSigRedeemScript()
	if err != nil {
		s.log.Fatal(fmt.Sprintf("failed to create multisignature script: %s", err.Error()))
	}
	return hash.Hash160(script)
}

func (s *service) newBlockFromContext(ctx *dbft.Context) block.Block {
	block := &consensusBlock{chainId: s.ProtocolConfiguration.ChainID}

	block.Block.Timestamp = ctx.Timestamp / nsInMs
	block.Block.Nonce = ctx.Nonce
	block.Block.Index = ctx.BlockIndex

	block.Block.PrevHash = ctx.PrevHash
	block.Block.Version = ctx.Version

	validators, err := s.Chain.GetValidators(ctx.BlockIndex + 1)
	if err != nil {
		s.log.Fatal(fmt.Sprintf("failed to get validators: %s", err.Error()))
	}
	script, err := keys.PublicKeys(validators).CreateDefaultMultiSigRedeemScript()
	if err != nil {
		s.log.Fatal(fmt.Sprintf("failed to create multisignature script: %s", err.Error()))
	}
	block.Block.NextConsensus = hash.Hash160(script)

	primaryIndex := byte(ctx.PrimaryIndex)
	block.Block.PrimaryIndex = primaryIndex

	// it's OK to have ctx.TransactionsHashes == nil here
	hashes := make([]common.Hash, len(ctx.TransactionHashes))
	copy(hashes, ctx.TransactionHashes)
	block.Block.MerkleRoot = hash.CalcMerkleRoot(hashes)

	return block
}
