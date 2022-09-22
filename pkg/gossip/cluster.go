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
		State      *ClusterState
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
	c.State = newClusterState(c.logger, c.Memberlist.LocalNode().Name)
	mlc.Delegate = newDelegate(c.logger, c.Memberlist, c.State)

	if len(c.Config.JoinNodes) > 0 {
		if err = c.State.Trigger(Join); err != nil {
			panic(err)
		}
		if err = c.join(); err != nil {
			panic(err)
		}
		if err = c.State.Trigger(Joined); err != nil {
			panic(err)
		}
	}

	if err = c.State.Trigger(Assemble); err != nil {
		panic(err)
	}
	if err = c.assemble(); err != nil {
		panic(err)
	}
	if err = c.State.Trigger(Assembled); err != nil {
		panic(err)
	}
}

func (c *Cluster) join() error {
	var err error

	i := 0
	for {
		if i == c.Config.JoinTimeoutS {
			return fmt.Errorf("gossip.Cluster.join() timeout: %d s", c.Config.JoinTimeoutS)
		}

		_, err = c.Memberlist.Join(c.Config.JoinNodes)
		if err == nil {
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

	i := 0
	for {
		if i == c.Config.AssembleTimeoutS {
			return fmt.Errorf("gossip.Cluster.assemble() timeout: %d s", c.Config.AssembleTimeoutS)
		}

		if c.Memberlist.NumMembers() == c.Config.JoinNodesNum {
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

	if c.Name != "" {
		mlc.Name = c.Name
	}

	if c.BindAddr != "" {
		mlc.BindAddr = c.BindAddr
	}

	if c.BindPort > 0 {
		mlc.BindPort = c.BindPort
	}

	if c.AdvertiseAddr != "" {
		mlc.AdvertiseAddr = c.AdvertiseAddr
	}

	if c.AdvertisePort > 0 {
		mlc.AdvertisePort = c.AdvertisePort
	} else {
		mlc.AdvertisePort = mlc.BindPort
	}

	if c.PushPullIntervalMS >= 0 {
		mlc.PushPullInterval = time.Duration(c.PushPullIntervalMS) * time.Millisecond
	} else {
		mlc.PushPullInterval = 1 * time.Second
	}

	return mlc
}
