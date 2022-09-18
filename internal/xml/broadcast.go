package xml

import "github.com/hashicorp/memberlist"

type (
	Broadcast struct {
		msg    []byte
		notify chan<- struct{}
	}
)

func (b *Broadcast) Invalidates(other memberlist.Broadcast) bool {
	return false
}

func (b *Broadcast) Message() []byte {
	return b.msg
}

func (b *Broadcast) Finished() {
	if b.notify != nil {
		close(b.notify)
	}
}
