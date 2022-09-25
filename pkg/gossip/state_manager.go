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
		debug         bool
		logger        *zap.Logger
		fsm           *fsm.FSM
		localNodeID   uint16
		localNodeName string
		state         *State
		rwm           sync.RWMutex
	}
)

func newStateManager(debug bool, logger *zap.Logger, localNodeID uint16, localNodeName string, active bool) *StateManager {
	sm := &StateManager{
		debug:         debug,
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

func (s *StateManager) IsLocalNode(key uint16) bool {
	return s.localNodeID == key
}

func (s *StateManager) HasNode(key uint16) bool {
	s.rwm.RLock()
	defer s.rwm.RUnlock()

	_, ok := s.state.Nodes[key]
	return ok
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

func (s *StateManager) LocalNodeState() map[uint16]NodeState {
	s.rwm.RLock()
	defer s.rwm.RUnlock()

	ns := s.state.Nodes[s.localNodeID]
	mns := map[uint16]NodeState{
		s.localNodeID: ns,
	}

	return mns
}

func (s *StateManager) ImportState(state map[uint16]NodeState) {
	s.rwm.Lock()
	defer s.rwm.Unlock()

	for key, node := range state {
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

func (s *StateManager) CurrentState() string {
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

func (s *StateManager) SetState(name StateName) {
	s.fsm.SetState(name)
	s.setLocalState()
}

func (s *StateManager) IsLeader() bool {
	s.rwm.RLock()
	defer s.rwm.RUnlock()

	return s.state.Nodes[s.localNodeID].Leader == s.localNodeID
}

// ElectLeader sets leader for LocalNode & returns true when 100% quorum is achieved
func (s *StateManager) ElectLeader() bool {
	s.rwm.Lock()
	defer s.rwm.Unlock()

	// choose node with the smallest Config.NodeID
	i := 0
	first := uint16(math.MaxUint16)
	for key := range s.state.Nodes {
		if key < first {
			first = key
			i++
		}
	}

	// set the node as the leader, if not already
	if s.state.Nodes[s.localNodeID].Leader != first {
		ns := s.state.Nodes[s.localNodeID]
		ns.Leader = first
		ns.Timestamp = time.Now().UTC()
		s.state.Nodes[s.localNodeID] = ns
	}

	for _, node := range s.state.Nodes {
		if node.Leader != first {
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
