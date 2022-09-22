package gossip

import (
	"github.com/looplab/fsm"
	"go.uber.org/zap"
	"sync"
	"time"
)

type (
	StateManager struct {
		logger        *zap.Logger
		fsm           *fsm.FSM
		localNodeID   uint16
		localNodeName string
		state         *State
		rwm           sync.RWMutex
	}
)

func newStateManager(logger *zap.Logger, localNodeID uint16, localNodeName string) *StateManager {
	cs := &StateManager{
		logger:        logger,
		localNodeID:   localNodeID,
		localNodeName: localNodeName,
	}

	return cs.init()
}

func (s *StateManager) LocalNodeID() uint16 {
	return s.localNodeID
}

func (s *StateManager) GetState() State {
	s.rwm.RLock()
	defer s.rwm.RUnlock()

	return *s.state
}

func (s *StateManager) SetState(state State) {
	s.rwm.Lock()
	defer s.rwm.Unlock()

	s.state.Leader = state.Leader
	for key, node := range state.Nodes {
		if key == s.localNodeID {
			continue
		}

		if _, ok := s.state.Nodes[key]; !ok {
			s.state.Nodes[key] = node
			continue
		}

		if node.Timestamp.After(s.state.Nodes[key].Timestamp) {
			s.state.Nodes[key] = node
			continue
		}
	}
}

func (s *StateManager) Trigger(name EventName, args ...interface{}) error {
	if err := s.fsm.Event(name, args...); err != nil {
		return err
	}

	s.setLocalState()
	return nil
}

func (s *StateManager) Size() int {
	return len(s.state.Nodes)
}

func (s *StateManager) setLocalState() {
	s.rwm.Lock()
	defer s.rwm.Unlock()

	ns := s.state.Nodes[s.localNodeID]
	ns.State = s.fsm.Current()
	ns.Timestamp = time.Now().UTC()

	s.state.Nodes[s.localNodeID] = ns
}

func (s *StateManager) init() *StateManager {
	s.fsm = fsm.NewFSM(
		Starting,
		fsm.Events{
			{Name: Join, Src: []string{Starting}, Dst: Joining},
			{Name: Joined, Src: []string{Joining}, Dst: Starting},
			{Name: Assemble, Src: []string{Starting}, Dst: Assembling},
			{Name: Assembled, Src: []string{Assembling}, Dst: Idle},
			{Name: Finish, Src: []string{Assembling}, Dst: Idle},
		},
		fsm.Callbacks{
			Join: func(e *fsm.Event) {
				s.logger.Info("event",
					zap.String("name", e.Event),
					zap.String("src", e.Src),
					zap.String("dst", e.Dst),
				)
			},
			Assemble: func(e *fsm.Event) {
				s.logger.Info("event",
					zap.String("name", e.Event),
					zap.String("src", e.Src),
					zap.String("dst", e.Dst),
				)
			},
			Finish: func(e *fsm.Event) {
				s.logger.Info("event",
					zap.String("name", e.Event),
					zap.String("src", e.Src),
					zap.String("dst", e.Dst),
				)
			},
		},
	)

	s.state = newState(s.localNodeID, s.localNodeName, s.fsm.Current())

	return s
}
