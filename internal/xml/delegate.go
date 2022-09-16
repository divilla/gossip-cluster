package xml

import (
	"encoding/json"
	"fmt"
	"github.com/hashicorp/memberlist"
	"github.com/tidwall/gjson"
	"go.uber.org/zap"
	"sync"
)

type (
	Delegate struct {
		logger     *zap.Logger
		memberlist *memberlist.Memberlist
		broadcasts *memberlist.TransmitLimitedQueue
		items      map[string]string
		rwm        sync.RWMutex
	}

	Update struct {
		Action string
		Data   map[string]string
	}
)

func NewDelegate(logger *zap.Logger, ml *memberlist.Memberlist) *Delegate {
	return &Delegate{
		logger:     logger,
		memberlist: ml,
		items:      make(map[string]string),
	}
}

func (d *Delegate) NodeMeta(limit int) []byte {
	return []byte{}
}

func (d *Delegate) NotifyMsg(b []byte) {
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
	return d.broadcasts.GetBroadcasts(overhead, limit)
}

func (d *Delegate) LocalState(join bool) []byte {
	if join {
		for _, mem := range d.memberlist.Members() {
			d.logger.Info("Member",
				zap.String("Name", mem.Name),
				zap.String("Address", fmt.Sprintf("%s:%d", mem.Addr, mem.Port)))
		}

		return nil
	}

	d.rwm.RLock()
	defer d.rwm.RUnlock()

	b, _ := json.Marshal(d.items)
	return b
}

func (d *Delegate) MergeRemoteState(buf []byte, join bool) {
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
