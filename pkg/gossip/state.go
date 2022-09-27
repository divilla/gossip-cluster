package gossip

import (
	"github.com/looplab/fsm"
	"go.uber.org/zap"
	"time"
)

const (
	Idle        StateName = "idle"
	Configuring StateName = "configuring"
	Joining     StateName = "joining"
	Assembling  StateName = "assembling"
	Electing    StateName = "electing"
	Assigning   StateName = "balancing"
	Working     StateName = "working"
	Starting    StateName = "starting"
	Stopping    StateName = "stopping"

	Join      EventName = "join"
	Joined    EventName = "joined"
	Assemble  EventName = "assemble"
	Assembled EventName = "assembled"
	Elect     EventName = "elect"
	Elected   EventName = "elected"
	Assign    EventName = "assign"
	Assigned  EventName = "assigned"
	Start     EventName = "start"
	Started   EventName = "started"
	Stop      EventName = "stop"
	Stopped   EventName = "stopped"
	Finish    EventName = "finish"

	PlDb Worker = "pl_db"
	UaDb Worker = "ua_db"
	RoDb Worker = "ro_db"
	KzDb Worker = "kz_db"
	PtDb Worker = "pt_db"
	BgDb Worker = "bg_db"
	UzDb Worker = "uz_db"
)

var Workers = []Worker{PlDb, UaDb, RoDb, KzDb, PtDb, BgDb, UzDb}

type (
	State struct {
		Indexes []uint16
		Nodes   map[uint16]NodeState `json:"nodes"`
		Working map[string]bool
	}

	NodeState struct {
		Name      string    `json:"name"`
		State     StateName `json:"state"`
		Leader    uint16    `json:"leader"`
		Workers   []Worker  `json:"workers"`
		Working   bool      `json:"working"`
		Timestamp time.Time `json:"timestamp"`
	}

	StateName = string
	EventName = string
	Worker    = string
)

func newState(localNodeID uint16, localNodeName string, localNodeState StateName) *State {
	working := make(map[string]bool)
	for _, worker := range Workers {
		working[worker] = false
	}

	return &State{
		Indexes: []uint16{localNodeID},
		Nodes: map[uint16]NodeState{
			localNodeID: {
				Name:      localNodeName,
				State:     localNodeState,
				Workers:   make([]string, 0),
				Timestamp: time.Now().UTC(),
			},
		},
		Working: working,
	}
}

func newFSM(sm *StateManager) *fsm.FSM {
	events := fsm.Events{
		{Name: Join, Src: []string{Idle, Configuring, Joining, Assembling, Electing, Assigning, Working, Starting, Stopping}, Dst: Joining},
		{Name: Joined, Src: []string{Joining}, Dst: Configuring},
		{Name: Assemble, Src: []string{Configuring}, Dst: Assembling},
		{Name: Assembled, Src: []string{Assembling}, Dst: Configuring},
		{Name: Elect, Src: []string{Configuring}, Dst: Electing},
		{Name: Elected, Src: []string{Electing}, Dst: Configuring},
		{Name: Assign, Src: []string{Configuring}, Dst: Assigning},
		{Name: Assigned, Src: []string{Assigning}, Dst: Idle},
		{Name: Start, Src: []string{Idle}, Dst: Starting},
		{Name: Started, Src: []string{Starting}, Dst: Working},
		{Name: Stop, Src: []string{Idle, Configuring, Joining, Assembling, Electing, Assigning, Working, Starting, Stopping}, Dst: Stopping},
		{Name: Stopped, Src: []string{Stopping}, Dst: Configuring},
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
