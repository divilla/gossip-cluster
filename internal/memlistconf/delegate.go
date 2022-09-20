package memlistconf

import (
	"encoding/json"
	"github.com/hashicorp/memberlist"
	"go.uber.org/zap"
)

type (
	Delegate struct {
		logger *zap.Logger
		ml     *memberlist.Memberlist
		tlq    *memberlist.TransmitLimitedQueue
		state  *GlobalState
	}

	Message struct {
		Topic string
		Data  map[string]string
	}
)

func NewDelegate(logger *zap.Logger, ml *memberlist.Memberlist, tlq *memberlist.TransmitLimitedQueue, gs *GlobalState) *Delegate {
	return &Delegate{
		logger: logger,
		ml:     ml,
		tlq:    tlq,
		state:  gs,
	}
}

func (d *Delegate) NodeMeta(limit int) []byte {
	d.logger.Info("Delegate.NodeMeta()", zap.Int("limit", limit))
	return []byte{}
}

func (d *Delegate) NotifyMsg(b []byte) {
	d.logger.Info("Delegate.NotifyMsg()", zap.String("b", string(b)))

	//if len(b) == 0 {
	//	return
	//}
	//
	//switch b[0] {
	//case 'd':
	//	var updates []*Message
	//	if err := json.Unmarshal(b[1:], &updates); err != nil {
	//		return
	//	}
	//	d.rwm.Lock()
	//	defer d.rwm.Unlock()
	//
	//	for _, u := range updates {
	//		for k, v := range u.Data {
	//			switch u.Topic {
	//			case "add":
	//				d.items[k] = v
	//			case "del":
	//				delete(d.items, k)
	//			}
	//		}
	//	}
	//}
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

	b, _ := json.Marshal(d.state.LocalState())
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

	//d.rwm.Lock()
	//defer d.rwm.Unlock()
	//
	//gjson.GetBytes(buf, "").ForEach(func(key, value gjson.Result) bool {
	//	d.items[key.String()] = value.String()
	//	return true
	//})
}
