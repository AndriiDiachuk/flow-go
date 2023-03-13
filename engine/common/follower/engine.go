package follower

import (
	"fmt"

	"github.com/rs/zerolog"

	"github.com/onflow/flow-go/consensus/hotstuff/model"
	"github.com/onflow/flow-go/consensus/hotstuff/tracker"
	"github.com/onflow/flow-go/engine"
	"github.com/onflow/flow-go/engine/common"
	"github.com/onflow/flow-go/engine/common/fifoqueue"
	"github.com/onflow/flow-go/engine/consensus"
	"github.com/onflow/flow-go/model/flow"
	"github.com/onflow/flow-go/model/messages"
	"github.com/onflow/flow-go/module"
	"github.com/onflow/flow-go/module/component"
	"github.com/onflow/flow-go/module/irrecoverable"
	"github.com/onflow/flow-go/module/metrics"
	"github.com/onflow/flow-go/network"
	"github.com/onflow/flow-go/network/channels"
	"github.com/onflow/flow-go/storage"
)

type EngineOption func(*Engine)

// WithChannel sets the channel the follower engine will use to receive blocks.
func WithChannel(channel channels.Channel) EngineOption {
	return func(e *Engine) {
		e.channel = channel
	}
}

// defaultBlockQueueCapacity maximum capacity of inbound queue for `messages.BlockProposal`s
const defaultBlockQueueCapacity = 10_000

// Engine follows and maintains the local copy of the protocol state. It is a
// passive (read-only) version of the compliance engine. The compliance engine
// is employed by consensus nodes (active consensus participants) where the
// Follower engine is employed by all other node roles.
// Implements consensus.Compliance interface.
type Engine struct {
	*component.ComponentManager
	log                    zerolog.Logger
	me                     module.Local
	engMetrics             module.EngineMetrics
	con                    network.Conduit
	channel                channels.Channel
	headers                storage.Headers
	pendingBlocks          *fifoqueue.FifoQueue // queues for processing inbound blocks
	pendingBlocksNotifier  engine.Notifier
	finalizedBlockTracker  *tracker.NewestBlockTracker
	finalizedBlockNotifier engine.Notifier

	core common.FollowerCore
}

var _ network.MessageProcessor = (*Engine)(nil)
var _ consensus.Compliance = (*Engine)(nil)

func New(
	log zerolog.Logger,
	net network.Network,
	me module.Local,
	engMetrics module.EngineMetrics,
	core common.FollowerCore,
	opts ...EngineOption,
) (*Engine, error) {
	// FIFO queue for block proposals
	pendingBlocks, err := fifoqueue.NewFifoQueue(defaultBlockQueueCapacity)
	if err != nil {
		return nil, fmt.Errorf("failed to create queue for inbound blocks: %w", err)
	}

	e := &Engine{
		log:                   log.With().Str("engine", "follower").Logger(),
		me:                    me,
		engMetrics:            engMetrics,
		channel:               channels.ReceiveBlocks,
		pendingBlocks:         pendingBlocks,
		pendingBlocksNotifier: engine.NewNotifier(),
		core:                  core,
	}

	for _, apply := range opts {
		apply(e)
	}

	con, err := net.Register(e.channel, e)
	if err != nil {
		return nil, fmt.Errorf("could not register engine to network: %w", err)
	}
	e.con = con

	e.ComponentManager = component.NewComponentManagerBuilder().
		AddWorker(e.processMessagesLoop).
		Build()

	return e, nil
}

// OnBlockProposal errors when called since follower engine doesn't support direct ingestion via internal method.
func (e *Engine) OnBlockProposal(_ flow.Slashable[*messages.BlockProposal]) {
	e.log.Error().Msg("received unexpected block proposal via internal method")
}

// OnSyncedBlocks performs processing of incoming blocks by pushing into queue and notifying worker.
func (e *Engine) OnSyncedBlocks(blocks flow.Slashable[[]*messages.BlockProposal]) {
	e.engMetrics.MessageReceived(metrics.EngineFollower, metrics.MessageSyncedBlocks)
	// a blocks batch that is synced has to come locally, from the synchronization engine
	// the block itself will contain the proposer to indicate who created it

	// queue proposal
	if e.pendingBlocks.Push(blocks) {
		e.pendingBlocksNotifier.Notify()
	}
}

