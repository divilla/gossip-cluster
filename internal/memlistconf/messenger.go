package memlistconf

import (
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

func NewMessenger(logger *zap.Logger, ml *memberlist.Memberlist, tlq *memberlist.TransmitLimitedQueue) *Messenger {
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
