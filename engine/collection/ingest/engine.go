// Package ingest implements an engine for receiving transactions that need
// to be packaged into a collection.
package ingest

import (
	"errors"
	"fmt"

	"github.com/onflow/cadence/runtime/parser2"
	"github.com/rs/zerolog"

	"github.com/dapperlabs/flow-go/engine"
	"github.com/dapperlabs/flow-go/model/flow"
	"github.com/dapperlabs/flow-go/model/flow/filter"
	"github.com/dapperlabs/flow-go/module"
	"github.com/dapperlabs/flow-go/module/mempool"
	"github.com/dapperlabs/flow-go/module/metrics"
	"github.com/dapperlabs/flow-go/network"
	"github.com/dapperlabs/flow-go/state/protocol"
	"github.com/dapperlabs/flow-go/storage"
	"github.com/dapperlabs/flow-go/utils/logging"
)

// Engine is the transaction ingestion engine, which ensures that new
// transactions are delegated to the correct collection cluster, and prepared
// to be included in a collection.
type Engine struct {
	unit       *engine.Unit
	log        zerolog.Logger
	engMetrics module.EngineMetrics
	colMetrics module.CollectionMetrics
	con        network.Conduit
	me         module.Local
	state      protocol.State
	pool       mempool.Transactions

	config Config
}

// New creates a new collection ingest engine.
func New(
	log zerolog.Logger,
	net module.Network,
	state protocol.State,
	engMetrics module.EngineMetrics,
	colMetrics module.CollectionMetrics,
	me module.Local,
	pool mempool.Transactions,
	config Config,
) (*Engine, error) {

	logger := log.With().Str("engine", "ingest").Logger()

	e := &Engine{
		unit:       engine.NewUnit(),
		log:        logger,
		engMetrics: engMetrics,
		colMetrics: colMetrics,
		me:         me,
		state:      state,
		pool:       pool,
		config:     config,
	}

	con, err := net.Register(engine.PushTransactions, e)
	if err != nil {
		return nil, fmt.Errorf("could not register engine: %w", err)
	}

	e.con = con

	return e, nil
}

// Ready returns a ready channel that is closed once the engine has fully
// started.
func (e *Engine) Ready() <-chan struct{} {
	return e.unit.Ready()
}

// Done returns a done channel that is closed once the engine has fully stopped.
func (e *Engine) Done() <-chan struct{} {
	return e.unit.Done()
}

// SubmitLocal submits an event originating on the local node.
func (e *Engine) SubmitLocal(event interface{}) {
	e.Submit(e.me.NodeID(), event)
}

// Submit submits the given event from the node with the given origin ID
// for processing in a non-blocking manner. It returns instantly and logs
// a potential processing error internally when done.
func (e *Engine) Submit(originID flow.Identifier, event interface{}) {
	e.unit.Launch(func() {
		err := e.process(originID, event)
		if err != nil {
			engine.LogError(e.log, err)
		}
	})
}

// ProcessLocal processes an event originating on the local node.
func (e *Engine) ProcessLocal(event interface{}) error {
	return e.Process(e.me.NodeID(), event)
}

// Process processes the given event from the node with the given origin ID in
// a blocking manner. It returns the potential processing error when done.
func (e *Engine) Process(originID flow.Identifier, event interface{}) error {
	return e.unit.Do(func() error {
		return e.process(originID, event)
	})
}

// process processes engine events.
//
// Transactions are validated and routed to the correct cluster, then added
// to the transaction mempool.
func (e *Engine) process(originID flow.Identifier, event interface{}) error {
	switch ev := event.(type) {
	case *flow.TransactionBody:
		e.engMetrics.MessageReceived(metrics.EngineCollectionIngest, metrics.MessageTransaction)
		defer e.engMetrics.MessageHandled(metrics.EngineCollectionIngest, metrics.MessageTransaction)
		return e.onTransaction(originID, ev)
	default:
		return fmt.Errorf("invalid event type (%T)", event)
	}
}

