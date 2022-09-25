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

func (s *StateManager) GetNodes() map[uint16]NodeState {
	s.rwm.RLock()
	defer s.rwm.RUnlock()

	return s.state.Nodes
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

	if i <= 1 {
		return true
	}

	// set the node as the leader, if not already
	if s.state.Nodes[s.localNodeID].Leader != first {
		ns := s.state.Nodes[s.localNodeID]
		ns.Leader = first
		s.state.Nodes[s.localNodeID] = ns
	} else {
		return false
	}

	i = 0
	for key := range s.state.Nodes {
		if key < first {
			first = key
			i++
		}
	}

	return i <= 1
}

func (s *StateManager) CompareNodesDef() bool {
	s.rwm.RLock()
	defer s.rwm.RUnlock()

	for nd := range s.state.nodesDef {
		if _, ok := s.state.Nodes[nd]; !ok {
			return false
		}
	}

	return len(s.state.nodesDef) == len(s.state.Nodes)
}

func (s *StateManager) MakeNodesDef(nds []uint16) {
	s.rwm.Lock()
	defer s.rwm.Unlock()

	s.state.nodesDef = make(map[uint16]struct{})
	for _, nd := range nds {
		s.state.nodesDef[nd] = struct{}{}
	}
}

func (s *StateManager) SetNodesDef(nd uint16) {
	s.rwm.Lock()
	defer s.rwm.Unlock()

	s.state.nodesDef[nd] = struct{}{}
}

func (s *StateManager) UnsetNodesDef(nd uint16) {
	s.rwm.Lock()
	defer s.rwm.Unlock()

	if _, ok := s.state.nodesDef[nd]; ok {
		delete(s.state.nodesDef, nd)
	}
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
