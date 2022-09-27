package gossip

import (
	"github.com/looplab/fsm"
	"go.uber.org/zap"
	"math"
	"sort"
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

func (s *StateManager) LocalNodeState() NodeState {
	return s.state.Nodes[s.localNodeID]
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

	s.setIndexes()

	return true
}

func (s *StateManager) LocalState() map[uint16]NodeState {
	s.rwm.RLock()
	defer s.rwm.RUnlock()

	ns := s.LocalNodeState()
	mns := map[uint16]NodeState{
		s.localNodeID: ns,
	}

	return mns
}

func (s *StateManager) ImportState(state map[uint16]NodeState) {
	s.rwm.Lock()
	defer s.rwm.Unlock()

	var hasNew bool

	for key, node := range state {
		if key == s.localNodeID {
			continue
		}

		if _, ok := s.state.Nodes[key]; !ok {
			hasNew = true
			s.state.Nodes[key] = node
			continue
		}

		if node.Timestamp.After(s.state.Nodes[key].Timestamp) {
			s.state.Nodes[key] = node
			continue
		}
	}

	if hasNew {
		s.setIndexes()
	}
}

func (s *StateManager) CurrentState() string {
	s.rwm.RLock()
	defer s.rwm.RUnlock()

	return s.LocalNodeState().State
}

func (s *StateManager) Trigger(name EventName, args ...interface{}) error {
	s.rwm.Lock()
	defer s.rwm.Unlock()

	if err := s.fsm.Event(name, args...); err != nil {
		return err
	}
	s.setCurrentState()

	return nil
}

func (s *StateManager) SetState(name StateName) {
	s.rwm.Lock()
	defer s.rwm.Unlock()

	s.fsm.SetState(name)
	s.setCurrentState()
}

func (s *StateManager) IsLeader() bool {
	s.rwm.RLock()
	defer s.rwm.RUnlock()

	return s.LocalNodeState().Leader == s.localNodeID
}

// ElectLeader sets leader for LocalNode & returns true when 100% quorum is achieved
func (s *StateManager) ElectLeader() bool {
	s.rwm.Lock()
	defer s.rwm.Unlock()

	// choose node with the smallest Config.NodeID
	i := 0
	min := uint16(math.MaxUint16)
	for id := range s.state.Nodes {
		if id < min {
			min = id
			i++
		}
	}

	if i == 0 {
		return true
	}

	// set the node as the leader, if not already
	if s.LocalNodeState().Leader != min {
		ns := s.state.Nodes[s.localNodeID]
		ns.Leader = min
		ns.Timestamp = time.Now().UTC()
		s.state.Nodes[s.localNodeID] = ns
	}

	for _, node := range s.state.Nodes {
		if node.Leader != min {
			return false
		}
	}

	return true
}

func (s *StateManager) AssignWorkers() {
	s.rwm.Lock()
	defer s.rwm.Unlock()

	var workers []Worker
	length := len(s.state.Nodes)
	index := s.getLocalNodeIndex()
	for key, worker := range Workers {
		if key%length == index {
			workers = append(workers, worker)
		}
	}

	ns := s.LocalNodeState()
	ns.Workers = workers
	ns.Timestamp = time.Now().UTC()
	s.state.Nodes[s.localNodeID] = ns
}

func (s *StateManager) ReleaseWorkers() {
	s.rwm.Lock()
	defer s.rwm.Unlock()

	ns := s.LocalNodeState()
	ns.Workers = make([]string, 0)
	ns.Timestamp = time.Now().UTC()
	s.state.Nodes[s.localNodeID] = ns
}

func (s *StateManager) StartWorkers() {
	s.rwm.Lock()
	defer s.rwm.Unlock()

	for worker := range s.state.Working {
		s.state.Working[worker] = false
	}

	var isWorking bool
	for _, worker := range s.LocalNodeState().Workers {
		isWorking = true
		s.state.Working[worker] = true
	}

	ns := s.LocalNodeState()
	ns.Working = isWorking
	ns.Timestamp = time.Now().UTC()
	s.state.Nodes[s.localNodeID] = ns
}

func (s *StateManager) StopWorkers() {
	s.rwm.Lock()
	defer s.rwm.Unlock()

	if !s.LocalNodeState().Working {
		return
	}

	for worker := range s.state.Working {
		s.state.Working[worker] = false
	}

	ns := s.LocalNodeState()
	ns.Working = false
	ns.Timestamp = time.Now().UTC()
	s.state.Nodes[s.localNodeID] = ns
}

func (s *StateManager) Size() int {
	s.rwm.RLock()
	defer s.rwm.RUnlock()

	return len(s.state.Nodes)
}

func (s *StateManager) getLocalNodeIndex() int {
	for index, id := range s.state.Indexes {
		if s.localNodeID == id {
			return index
		}
	}

	return -1
}

func (s *StateManager) setIndexes() {
	nodes := s.state.Nodes
	indexes := make([]uint16, len(nodes))

	i := 0
	for id := range nodes {
		indexes[i] = id
		i++
	}

	sort.Slice(indexes,
		func(i, j int) bool {
			return indexes[i] < indexes[j]
		},
	)

	s.state.Indexes = indexes
}

func (s *StateManager) setCurrentState() {
	ns := s.LocalNodeState()
	ns.State = s.fsm.Current()
	ns.Timestamp = time.Now().UTC()
	s.state.Nodes[s.localNodeID] = ns
}
