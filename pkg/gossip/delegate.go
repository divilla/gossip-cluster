package gossip

import (
	"encoding/json"
	"fmt"
	"github.com/hashicorp/memberlist"
	"go.uber.org/zap"
)

type (
	Delegate struct {
		logger *zap.Logger
		ml     *memberlist.Memberlist
		tlq    *memberlist.TransmitLimitedQueue
		state  *StateManager
	}

	Update struct {
		Action string
		Data   map[string]string
	}
)

func newDelegate(logger *zap.Logger, ml *memberlist.Memberlist, state *StateManager) *Delegate {
	return &Delegate{
		logger: logger,
		tlq: &memberlist.TransmitLimitedQueue{
			NumNodes: func() int {
				return ml.NumMembers()
			},
			RetransmitMult: 3,
		},
		state: state,
	}
}

func (d *Delegate) NodeMeta(limit int) []byte {
	d.logger.Info("Delegate.NodeMeta()", zap.Int("limit", limit))
	return []byte{}
}

func (d *Delegate) NotifyMsg(b []byte) {
	d.logger.Info("Delegate.NotifyMsg()", zap.String("b", string(b)))

	if len(b) == 0 {
		return
	}

	fmt.Println("NotifyMsg", string(b))
}

func (d *Delegate) GetBroadcasts(overhead, limit int) [][]byte {
	broadcasts := d.tlq.GetBroadcasts(overhead, limit)

	for key, val := range broadcasts {
		d.logger.Info("Delegate.GetBroadcasts()",
			zap.Int("overhead", overhead),
			zap.Int("limit", limit),
			zap.Int("key", key),
			zap.String("b", string(val)))
	}

	return broadcasts
}

func (d *Delegate) LocalState(join bool) []byte {
	if join {
		d.logger.Info("gossip.Delegate.LocalState", zap.Bool("join", true))
		return nil
	}

	b, _ := json.Marshal(d.state.GetState())
	return b
}

func (d *Delegate) MergeRemoteState(buf []byte, join bool) {
	d.logger.Info("Delegate.MergeRemoteState()",
		zap.String("buf", string(buf)),
		zap.Bool("join", join),
	)

	if len(buf) == 0 {
		return
	}
	if join {
		return
	}

	var state State
	if err := json.Unmarshal(buf, &state); err != nil {
		panic(err)
	}

	d.state.SetState(state)
}
