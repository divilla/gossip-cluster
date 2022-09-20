package memlistconf

import (
	"encoding/json"
	"github.com/hashicorp/memberlist"
	"github.com/tidwall/gjson"
	"go.uber.org/zap"
	"sync"
)

type (
	Delegate struct {
		logger *zap.Logger
		ml     *memberlist.Memberlist
		tlq    *memberlist.TransmitLimitedQueue
		state  *GlobalState
		items  map[string]string
		rwm    sync.RWMutex
	}

	Update struct {
		Action string
		Data   map[string]string
	}
)

func NewDelegate(logger *zap.Logger, ml *memberlist.Memberlist, state *GlobalState) *Delegate {
	return &Delegate{
		logger: logger,
		items:  make(map[string]string),
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

	switch b[0] {
	case 'd':
		var updates []*Update
		if err := json.Unmarshal(b[1:], &updates); err != nil {
			return
		}
		d.rwm.Lock()
		defer d.rwm.Unlock()

		for _, u := range updates {
			for k, v := range u.Data {
				switch u.Action {
				case "add":
					d.items[k] = v
				case "del":
					delete(d.items, k)
				}
			}
		}
	}
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
		d.logger.Info("xml.Delegate.LocalState", zap.Bool("join", true))
		return nil
	}

	d.rwm.RLock()
	defer d.rwm.RUnlock()

	b, _ := json.Marshal(d.items)
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
	if !join {
		return
	}

	d.rwm.Lock()
	defer d.rwm.Unlock()

	gjson.GetBytes(buf, "").ForEach(func(key, value gjson.Result) bool {
		d.items[key.String()] = value.String()
		return true
	})
}
