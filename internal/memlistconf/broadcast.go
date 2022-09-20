package memlistconf

import (
	"github.com/hashicorp/memberlist"
	"go.uber.org/zap"
)

type (
	Broadcast struct {
		memberlist.NamedBroadcast
		logger   *zap.Logger
		ml       *memberlist.Memberlist
		name     string
		msg      []byte
		notifyCh chan<- struct{}
	}
)

func NewBroadcast(logger *zap.Logger, ml *memberlist.Memberlist, name string, msg []byte, notifyCh chan<- struct{}) *Broadcast {
	return &Broadcast{
		logger:   logger,
		ml:       ml,
		name:     name,
		msg:      msg,
		notifyCh: notifyCh,
	}
}

func (b *Broadcast) Invalidates(other memberlist.Broadcast) bool {
	b.logger.Info("Broadcast.Invalidates()", zap.String("other", string(other.Message())))
	nb, ok := other.(memberlist.NamedBroadcast)
	if !ok {
		return false
	}

	return b.Name() == nb.Name()
}

func (b *Broadcast) Message() []byte {
	b.logger.Info("Broadcast.Message()")
	return b.msg
}

func (b *Broadcast) Finished() {
	b.logger.Info("Broadcast.Finished()")
	if b.notifyCh != nil {
		close(b.notifyCh)
	}
}

func (b *Broadcast) Name() string {
	return b.name
}
