package gossip

import (
	"github.com/looplab/fsm"
	"go.uber.org/zap"
	"sync"
	"time"
)

const (
	State     KeyName = "state"
	TimeStamp KeyName = "timestamp"

	Idle       StateName = "idle"
	Starting   StateName = "starting"
	Joining    StateName = "joining"
	Assembling StateName = "assembling"

	Join      EventName = "join"
	Joined    EventName = "joined"
	Assemble  EventName = "assemble"
	Assembled EventName = "assembled"
	Finish    EventName = "finish"
)

type (
	ClusterState struct {
		logger        *zap.Logger
		fsm           *fsm.FSM
		localNodeName string
		globalNodes   map[string]NodeState
		rwm           sync.RWMutex
	}

	NodeState struct {
		State     StateName `json:"state"`
		Leader    bool      `json:"leader"`
		Timestamp time.Time `json:"timestamp"`
	}

	KeyName   = string
	StateName = string
	EventName = string
)

func newClusterState(logger *zap.Logger, localNode string) *ClusterState {
	cs := &ClusterState{
		logger:        logger,
		localNodeName: localNode,
		globalNodes: map[string]NodeState{
			localNode: {
				State: Starting,
			},
		},
	}

	return cs.init()
}

func (s *ClusterState) LocalNodeName() string {
	return s.localNodeName
}

func (s *ClusterState) GetState() map[string]NodeState {
	s.rwm.RLock()
	defer s.rwm.RUnlock()

	return s.globalNodes
}

func (s *ClusterState) SetState(state map[string]NodeState) {
	s.rwm.Lock()
	defer s.rwm.Unlock()

	for key, node := range state {
		if key == s.localNodeName {
			continue
		}
		if _, ok := s.globalNodes[key]; !ok {
			s.globalNodes[key] = node
			continue
		}
		if node.Timestamp.After(s.globalNodes[key].Timestamp) {
			s.globalNodes[key] = node
			continue
		}
	}
}

func (s *ClusterState) Trigger(name EventName, args ...interface{}) error {
	if err := s.fsm.Event(name, args...); err != nil {
		return err
	}

	s.setLocalState()
	return nil
}

func (s *ClusterState) Size() int {
	return len(s.globalNodes)
}

func (s *ClusterState) localNode() NodeState {
	s.rwm.RLock()
	defer s.rwm.Unlock()

	return s.globalNodes[s.localNodeName]
}

func (s *ClusterState) setLocalState() {
	s.rwm.Lock()
	defer s.rwm.Unlock()

	ns := NodeState{
		State:     s.fsm.Current(),
		Timestamp: time.Now().UTC(),
	}

	s.globalNodes[s.localNodeName] = ns
}

func (s *ClusterState) init() *ClusterState {
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

	return s
}