// onTransaction handles receipt of a new transaction. This can be submitted
// from outside the system or routed from another collection node.
func (e *Engine) onTransaction(originID flow.Identifier, tx *flow.TransactionBody) error {

	log := e.log.With().
		Hex("origin_id", originID[:]).
		Hex("tx_id", logging.Entity(tx)).
		Hex("ref_block_id", tx.ReferenceBlockID[:]).
		Logger()

	// TODO log the reference block and final height for debug purposes
	{
		final, err := e.state.Final().Head()
		if err != nil {
			return fmt.Errorf("could not get final height: %w", err)
		}
		log = log.With().Uint64("final_height", final.Height).Logger()
		ref, err := e.state.AtBlockID(tx.ReferenceBlockID).Head()
		if err == nil {
			log = log.With().Uint64("ref_block_height", ref.Height).Logger()
		}
	}

	log.Info().Msg("transaction message received")

	// short-circuit if we have already stored the transaction
	if e.pool.Has(tx.ID()) {
		e.log.Debug().Msg("received dupe transaction")
		return nil
	}

	// first, we check if the transaction is valid
	err := e.validateTransaction(tx)
	if err != nil {
		return engine.NewInvalidInputErrorf("invalid transaction: %w", err)
	}

	// retrieve the set of collector clusters
	clusters, err := e.state.Final().Clusters()
	if err != nil {
		return fmt.Errorf("could not cluster collection nodes: %w", err)
	}

	// get the locally assigned cluster and the cluster responsible for the transaction
	txCluster, ok := clusters.ByTxID(tx.ID())
	if !ok {
		return fmt.Errorf("could not get local cluster by txID: %x", tx.ID())
	}

	localID := e.me.NodeID()
	localCluster, _, ok := clusters.ByNodeID(localID)
	if !ok {
		return fmt.Errorf("could not get local cluster")
	}

	log = log.With().
		Hex("local_cluster", logging.ID(localCluster.Fingerprint())).
		Hex("tx_cluster", logging.ID(txCluster.Fingerprint())).
		Logger()

	// if our cluster is responsible for the transaction, add it to the mempool
	if localCluster.Fingerprint() == txCluster.Fingerprint() {
		_ = e.pool.Add(tx)
		e.colMetrics.TransactionIngested(tx.ID())
		log.Debug().Msg("added transaction to pool")
	}

	// if the message was submitted internally (ie. via the Access API)
	// propagate it to all members of the responsible cluster
	if originID == localID {

		// always send the transaction to one node in the responsible cluster
		// send to additional nodes based on configuration
		targetIDs := txCluster.
			Filter(filter.Not(filter.HasNodeID(localID))).
			Sample(e.config.PropagationRedundancy + 1)

		log.Debug().
			Str("recipients", fmt.Sprintf("%v", targetIDs.NodeIDs())).
			Msg("propagating transaction to cluster")

		err = e.con.Submit(tx, targetIDs.NodeIDs()...)
		if err != nil {
			return fmt.Errorf("could not route transaction to cluster: %w", err)
		}

		e.engMetrics.MessageSent(metrics.EngineCollectionIngest, metrics.MessageTransaction)
	}

	log.Info().Msg("transaction processed")

	return nil
}

// validateTransaction validates the transaction in order to determine whether
// the transaction should be included in a collection.
func (e *Engine) validateTransaction(tx *flow.TransactionBody) error {

	// ensure all required fields are set
	missingFields := tx.MissingFields()
	if len(missingFields) > 0 {
		return IncompleteTransactionError{Missing: missingFields}
	}

	// ensure the gas limit is not over the maximum
	if tx.GasLimit > e.config.MaxGasLimit {
		return GasLimitExceededError{Actual: tx.GasLimit, Maximum: e.config.MaxGasLimit}
	}

	// ensure the reference block is valid
	err := e.checkTransactionExpiry(tx)
	if err != nil {
		return err
	}

	if e.config.CheckScriptsParse {
		// ensure the script is at least parse-able
		_, err = parser2.ParseProgram(string(tx.Script))
		if err != nil {
			return InvalidScriptError{ParserErr: err}
		}
	}

	// TODO check account/payer signatures

	return nil
}

// checkTransactionExpiry checks whether a transaction's reference block ID is
// valid. Returns nil if the reference is valid, returns an error if the
// reference is invalid or we failed to check it.
func (e *Engine) checkTransactionExpiry(tx *flow.TransactionBody) error {

	// look up the reference block
	ref, err := e.state.AtBlockID(tx.ReferenceBlockID).Head()
	if errors.Is(err, storage.ErrNotFound) {
		// the transaction references an unknown block - at this point we decide
		// whether to consider it expired based on configuration
		if e.config.AllowUnknownReference {
			return nil
		}
		return ErrUnknownReferenceBlock
	}
	if err != nil {
		return fmt.Errorf("could not get reference block: %w", err)
	}

	// get the latest finalized block we know about
	final, err := e.state.Final().Head()
	if err != nil {
		return fmt.Errorf("could not get finalized header: %w", err)
	}

	diff := final.Height - ref.Height
	// check for overflow
	if ref.Height > final.Height {
		diff = 0
	}

	// discard transactions that are expired, or that will expire sooner than
	// our configured buffer allows
	if uint(diff) > flow.DefaultTransactionExpiry-e.config.ExpiryBuffer {
		return ExpiredTransactionError{
			RefHeight:   ref.Height,
			FinalHeight: final.Height,
		}
	}

	return nil
}
