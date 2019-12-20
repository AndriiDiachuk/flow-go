package verifier

import (
	"github.com/pkg/errors"
	"github.com/rs/zerolog"

	"github.com/dapperlabs/flow-go/engine"
	"github.com/dapperlabs/flow-go/model/execution"
	"github.com/dapperlabs/flow-go/model/flow"
	"github.com/dapperlabs/flow-go/model/flow/identity"
	"github.com/dapperlabs/flow-go/model/verification"
	"github.com/dapperlabs/flow-go/module"
	"github.com/dapperlabs/flow-go/network"
	"github.com/dapperlabs/flow-go/protocol"
	"github.com/dapperlabs/flow-go/utils/logging"
)

// Engine implements the verifier engine of the verification node,
// responsible for reception of a execution receipt, verifying that, and
// emitting its corresponding result approval to the entire system.
type Engine struct {
	unit  *engine.Unit    // used to control startup/shutdown
	log   zerolog.Logger  // used to log relevant actions
	con   network.Conduit // used for inter-node communication within the network
	me    module.Local    // used to access local node information
	state protocol.State  // used to access the  protocol state
}

// New creates and returns a new instance of a verifier engine
func New(loger zerolog.Logger, net module.Network, state protocol.State, me module.Local) (*Engine, error) {
	e := &Engine{
		unit:  engine.NewUnit(),
		log:   loger,
		state: state,
		me:    me,
	}

	// register the engine with the network layer and store the conduit
	con, err := net.Register(engine.VerificationVerifier, e)
	if err != nil {
		return nil, errors.Wrap(err, "could not register engine")
	}

	e.con = con

	return e, nil
}

// Ready returns a channel that is closed when the verifier engine is ready.
func (e *Engine) Ready() <-chan struct{} {
	return e.unit.Ready()
}

// Done returns a channel that is closed when the verifier engine is done.
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
		err := e.Process(originID, event)
		if err != nil {
			e.log.Error().Err(err).Msg("could not process submitted event")
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

// process receives and submits an event to the verifier engine for processing.
// It returns an error so the verifier engine will not propagate an event unless
// it is successfully processed by the engine.
// The origin ID indicates the node which originally submitted the event to
// the peer-to-peer network.
func (e *Engine) process(originID flow.Identifier, event interface{}) error {
	switch ev := event.(type) {
	case *execution.ExecutionReceipt:
		return e.onExecutionReceipt(originID, ev)
	default:
		return errors.Errorf("invalid event type (%T)", event)
	}
}

// onExecutionReceipt receives an execution receipt (exrcpt), verifies that and emits
// a result approval upon successful verification
func (e *Engine) onExecutionReceipt(originID flow.Identifier, exrcpt *execution.ExecutionReceipt) error {
	// todo: add id of the ER once gets available
	e.log.Info().
		Hex("origin_id", originID[:]).
		Msg("execution receipt received")

	// todo: correctness check for execution receipts

	// validating identity of the originID
	id, err := e.state.Final().Identity(originID)
	if err != nil {
		// todo: potential attack on authenticity
		return errors.Errorf("invalid origin id %s", originID[:])
	}

	// validating role of the originID
	// an execution receipt should be either coming from an execution node through the
	// Process method, or from the current verifier node itself through the Submit method
	if id.Role != flow.Role(flow.RoleExecution) && id.NodeID != e.me.NodeID() {
		// todo: potential attack on integrity
		return errors.Errorf("invalid role for generating an execution receipt, id: %s, role: %s", id.NodeID, id.Role)
	}

	// extracting list of consensus nodes' ids
	consIds, err := e.state.Final().
		Identities(identity.HasRole(flow.RoleConsensus))
	if err != nil {
		return errors.Wrap(err, "could not get identities")
	}

	// emitting a result approval to all consensus nodes
	resApprov := &verification.ResultApproval{}
	err = e.con.Submit(resApprov, consIds.NodeIDs()...)
	if err != nil {
		return errors.Wrap(err, "could not push result approval")
	}

	// todo: add a hex for hash of the result approval
	e.log.Info().
		Strs("target_ids", logging.HexSlice(consIds.NodeIDs())).
		Msg("result approval propagated")

	return nil
}
