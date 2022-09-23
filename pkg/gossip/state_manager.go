package gossip

import (
	"github.com/looplab/fsm"
	"go.uber.org/zap"
	"math"
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
	sm := &StateManager{
		logger:        logger,
		localNodeID:   localNodeID,
		localNodeName: localNodeName,
	}

	sm.fsm = newFSM(sm)
	sm.state = newState(localNodeID, localNodeName, sm.fsm.Current())

	return sm
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

	//s.ElectLeader()
}

func (s *StateManager) Trigger(name EventName, args ...interface{}) error {
	if err := s.fsm.Event(name, args...); err != nil {
		return err
	}

	s.setLocalState()
	return nil
}

func (s *StateManager) ElectLeader() bool {
	s.rwm.Lock()
	defer s.rwm.Unlock()

	// choose node with the smallest Config.ServerID
	first := uint16(math.MaxUint16)
	for key := range s.state.Nodes {
		if key < first {
			first = key
		}
	}

	// set the node as the leader, if not already
	if s.state.Nodes[s.localNodeID].Leader != first {
		ns := s.state.Nodes[s.localNodeID]
		ns.Leader = first
		s.state.Nodes[s.localNodeID] = ns
	}

	// check for 100% quorum
	for _, ns := range s.state.Nodes {
		if ns.Leader != first {
			return false
		}
	}

	return true
}

func (s *StateManager) Size() int {
	s.rwm.RLock()
	defer s.rwm.RUnlock()

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
