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
		stopCh     <-chan struct{}
	}
)

func NewCluster(logger *zap.Logger, cfg *Config, stopCh <-chan struct{}) (*Cluster, error) {
	cluster := &Cluster{
		Config: parseDefaults(cfg),
		logger: logger,
		stopCh: stopCh,
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
	c.State = newStateManager(c.logger, c.Config.ServerID, c.Memberlist.LocalNode().Name)
	tlq := newTlq(c.Memberlist)
	mlc.Delegate = newDelegate(c.logger, c.Memberlist, tlq, c.State)
	c.Messenger = newMessenger(c.logger, c.Memberlist, tlq)

	if len(c.Config.JoinNodes) > 0 {
		if err = c.join(); err != nil {
			panic(err)
		}
	}

	if err = c.assemble(); err != nil {
		panic(err)
	}

	if err = c.electLeader(); err != nil {
		panic(err)
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

func (c *Cluster) assemble() error {
	var err error

	if err = c.State.Trigger(Assemble); err != nil {
		return err
	}

	i := 0
	for {
		if i == c.Config.AssembleTimeoutS {
			return fmt.Errorf("gossip.Cluster.assemble() timeout: %d s", c.Config.AssembleTimeoutS)
		}

		if c.Memberlist.NumMembers() >= c.Config.JoinNodesNum && c.State.Size() >= c.Config.JoinNodesNum {
			if err = c.State.Trigger(Assembled); err != nil {
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
	mlc.Name = fmt.Sprintf("%06d-%s", c.ServerID, mlc.Name)

	if c.BindAddr != "" {
		mlc.BindAddr = c.BindAddr
		//mlc.AdvertiseAddr = c.AdvertiseAddr
	}

	if c.BindPort > 0 {
		mlc.BindPort = c.BindPort
		mlc.AdvertisePort = c.BindPort
	}

	//if c.AdvertiseAddr != "" {
	//	mlc.AdvertiseAddr = c.AdvertiseAddr
	//}
	//
	//if c.AdvertisePort > 0 {
	//	mlc.AdvertisePort = c.AdvertisePort
	//}

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
