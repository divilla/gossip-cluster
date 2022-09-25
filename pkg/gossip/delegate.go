package gossip

import (
	"encoding/json"
	"fmt"
	"github.com/hashicorp/memberlist"
	"github.com/tidwall/gjson"
	"go.uber.org/zap"
)

type (
	Delegate struct {
		debug  bool
		logger *zap.Logger
		ml     *memberlist.Memberlist
		tlq    *memberlist.TransmitLimitedQueue
		State  *StateManager
	}

	Update struct {
		Action string
		Data   map[string]string
	}
)

func newDelegate(debug bool, logger *zap.Logger, ml *memberlist.Memberlist, tlq *memberlist.TransmitLimitedQueue, state *StateManager) *Delegate {
	return &Delegate{
		debug:  debug,
		logger: logger,
		ml:     ml,
		tlq:    tlq,
		State:  state,
	}
}

// NodeMeta is used to retrieve meta-data about the current node
// when broadcasting an alive message. Its length is limited to
// the given byte size. This metadata is available in the Node structure.
func (d *Delegate) NodeMeta(limit int) []byte {
	d.logger.Info("Delegate.NodeMeta()", zap.Int("limit", limit))
	return []byte{}
}

// NotifyMsg is called when a user-data message is received.
// Care should be taken that this method does not block, since doing
// so would block the entire UDP packet receive loop. Additionally, the byte
// slice may be modified after the call returns, so it should be copied if needed
func (d *Delegate) NotifyMsg(b []byte) {
	d.logger.Info("Delegate.NotifyMsg()", zap.String("b", string(b)))

	if len(b) == 0 {
		return
	}

	fmt.Println("NotifyMsg", string(b))
}

// GetBroadcasts is called when user data messages can be broadcast.
// It can return a list of buffers to send. Each buffer should assume an
// overhead as provided with a limit on the total byte size allowed.
// The total byte size of the resulting data to send must not exceed
// the limit. Care should be taken that this method does not block,
// since doing so would block the entire UDP packet receive loop.
func (d *Delegate) GetBroadcasts(overhead, limit int) [][]byte {
	broadcasts := d.tlq.GetBroadcasts(overhead, limit)

	for _, data := range broadcasts {
		fmt.Println(string(data))
		method := gjson.GetBytes(data, "method").String()
		switch method {
		case "select_leader":
			var slm SelectLeaderMessage
			if err := json.Unmarshal(data, &slm); err != nil {
				panic(err)
			}
		}
	}

	return broadcasts
}

// LocalState is used for a TCP Push/Pull. This is sent to
// the remote side in addition to the membership information. Any
// data can be sent here. See MergeRemoteState as well. The `join`
// boolean indicates this is for a join instead of a push/pull.
func (d *Delegate) LocalState(join bool) []byte {
	if join {
		d.logger.Info("gossip.Delegate.LocalState", zap.String("node", d.State.localNodeName), zap.Bool("join", true))
		return nil
	}

	b, _ := json.Marshal(d.State.GetNodes())
	d.logger.Info("gossip.Delegate.LocalState", zap.String("node", d.State.localNodeName), zap.ByteString("join", b))
	return b
}

// MergeRemoteState is invoked after a TCP Push/Pull. This is the
// state received from the remote side and is the result of the
// remote side's LocalState call. The 'join'
// boolean indicates this is for a join instead of a push/pull.
func (d *Delegate) MergeRemoteState(buf []byte, join bool) {
	d.logger.Info("Delegate.MergeRemoteState()",
		zap.String("name", d.State.localNodeName),
		zap.String("buf", string(buf)),
		zap.Bool("join", join),
	)

	if join || len(buf) == 0 {
		return
	}

	var state map[uint16]NodeState
	if err := json.Unmarshal(buf, &state); err != nil {
		panic(err)
	}
	d.State.SetState(state)
}
