package gossip

import (
	"github.com/looplab/fsm"
	"go.uber.org/zap"
	"time"
)

const (
	Idle       StateName = "idle"
	Starting   StateName = "starting"
	Joining    StateName = "joining"
	Assembling StateName = "assembling"
	Electing   StateName = "electing"

	Join      EventName = "join"
	Joined    EventName = "joined"
	Assemble  EventName = "assemble"
	Assembled EventName = "assembled"
	Elect     EventName = "elect"
	Elected   EventName = "elected"
	Finish    EventName = "finish"
)

type (
	State struct {
		Nodes map[uint16]NodeState `json:"nodes"`
	}

	NodeState struct {
		Leader    uint16    `json:"leader"`
		Name      string    `json:"name"`
		State     StateName `json:"State"`
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

func newFSM(sm *StateManager) *fsm.FSM {
	return fsm.NewFSM(
		Starting,
		fsm.Events{
			{Name: Join, Src: []string{Starting}, Dst: Joining},
			{Name: Joined, Src: []string{Joining}, Dst: Starting},
			{Name: Assemble, Src: []string{Starting}, Dst: Assembling},
			{Name: Assembled, Src: []string{Assembling}, Dst: Idle},
			{Name: Elect, Src: []string{Idle}, Dst: Electing},
			{Name: Elected, Src: []string{Electing}, Dst: Idle},
			{Name: Finish, Src: []string{Assembling}, Dst: Idle},
		},
		fsm.Callbacks{
			Join: func(e *fsm.Event) {
				sm.logger.Info("event",
					zap.String("name", e.Event),
					zap.String("src", e.Src),
					zap.String("dst", e.Dst),
				)
			},
			Assemble: func(e *fsm.Event) {
				sm.logger.Info("event",
					zap.String("name", e.Event),
					zap.String("src", e.Src),
					zap.String("dst", e.Dst),
				)
			},
			Finish: func(e *fsm.Event) {
				sm.logger.Info("event",
					zap.String("name", e.Event),
					zap.String("src", e.Src),
					zap.String("dst", e.Dst),
				)
			},
		},
	)
}
