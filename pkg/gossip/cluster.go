package gossip

import (
	"context"
	"encoding/json"
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
		joinCh:  make(chan uint16),
		leaveCh: make(chan uint16),
		stopCh:  stopCh,
	}

	go cluster.init()

	return cluster, nil
}

func (c *Cluster) init() {
	var err error

	mlc := newMemberListConfig(c.Config)
	c.State = newStateManager(c.logger, c.Config.NodeID, mlc.Name, len(c.Config.JoinNodes) == 0)
	tlq := newTlq(c.Memberlist)
	mlc.Events = newEventDelegate(c.Config.Debug, c.logger, c)

	nodeMeta := &NodeMeta{NodeID: c.Config.NodeID}
	if mlc.Delegate, err = newDelegate(c.Config.Debug, c.logger, tlq, nodeMeta, c.State); err != nil {
		panic(err)
	}

	if c.Memberlist, err = memberlist.Create(mlc); err != nil {
		panic(fmt.Errorf("memberlist.Create() error: %w", err))
	}

	c.Messenger = newMessenger(c.logger, c.Memberlist, tlq)

	if len(c.Config.JoinNodes) > 0 {
		ctx, cancel := context.WithTimeout(context.TODO(), time.Duration(c.Config.JoinTimeoutS)*time.Second)
		if err = c.join(ctx); err != nil {
			panic(err)
		}
		cancel()
	} else {
		c.State.SetState(Idle)
	}

	go c.onJoinOrLeave()
}

func (c *Cluster) onJoinOrLeave() {
	var err error
	var id uint16
	var oldCancel context.CancelFunc

	for {
		select {
		case <-c.stopCh:
			return
		case id = <-c.joinCh:
			if oldCancel != nil {
				oldCancel()
			}
			c.State.SetNodesDef(id)
			ctx, cancel := context.WithTimeout(context.TODO(), time.Duration(c.Config.AssembleTimeoutS)*time.Second)
			oldCancel = cancel
			if err = c.assemble(ctx); err != nil {
				panic(err)
			}
			if err = c.electLeader(ctx); err != nil {
				panic(err)
			}
			cancel()
			oldCancel = nil
		case id = <-c.leaveCh:
			if oldCancel != nil {
				oldCancel()
			}
			c.State.UnsetNodesDef(id)
			c.State.RemoveNode(id)
			ctx, cancel := context.WithTimeout(context.TODO(), time.Duration(c.Config.AssembleTimeoutS)*time.Second)
			oldCancel = cancel
			if err = c.assemble(ctx); err != nil {
				panic(err)
			}
			if err = c.electLeader(ctx); err != nil {
				panic(err)
			}
			cancel()
			oldCancel = nil
		}
	}
}

func (c *Cluster) join(ctx context.Context) error {
	var err error
	var nodeMeta NodeMeta
	var nds []uint16

	if err = c.State.Trigger(Join); err != nil {
		return err
	}

	for {
		_, err = c.Memberlist.Join(c.Config.JoinNodes)
		if err == nil {
			for _, node := range c.Memberlist.Members() {
				if err = json.Unmarshal(node.Meta, &nodeMeta); err != nil {
					return fmt.Errorf("gossip.Cluster.join(), json.Unmarshal() error: %w", err)
				}
				nds = append(nds, nodeMeta.NodeID)
			}

			c.State.MakeNodesDef(nds)

			if err = c.State.Trigger(Joined); err != nil {
				return err
			}

			return nil
		}

		select {
		case <-ctx.Done():
			return fmt.Errorf("gossip.Cluster.join(), context canceled: %w", ctx.Err())
		case <-time.After(time.Second):
		}
	}
}

func (c *Cluster) assemble(ctx context.Context) error {
	var err error

	// reach Assembling state
	for {
		if err = c.State.Trigger(Assemble); err == nil {
			break
		}

		select {
		case <-ctx.Done():
			return fmt.Errorf("gossip.Cluster.assemble(), context canceled: %w", ctx.Err())
		case <-time.After(time.Second):
		}
	}

	for {
		if c.State.CompareNodesDef() {
			return c.State.Trigger(Assembled)
		}

		select {
		case <-ctx.Done():
			return fmt.Errorf("gossip.Cluster.assemble(), context canceled: %w", ctx.Err())
		case <-time.After(time.Second):
		}
	}
}

func (c *Cluster) electLeader(ctx context.Context) error {
	var err error

	if err = c.State.Trigger(Elect); err != nil {
		return err
	}

	for {
		if c.State.ElectLeader() {
			if err = c.State.Trigger(Elected); err != nil {
				return err
			}
			return nil
		}

		select {
		case <-ctx.Done():
			return fmt.Errorf("gossip.Cluster.electLeader(), context canceled: %w", ctx.Err())
		case <-time.After(time.Second):
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
