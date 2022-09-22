package gossip

import (
	"encoding/json"
	"fmt"
	"github.com/hashicorp/memberlist"
	"github.com/tidwall/gjson"
	"go.uber.org/zap"
	"time"
)

type (
	Delegate struct {
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

func newDelegate(logger *zap.Logger, ml *memberlist.Memberlist, tlq *memberlist.TransmitLimitedQueue, state *StateManager) *Delegate {
	return &Delegate{
		logger: logger,
		ml:     ml,
		tlq:    tlq,
		State:  state,
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

	for _, data := range broadcasts {
		fmt.Println(string(data))
		method := gjson.GetBytes(data, "method").String()
		switch method {
		case "select_leader":
			var slm SelectLeaderMessage
			if err := json.Unmarshal(data, &slm); err != nil {
				panic(err)
			}
			d.State.state.Leader = slm.Args.Leader
			d.State.state.Timestamp = time.Now().UTC()
		}
	}

	return broadcasts
}

func (d *Delegate) LocalState(join bool) []byte {
	if join {
		d.logger.Info("gossip.Delegate.LocalState", zap.Bool("join", true))
		return nil
	}

	b, _ := json.Marshal(d.State.GetState())
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

	d.State.SetState(state)
}
