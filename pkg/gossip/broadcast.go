package gossip

import (
	"github.com/hashicorp/memberlist"
	"go.uber.org/zap"
)

type (
	Broadcast struct {
		memberlist.NamedBroadcast
		logger *zap.Logger
		ml     *memberlist.Memberlist
		name   string
		msg    []byte
		notify chan<- struct{}
	}
)

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
	if b.notify != nil {
		close(b.notify)
	}
}

func (b *Broadcast) Name() string {
	return b.name
}
