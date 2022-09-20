package memlistconf

import "github.com/hashicorp/memberlist"

func NewTLQ(ml *memberlist.Memberlist) *memberlist.TransmitLimitedQueue {
	return &memberlist.TransmitLimitedQueue{
		NumNodes: func() int {
			return ml.NumMembers()
		},
		RetransmitMult: 3,
	}
}
