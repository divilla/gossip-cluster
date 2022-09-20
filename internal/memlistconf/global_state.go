package memlistconf

import (
	"github.com/hashicorp/memberlist"
	"github.com/looplab/fsm"
	"go.uber.org/zap"
)

const (
	State KeyName = "state"

	Idle       StateName = "idle"
	Joining    StateName = "joining"
	Assembling StateName = "assembling"

	Join      EventName = "join"
	Joined    EventName = "joined"
	Assemble  EventName = "assemble"
	Assembled EventName = "assembled"
	Finish    EventName = "finish"
)

type (
	GlobalState struct {
		logger *zap.Logger
		node   *memberlist.Node
		fsm    *fsm.FSM
		nodes  Nodes
	}

	Nodes     = map[string]NodeData
	NodeData  = map[string]interface{}
	KeyName   = string
	StateName = string
	EventName = string
)

func NewState(logger *zap.Logger, node *memberlist.Node) *GlobalState {
	f := fsm.NewFSM(
		Idle,
		fsm.Events{
			{Name: Join, Src: []string{Idle}, Dst: Joining},
			{Name: Joined, Src: []string{Joining}, Dst: Idle},
			{Name: Assemble, Src: []string{Idle}, Dst: Assembling},
			{Name: Assembled, Src: []string{Assembling}, Dst: Idle},
			{Name: Finish, Src: []string{Assembling}, Dst: Idle},
		},
		fsm.Callbacks{
			Join: func(e *fsm.Event) {
				logger.Info("event",
					zap.String("name", e.Event),
					zap.String("src", e.Src),
					zap.String("dst", e.Dst),
				)
			},
			Assemble: func(e *fsm.Event) {
				logger.Info("event",
					zap.String("name", e.Event),
					zap.String("src", e.Src),
					zap.String("dst", e.Dst),
				)
			},
			Finish: func(e *fsm.Event) {
				logger.Info("event",
					zap.String("name", e.Event),
					zap.String("src", e.Src),
					zap.String("dst", e.Dst),
				)
			},
		},
	)

	return &GlobalState{
		logger: logger,
		node:   node,
		fsm:    f,
		nodes: Nodes{
			node.Name: NodeData{
				State: Idle,
			},
		},
	}
}

func (s *GlobalState) Name() string {
	return s.node.String()
}

func (s *GlobalState) LocalNode() NodeData {
	return s.nodes[s.Name()]
}

func (s *GlobalState) LocalState() Nodes {
	return Nodes{s.Name(): s.nodes[s.Name()]}
}

func (s *GlobalState) Event(name EventName, args ...interface{}) error {
	if err := s.fsm.Event(name, args...); err != nil {
		return err
	}

	s.LocalNode()[State] = s.fsm.Current()
	return nil
}

func (s *GlobalState) Size() int {
	return len(s.nodes)
}
