package p2p

import (
	"github.com/hashicorp/go-multierror"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/core/peer"

	"github.com/onflow/flow-go/module/component"
	p2pmsg "github.com/onflow/flow-go/network/p2p/message"
)

// InvCtrlMsgErrSeverity the severity of the invalid control message error. The gossip score penalty for an invalid control message error
// is amplified by the severity of the error.
type InvCtrlMsgErrSeverity float64

const (
	LowErrSeverity      InvCtrlMsgErrSeverity = .10
	ModerateErrSeverity InvCtrlMsgErrSeverity = .15
	HighErrSeverity     InvCtrlMsgErrSeverity = .20
	CriticalErrSeverity InvCtrlMsgErrSeverity = .25
)

// InvCtrlMsgErrs list of InvCtrlMsgErr's
type InvCtrlMsgErrs []*InvCtrlMsgErr

func (i InvCtrlMsgErrs) Error() error {
	var errs *multierror.Error
	for _, err := range i {
		errs = multierror.Append(errs, err.Err)
	}
	return errs.ErrorOrNil()
}

func (i InvCtrlMsgErrs) Len() int {
	return len(i)
}

// InvCtrlMsgErr struct that wraps an error that occurred with during control message inspection and holds some metadata about the err such as the errors InvCtrlMsgErrSeverity.
type InvCtrlMsgErr struct {
	Err      error
	severity InvCtrlMsgErrSeverity
}

func (i InvCtrlMsgErr) Severity() InvCtrlMsgErrSeverity {
	return i.severity
}

// NewInvCtrlMsgErr returns a new InvCtrlMsgErr.
// Args:
// - err: the error.
// - severity: the error severity.
// Returns:
// - *InvCtrlMsgErr: the invalid control message error.
func NewInvCtrlMsgErr(err error, severity InvCtrlMsgErrSeverity) *InvCtrlMsgErr {
	return &InvCtrlMsgErr{
		Err:      err,
		severity: severity,
	}
}

// InvCtrlMsgNotif is the notification sent to the consumer when an invalid control message is received.
// It models the information that is available to the consumer about a misbehaving peer.
type InvCtrlMsgNotif struct {
	// PeerID is the ID of the peer that sent the invalid control message.
	PeerID peer.ID
	// Errors the errors that occurred during validation.
	Errors InvCtrlMsgErrs
	// MsgType the control message type.
	MsgType p2pmsg.ControlMessageType
}

// NewInvalidControlMessageNotification returns a new *InvCtrlMsgNotif.
// Args:
// - peerID: peer ID of the sender.
// - ctlMsgType: the control message type.
// - errs: validation errors that occurred.
// Returns:
// - *InvCtrlMsgNotif: the invalid control message notification.
func NewInvalidControlMessageNotification(peerID peer.ID, ctlMsgType p2pmsg.ControlMessageType, errs InvCtrlMsgErrs) *InvCtrlMsgNotif {
	return &InvCtrlMsgNotif{
		PeerID:  peerID,
		Errors:  errs,
		MsgType: ctlMsgType,
	}
}

// GossipSubInspectorNotifDistributor is the interface for the distributor that distributes gossip sub inspector notifications.
// It is used to distribute notifications to the consumers in an asynchronous manner and non-blocking manner.
// The implementation should guarantee that all registered consumers are called upon distribution of a new event.
type GossipSubInspectorNotifDistributor interface {
	component.Component
	// Distribute distributes the event to all the consumers.
	// Any error returned by the distributor is non-recoverable and will cause the node to crash.
	// Implementation must be concurrency safe, and non-blocking.
	Distribute(notification *InvCtrlMsgNotif) error

	// AddConsumer adds a consumer to the distributor. The consumer will be called the distributor distributes a new event.
	// AddConsumer must be concurrency safe. Once a consumer is added, it must be called for all future events.
	// There is no guarantee that the consumer will be called for events that were already received by the distributor.
	AddConsumer(GossipSubInvCtrlMsgNotifConsumer)
}

// GossipSubInvCtrlMsgNotifConsumer is the interface for the consumer that consumes gossipsub inspector notifications.
// It is used to consume notifications in an asynchronous manner.
// The implementation must be concurrency safe, but can be blocking. This is due to the fact that the consumer is called
// asynchronously by the distributor.
type GossipSubInvCtrlMsgNotifConsumer interface {
	// OnInvalidControlMessageNotification is called when a new invalid control message notification is distributed.
	// Any error on consuming event must handle internally.
	// The implementation must be concurrency safe, but can be blocking.
	OnInvalidControlMessageNotification(*InvCtrlMsgNotif)
}

// GossipSubInspectorSuite is the interface for the GossipSub inspector suite.
// It encapsulates the rpc inspectors and the notification distributors.
type GossipSubInspectorSuite interface {
	component.Component
	CollectionClusterChangesConsumer
	// InspectFunc returns the inspect function that is used to inspect the gossipsub rpc messages.
	// This function follows a dependency injection pattern, where the inspect function is injected into the gossipsu, and
	// is called whenever a gossipsub rpc message is received.
	InspectFunc() func(peer.ID, *pubsub.RPC) error

	// AddInvalidControlMessageConsumer adds a consumer to the invalid control message notification distributor.
	// This consumer is notified when a misbehaving peer regarding gossipsub control messages is detected. This follows a pub/sub
	// pattern where the consumer is notified when a new notification is published.
	// A consumer is only notified once for each notification, and only receives notifications that were published after it was added.
	AddInvalidControlMessageConsumer(GossipSubInvCtrlMsgNotifConsumer)

	// SetTopicOracle sets the topic oracle of the gossipsub inspector suite.
	// The topic oracle is used to determine the list of topics that the node is subscribed to.
	// If an oracle is not set, the node will not be able to determine the list of topics that the node is subscribed to.
	// This func is expected to be called once and will return an error on all subsequent calls.
	// All errors returned from this func are considered irrecoverable.
	SetTopicOracle(topicOracle func() []string) error
}