// OnFinalizedBlock implements the `OnFinalizedBlock` callback from the `hotstuff.FinalizationConsumer`
// It informs follower.Core about finalization of the respective block.
//
// CAUTION: the input to this callback is treated as trusted; precautions should be taken that messages
// from external nodes cannot be considered as inputs to this function
func (e *Engine) OnFinalizedBlock(block *model.Block) {
	if e.finalizedBlockTracker.Track(block) {
		e.finalizedBlockNotifier.Notify()
	}
}

// Process processes the given event from the node with the given origin ID in
// a blocking manner. It returns the potential processing error when done.
func (e *Engine) Process(channel channels.Channel, originID flow.Identifier, message interface{}) error {
	switch msg := message.(type) {
	case *messages.BlockProposal:
		e.onBlockProposal(flow.Slashable[*messages.BlockProposal]{
			OriginID: originID,
			Message:  msg,
		})
	default:
		e.log.Warn().Msgf("%v delivered unsupported message %T through %v", originID, message, channel)
	}
	return nil
}

// processMessagesLoop processes available block and finalization events as they are queued.
func (e *Engine) processMessagesLoop(ctx irrecoverable.SignalerContext, ready component.ReadyFunc) {
	ready()

	doneSignal := ctx.Done()
	newPendingBlockSignal := e.pendingBlocksNotifier.Channel()
	newFinalizedBlockSignal := e.finalizedBlockNotifier.Channel()
	for {
		select {
		case <-doneSignal:
			return
		case <-newPendingBlockSignal:
			err := e.processQueuedBlocks(doneSignal, newFinalizedBlockSignal) // no errors expected during normal operations
			if err != nil {
				ctx.Throw(err)
			}
		case <-newFinalizedBlockSignal:
			err := e.processFinalizedBlock()
			if err != nil {
				ctx.Throw(err)
			}
		}
	}
}

// processQueuedBlocks processes any available messages until the message queue is empty.
// Only returns when all inbound queues are empty (or the engine is terminated).
// No errors are expected during normal operation. All returned exceptions are potential
// symptoms of internal state corruption and should be fatal.
func (e *Engine) processQueuedBlocks(doneSignal, newFinalizedBlock <-chan struct{}) error {
	for {
		select {
		case <-doneSignal:
			return nil
		case <-newFinalizedBlock:
			// finalization events should get priority.
			err := e.processFinalizedBlock()
			if err != nil {
				return err
			}
		default:
		}

		msg, ok := e.pendingBlocks.Pop()
		if ok {
			batch := msg.(flow.Slashable[[]*messages.BlockProposal])
			// NOTE: this loop might need tweaking, we might want to check channels that were passed as arguments more often.
			for _, block := range batch.Message {
				err := e.core.OnBlockProposal(batch.OriginID, block)
				if err != nil {
					return fmt.Errorf("could not handle block proposal: %w", err)
				}
				e.engMetrics.MessageHandled(metrics.EngineFollower, metrics.MessageBlockProposal)
			}
			continue
		}

		// when there are no more messages in the queue, back to the processMessagesLoop to wait
		// for the next incoming message to arrive.
		return nil
	}
}

// processFinalizedBlock performs processing of finalized block by querying it from storage
// and propagating to follower core.
func (e *Engine) processFinalizedBlock() error {
	blockID := e.finalizedBlockTracker.NewestBlock().BlockID
	// retrieve the latest finalized header, so we know the height
	finalHeader, err := e.headers.ByBlockID(blockID)
	if err != nil { // no expected errors
		return fmt.Errorf("could not query finalized block %x: %w", blockID, err)
	}

	err = e.core.OnFinalizedBlock(finalHeader)
	if err != nil {
		return fmt.Errorf("could not process finalized block %x: %w", blockID, err)
	}
	return nil
}

// onBlockProposal performs processing of incoming block by pushing into queue and notifying worker.
func (e *Engine) onBlockProposal(proposal flow.Slashable[*messages.BlockProposal]) {
	e.engMetrics.MessageReceived(metrics.EngineFollower, metrics.MessageBlockProposal)
	proposalAsList := flow.Slashable[[]*messages.BlockProposal]{
		OriginID: proposal.OriginID,
		Message:  []*messages.BlockProposal{proposal.Message},
	}
	// queue proposal
	if e.pendingBlocks.Push(proposalAsList) {
		e.pendingBlocksNotifier.Notify()
	}
}
