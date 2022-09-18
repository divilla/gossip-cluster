package memlistconf

const (
	Starting NodeState = "starting"
	Ready    NodeState = "ready"
)

type (
	State struct {
		name string
		sns  StateNode
	}

	StateNode = map[string]interface{}
	NodeState = string
)

func NewState(name string, ns NodeState) *State {
	j := make(StateNode)
	j["state"] = ns

	sns := make(StateNode)
	sns[name] = StateNode{"state": Starting}

	return &State{
		name: name,
		sns:  sns,
	}
}

func (s *State) Local() *StateNode {
	if _, ok := s.sns[s.name]; !ok {
		return nil
	}
	if sn, ok := s.sns[s.name].(StateNode); !ok {
		return nil
	} else {
		return &sn
	}
}

func (s *State) Full() StateNode {
	m := make(StateNode)
	for key, val := range s.sns {
		m[key] = val
		break
	}

	return m
}

func (s *State) Size() int {
	return len(s.sns)
}
