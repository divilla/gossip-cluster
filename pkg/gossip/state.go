package gossip

import (
	"github.com/looplab/fsm"
	"go.uber.org/zap"
	"time"
)

const (
	Idle        StateName = "idle"
	Configuring StateName = "configuring"
	Starting    StateName = "starting"
	Joining     StateName = "joining"
	Assembling  StateName = "assembling"
	Electing    StateName = "electing"
	Assigning   StateName = "balancing"
	Working     StateName = "working"
	Stopping    StateName = "stopping"

	Join      EventName = "join"
	Joined    EventName = "joined"
	Assemble  EventName = "assemble"
	Assembled EventName = "assembled"
	Elect     EventName = "elect"
	Elected   EventName = "elected"
	Assign    EventName = "balance"
	Assigned  EventName = "balanced"
	Start     EventName = "start"
	Started   EventName = "started"
	Stop      EventName = "stop"
	Stopped   EventName = "stopped"
	Finish    EventName = "finish"
)

type (
	State struct {
		Nodes    map[uint16]NodeState `json:"nodes"`
		nodesDef map[uint16]struct{}
	}

	NodeState struct {
		Name      string    `json:"name"`
		State     StateName `json:"state"`
		Leader    uint16    `json:"leader"`
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
		nodesDef: map[uint16]struct{}{
			localNodeID: {},
		},
	}
}

func newFSM(sm *StateManager) *fsm.FSM {
	events := fsm.Events{
		{Name: Join, Src: []string{Starting}, Dst: Joining},
		{Name: Joined, Src: []string{Joining}, Dst: Idle},
		{Name: Assemble, Src: []string{Idle, Configuring}, Dst: Assembling},
		{Name: Assembled, Src: []string{Assembling}, Dst: Configuring},
		{Name: Elect, Src: []string{Configuring}, Dst: Electing},
		{Name: Elected, Src: []string{Electing}, Dst: Idle},
		{Name: Assign, Src: []string{Configuring}, Dst: Assigning},
		{Name: Assigned, Src: []string{Assigning}, Dst: Idle},
		{Name: Start, Src: []string{Idle}, Dst: Starting},
		{Name: Started, Src: []string{Starting}, Dst: Working},
		{Name: Stop, Src: []string{Working}, Dst: Stopping},
		{Name: Stopped, Src: []string{Stopping}, Dst: Idle},
		{Name: Finish, Src: []string{Assembling}, Dst: Idle},
	}

	callbacks := make(fsm.Callbacks, len(events))
	for _, event := range events {
		callbacks[event.Name] = func(e *fsm.Event) {
			if sm.debug {
				sm.logger.Info("event",
					zap.String("node", sm.localNodeName),
					zap.String("name", e.Event),
					zap.String("src", e.Src),
					zap.String("dst", e.Dst))
			}
		}
	}

	return fsm.NewFSM(Starting, events, callbacks)
}
