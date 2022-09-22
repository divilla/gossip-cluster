package gossip

import (
	"time"
)

const (
	Idle       StateName = "idle"
	Starting   StateName = "starting"
	Joining    StateName = "joining"
	Assembling StateName = "assembling"

	Join      EventName = "join"
	Joined    EventName = "joined"
	Assemble  EventName = "assemble"
	Assembled EventName = "assembled"
	Finish    EventName = "finish"
)

type (
	State struct {
		Leader uint16               `json:"leader"`
		Nodes  map[uint16]NodeState `json:"nodes"`
	}

	NodeState struct {
		Name      string    `json:"name"`
		State     StateName `json:"state"`
		Timestamp time.Time `json:"timestamp"`
	}

	StateName = string
	EventName = string
)

func newState(localNodeID uint16, localNodeName string, localNodeState StateName) *State {
	return &State{
		Nodes: map[uint16]NodeState{
			localNodeID: {
				Name:      localNodeName,
				State:     localNodeState,
				Timestamp: time.Now().UTC(),
			},
		},
	}
}
