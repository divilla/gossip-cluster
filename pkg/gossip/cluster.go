package gossip

import (
	"fmt"
	"github.com/hashicorp/memberlist"
	"go.uber.org/zap"
	"time"
)

type (
	Cluster struct {
		Config     *Config
		Memberlist *memberlist.Memberlist
		State      *StateManager
		Messenger  *Messenger
		logger     *zap.Logger
		joinCh     chan uint16
		leaveCh    chan uint16
		stopCh     <-chan struct{}
	}
)

func NewCluster(logger *zap.Logger, cfg *Config, stopCh <-chan struct{}) (*Cluster, error) {
	cluster := &Cluster{
		Config:  parseDefaults(cfg),
		logger:  logger,
		joinCh:  make(chan uint16, 10),
		leaveCh: make(chan uint16, 10),
		stopCh:  stopCh,
	}

	go cluster.init()

	return cluster, nil
}

func (c *Cluster) init() {
	var err error

	mlc := newMemberListConfig(c.Config)
	if c.Memberlist, err = memberlist.Create(mlc); err != nil {
		panic(fmt.Errorf("memberlist.Create() error: %w", err))
	}

	nm, err := newNodeMetaBytes(c.Config.NodeID)
	if err != nil {
		panic(err)
	}
	c.Memberlist.LocalNode().Meta = nm

	c.State = newStateManager(c.logger, c.Config.NodeID, c.Memberlist.LocalNode().Name, len(c.Config.JoinNodes) == 0)
	tlq := newTlq(c.Memberlist)
	mlc.Delegate = newDelegate(c.Config.Debug, c.logger, c.Memberlist, tlq, c.State)
	mlc.Events = NewEventDelegate(c.Config.Debug, c.logger, c.joinCh, c.leaveCh)
	c.Messenger = newMessenger(c.logger, c.Memberlist, tlq)

	if len(c.Config.JoinNodes) > 0 {
		if err = c.join(); err != nil {
			panic(err)
		}
	}

	go c.onJoinOrLeave()
}

func (c *Cluster) onJoinOrLeave() {
	var err error
	var id uint16

	for {
		select {
		case <-c.stopCh:
			return
		case id = <-c.joinCh:
			if err = c.assemble(id, true); err != nil {
				panic(err)
			}
			if err = c.electLeader(); err != nil {
				panic(err)
			}
		case id = <-c.leaveCh:
			if err = c.assemble(id, false); err != nil {
				panic(err)
			}
			if err = c.electLeader(); err != nil {
				panic(err)
			}
		}
	}
}

func (c *Cluster) join() error {
	var err error

	if err = c.State.Trigger(Join); err != nil {
		return err
	}

	i := 0
	for {
		if i == c.Config.JoinTimeoutS {
			return fmt.Errorf("gossip.Cluster.join() timeout: %d s", c.Config.JoinTimeoutS)
		}

		_, err = c.Memberlist.Join(c.Config.JoinNodes)
		if err == nil {
			if err = c.State.Trigger(Joined); err != nil {
				return err
			}
			return nil
		}

		select {
		case <-c.stopCh:
			return err
		case <-time.After(time.Second):
			i++
		}
	}
}

func (c *Cluster) assemble(id uint16, join bool) error {
	var err error
	i := 0

	// reach Assembling state
	for {
		if i == c.Config.AssembleTimeoutS {
			return fmt.Errorf("gossip.Cluster.assemble() timeout: %d s", c.Config.AssembleTimeoutS)
		}

		if err = c.State.Trigger(Assemble); err == nil {
			break
		}

		select {
		case <-c.stopCh:
			return err
		case <-time.After(time.Second):
			i++
		}
	}

	for {
		if i == c.Config.AssembleTimeoutS {
			return fmt.Errorf("gossip.Cluster.assemble() timeout: %d s", c.Config.AssembleTimeoutS)
		}

		if !join {
			if !c.State.RemoveNode(id) {
				return fmt.Errorf("assemble failed to remove node: %d", id)
			}
			return nil
		}

		if join && c.State.HasActiveNode(id) {
			return c.State.Trigger(Assembled)
		}

		select {
		case <-c.stopCh:
			return err
		case <-time.After(time.Second):
			i++
		}
	}
}

func (c *Cluster) electLeader() error {
	var err error

	if err = c.State.Trigger(Elect); err != nil {
		return err
	}

	i := 0
	for {
		if i == c.Config.ElectLeaderS {
			return fmt.Errorf("gossip.Cluster.ElectLeader() timeout: %d s", c.Config.ElectLeaderS)
		}

		if c.State.ElectLeader() {
			if err = c.State.Trigger(Elected); err != nil {
				return err
			}

			return nil
		}

		select {
		case <-c.stopCh:
			return err
		case <-time.After(time.Second):
			i++
		}
	}
}

func newMemberListConfig(c *Config) *memberlist.Config {
	mlc := memberlist.DefaultLANConfig()
	mlc.Logger = nil
	mlc.Name = fmt.Sprintf("%06d-%s", c.NodeID, mlc.Name)

	if c.BindAddr != "" {
		mlc.BindAddr = c.BindAddr
		//mlc.AdvertiseAddr = c.AdvertiseAddr
	}

	if c.BindPort > 0 {
		mlc.BindPort = c.BindPort
		mlc.AdvertisePort = c.BindPort
	}

	if c.AdvertiseAddr != "" {
		mlc.AdvertiseAddr = c.AdvertiseAddr
	}

	if c.AdvertisePort > 0 {
		mlc.AdvertisePort = c.AdvertisePort
	}

	if c.PushPullIntervalMS >= 0 {
		mlc.PushPullInterval = time.Duration(c.PushPullIntervalMS) * time.Millisecond
	} else {
		mlc.PushPullInterval = 1 * time.Second
	}

	return mlc
}

func newTlq(ml *memberlist.Memberlist) *memberlist.TransmitLimitedQueue {
	return &memberlist.TransmitLimitedQueue{
		NumNodes: func() int {
			return ml.NumMembers()
		},
		RetransmitMult: 3,
	}
}
