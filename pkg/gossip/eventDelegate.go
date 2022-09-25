package gossip

import (
	"encoding/json"
	"github.com/hashicorp/memberlist"
	"go.uber.org/zap"
)

type (
	// EventDelegate is a simpler delegate that is used only to receive
	// notifications about members joining and leaving. The methods in this
	// delegate may be called by multiple goroutines, but never concurrently.
	// This allows you to reason about ordering.
	EventDelegate struct {
		debug   bool
		logger  *zap.Logger
		joinCh  chan uint16
		leaveCh chan uint16
	}
)

func NewEventDelegate(debug bool, logger *zap.Logger, joinCh chan uint16, leaveCh chan uint16) *EventDelegate {
	return &EventDelegate{
		debug:   debug,
		logger:  logger,
		joinCh:  joinCh,
		leaveCh: leaveCh,
	}
}

// NotifyJoin is invoked when a node is detected to have joined.
// The Node argument must not be modified.
// - stop workers
// - assemble
// - elect leader
// - distribute load
// - start workers
func (d *EventDelegate) NotifyJoin(node *memberlist.Node) {
	if d.debug {
		d.logger.Debug("event_delegate", zap.String("node_name", node.Name))
	}

	var nodeMeta NodeMeta
	if err := json.Unmarshal(node.Meta, &nodeMeta); err != nil {
		panic(err)
	}

	d.joinCh <- nodeMeta.NodeID
}

// NotifyLeave is invoked when a node is detected to have left.
// The Node argument must not be modified.
func (d *EventDelegate) NotifyLeave(node *memberlist.Node) {
	if d.debug {
		d.logger.Debug("event_delegate", zap.String("node_name", node.Name))
	}

	var nodeMeta NodeMeta
	if err := json.Unmarshal(node.Meta, &nodeMeta); err != nil {
		panic(err)
	}

	d.leaveCh <- nodeMeta.NodeID
}

// NotifyUpdate is invoked when a node is detected to have
// updated, usually involving the metadata. The Node argument
// must not be modified.
func (d *EventDelegate) NotifyUpdate(node *memberlist.Node) {
	return
}
