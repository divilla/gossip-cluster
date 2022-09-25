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

func newStateManager(logger *zap.Logger, localNodeID uint16, localNodeName string, active bool) *StateManager {
	sm := &StateManager{
		logger:        logger,
		localNodeID:   localNodeID,
		localNodeName: localNodeName,
	}

	sm.fsm = newFSM(sm)
	sm.state = newState(localNodeID, localNodeName, sm.fsm.Current(), active)

	return sm
}

func (s *StateManager) LocalNodeID() uint16 {
	return s.localNodeID
}

func (s *StateManager) IsLocalNode(key uint16) bool {
	return s.localNodeID == key
}

func (s *StateManager) HasActiveNode(key uint16) bool {
	s.rwm.RLock()
	defer s.rwm.RUnlock()

	node, ok := s.state.Nodes[key]
	return node.Active && ok
}

func (s *StateManager) RemoveNode(id uint16) bool {
	s.rwm.Lock()
	defer s.rwm.Unlock()

	if _, ok := s.state.Nodes[id]; !ok {
		return false
	}

	delete(s.state.Nodes, id)

	return true
}

func (s *StateManager) GetNodes() map[uint16]NodeState {
	s.rwm.RLock()
	defer s.rwm.RUnlock()

	return s.state.Nodes
}

func (s *StateManager) SetState(state map[uint16]NodeState) {
	s.rwm.Lock()
	defer s.rwm.Unlock()

	for key, node := range state {
		if key == s.localNodeID && !node.Active {
			node.Active = true
			node.Timestamp = time.Now().UTC()
			s.state.Nodes[key] = node
			continue
		}

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

func (s *StateManager) Current() string {
	s.rwm.RLock()
	defer s.rwm.RUnlock()

	return s.fsm.Current()
}

func (s *StateManager) Trigger(name EventName, args ...interface{}) error {
	if err := s.fsm.Event(name, args...); err != nil {
		return err
	}

	s.setLocalState()
	return nil
}

func (s *StateManager) IsLeader() bool {
	s.rwm.RLock()
	defer s.rwm.RUnlock()

	return s.state.Nodes[s.localNodeID].Leader == s.localNodeID
}

func (s *StateManager) ElectLeader() bool {
	s.rwm.Lock()
	defer s.rwm.Unlock()

	// choose node with the smallest Config.NodeID
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
