package memlistconf

import (
	"github.com/hashicorp/memberlist"
	"github.com/looplab/fsm"
	"go.uber.org/zap"
)

const (
	State KeyName = "state"

	Started   StateName = "started"
	Joined    StateName = "joined"
	Assembled StateName = "assembled"
	Idle      StateName = "idle"

	Join     EventName = "join"
	Assemble EventName = "assemble"
	Done     EventName = "done"
)

type (
	GlobalState struct {
		logger *zap.Logger
		node   *memberlist.Node
		fsm    *fsm.FSM
		nodes  Node
	}

	Node      = map[string]interface{}
	KeyName   = string
	StateName = string
	EventName = string
)

func NewState(logger *zap.Logger, node *memberlist.Node) *GlobalState {
	return &GlobalState{
		logger: logger,
		node:   node,
		fsm: fsm.NewFSM(
			Started,
			fsm.Events{
				{Name: Join, Src: []string{Started}, Dst: Joined},
				{Name: Assemble, Src: []string{Started, Joined}, Dst: Idle},
				{Name: Done, Src: []string{Assembled}, Dst: Idle},
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
				Done: func(e *fsm.Event) {
					logger.Info("event",
						zap.String("name", e.Event),
						zap.String("src", e.Src),
						zap.String("dst", e.Dst),
					)
				},
			},
		),
		nodes: Node{
			node.Name: Node{
				State: Started,
			},
		},
	}
}

func (s *GlobalState) Name() string {
	return s.node.String()
}

func (s *GlobalState) LocalNode() Node {
	return (s.nodes[s.Name()]).(Node)
}

func (s *GlobalState) LocalState() Node {
	return Node{s.Name(): s.nodes[s.Name()]}
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
