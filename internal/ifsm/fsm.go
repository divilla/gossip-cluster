package ifsm

import (
	"github.com/looplab/fsm"
)

const (
	Started   StateName = "started"
	Joined    StateName = "joined"
	Assembled StateName = "assembled"
	Idle      StateName = "idle"

	Join     EventName = "join"
	Assemble EventName = "assemble"
	Done     EventName = "done"
)

type (
	StateName = string
	EventName = string
)

func New() *fsm.FSM {
	return fsm.NewFSM(
		Started,
		fsm.Events{
			{Name: Join, Src: []string{Started}, Dst: Joined},
			{Name: Assemble, Src: []string{Started, Joined}, Dst: Idle},
			{Name: Done, Src: []string{Assembled}, Dst: Idle},
		},
		fsm.Callbacks{},
	)
}
