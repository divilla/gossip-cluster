package gossip

import (
	"encoding/json"
	"github.com/hashicorp/memberlist"
	"go.uber.org/zap"
)

type (
	Messenger struct {
		logger *zap.Logger
		ml     *memberlist.Memberlist
		tlq    *memberlist.TransmitLimitedQueue
	}
)

func newMessenger(logger *zap.Logger, ml *memberlist.Memberlist, tlq *memberlist.TransmitLimitedQueue) *Messenger {
	return &Messenger{
		logger: logger,
		ml:     ml,
		tlq:    tlq,
	}
}

func (m *Messenger) Broadcast(topic string, data []byte) {
	bc := NewBroadcast(m.logger, m.ml, topic, data, nil)
	m.tlq.QueueBroadcast(bc)
}

func (m *Messenger) SelectLeader(leaderID uint16) error {
	var data []byte
	var err error
	topic := "select_leader"

	mes := SelectLeaderMessage{
		Method: topic,
	}
	mes.Args.Leader = leaderID

	if data, err = json.Marshal(mes); err != nil {
		return err
	}

	m.Broadcast(topic, data)

	return nil
}
