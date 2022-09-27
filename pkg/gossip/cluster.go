package gossip

import (
	"context"
	"errors"
	"fmt"
	"github.com/hashicorp/memberlist"
	"go.uber.org/zap"
	"runtime"
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

	NodeMeta struct {
		NodeID uint16 `json:"node_id"`
	}

	FinishFunc func()
)

func NewCluster(logger *zap.Logger, cfg *Config, stopCh <-chan struct{}) (*Cluster, error) {
	cluster := &Cluster{
		Config:  parseDefaults(cfg),
		logger:  logger,
		joinCh:  make(chan uint16, 1),
		leaveCh: make(chan uint16, 1),
		stopCh:  stopCh,
	}

	go cluster.init()

	return cluster, nil
}

func (c *Cluster) init() {
	var err error

	mlc := newMemberListConfig(c.Config)
	c.State = newStateManager(c.Config.Debug, c.logger, c.Config.NodeID, mlc.Name, len(c.Config.JoinNodes) == 0)

	tlq := newTlq(c.Memberlist)
	nodeMeta := &NodeMeta{NodeID: c.Config.NodeID}
	if mlc.Delegate, err = newDelegate(c.Config.Debug, c.logger, tlq, nodeMeta, c.State); err != nil {
		panic(err)
	}
	mlc.Events = newEventDelegate(c.Config.Debug, c.logger, mlc.Name, c.joinCh, c.leaveCh)

	if c.Memberlist, err = memberlist.Create(mlc); err != nil {
		panic(fmt.Errorf("memberlist.Create() error: %w", err))
	}

	go c.onJoinOrLeave()

	c.Messenger = newMessenger(c.logger, c.Memberlist, tlq)

	if !c.Config.First {
		if err = c.join(); err != nil {
			c.logger.Error("gossip.Cluster.join()", zap.Error(err))
		}
	} else {
		//c.joinCh <- c.Config.NodeID
		c.State.SetState(Idle)
	}
}

func (c *Cluster) onJoinOrLeave() {
	runtime.Gosched()

	var id uint16
	var oldCancel context.CancelFunc
	var err error

	for {
		select {
		case <-c.stopCh:
			return
		case id = <-c.joinCh:
			if oldCancel != nil {
				oldCancel()
				oldCancel = nil
			}

			go func() {
				runtime.Gosched()

				ctx, cancel := makeContext(c.Config.AssembleTimeoutS)
				oldCancel = cancel

				if err = c.stop(ctx, cancel); err != nil {
					c.logger.Error("gossip.Cluster.elect()", zap.Error(err))
					return
				}

				if err = c.assemble(ctx, cancel, id); err != nil {
					if !errors.Is(err, context.Canceled) {
						c.logger.Error("gossip.Cluster.assemble()", zap.Error(err))
					}

					return
				}

				if err = c.elect(ctx, cancel); err != nil {
					if !errors.Is(err, context.Canceled) {
						c.logger.Error("gossip.Cluster.elect()", zap.Error(err))
					}

					return
				}

				if err = c.assign(ctx, cancel); err != nil {
					if !errors.Is(err, context.Canceled) {
						c.logger.Error("gossip.Cluster.assign()", zap.Error(err))
					}

					return
				}

				if err = c.start(ctx, cancel); err != nil {
					if !errors.Is(err, context.Canceled) {
						c.logger.Error("gossip.Cluster.assign()", zap.Error(err))
					}

					return
				}
			}()
		case id = <-c.leaveCh:
			if oldCancel != nil {
				oldCancel()
			}

			ctx, cancel := makeContext(c.Config.AssembleTimeoutS)
			oldCancel = cancel

			go func() {
				runtime.Gosched()

				if err = c.stop(ctx, cancel); err != nil {
					c.logger.Error("gossip.Cluster.stop()", zap.Error(err))
					return
				}

				c.State.RemoveNode(id)

				if err = c.elect(ctx, cancel); err != nil {
					if !errors.Is(err, context.Canceled) {
						c.logger.Error("gossip.Cluster.elect()", zap.Error(err))
					}

					return
				}

				if err = c.assign(ctx, cancel); err != nil {
					if !errors.Is(err, context.Canceled) {
						c.logger.Error("gossip.Cluster.assign()", zap.Error(err))
					}

					return
				}
			}()
		}
	}
}

func (c *Cluster) join() error {
	var err error

	if err = c.State.Trigger(Join); err != nil {
		return fmt.Errorf("gossip.Cluster.join(), trigger 'Join' error: %w", err)
	}

	if _, err = c.Memberlist.Join(c.Config.JoinNodes); err != nil {
		return fmt.Errorf("gossip.Cluster.join() MemberList.Join error: %w", err)
	}

	if err = c.State.Trigger(Joined); err != nil {
		return fmt.Errorf("gossip.Cluster.join(), trigger 'Joined' error: %w", err)
	}

	return nil
}

func (c *Cluster) assemble(ctx context.Context, cancel context.CancelFunc, id uint16) error {
	var err error

	if c.State.HasNode(id) {
		return nil
	}

	select {
	case <-ctx.Done():
		c.logger.Warn("gossip.Cluster.assemble()", zap.Error(ctx.Err()))
		return nil
	default:
		if err = c.State.Trigger(Assemble); err != nil {
			cancel()
			c.logger.Warn("gossip.Cluster.assemble()", zap.Error(ctx.Err()))
		}
	}

	for {
		select {
		case <-ctx.Done():
			c.logger.Warn("gossip.Cluster.assemble()", zap.Error(ctx.Err()))
			return nil
		default:
		}

		if c.State.HasNode(id) {
			if err = c.State.Trigger(Assembled); err != nil {
				cancel()
				return fmt.Errorf("gossip.Cluster.assemble(): %w", err)
			}

			return nil
		}

		<-time.After(time.Second)
	}
}

func (c *Cluster) elect(ctx context.Context, cancel context.CancelFunc) error {
	var err error

	select {
	case <-ctx.Done():
		c.logger.Warn("gossip.Cluster.elect()", zap.Error(ctx.Err()))
		return ctx.Err()
	default:
		if err = c.State.Trigger(Elect); err != nil {
			cancel()
			c.logger.Warn("gossip.Cluster.elect()", zap.Error(ctx.Err()))
		}
	}

	for {
		select {
		case <-ctx.Done():
			c.logger.Warn("gossip.Cluster.elect()", zap.Error(ctx.Err()))
			return ctx.Err()
		default:
		}

		if c.State.ElectLeader() {
			if err = c.State.Trigger(Elected); err != nil {
				cancel()
				return fmt.Errorf("gossip.Cluster.elect(): %w", err)
			}

			break
		}

		<-time.After(time.Second)
	}

	return nil
}

func (c *Cluster) assign(ctx context.Context, cancel context.CancelFunc) error {
	var err error

	select {
	case <-ctx.Done():
		c.logger.Warn("gossip.Cluster.assign(), context canceled", zap.Error(ctx.Err()))
		return ctx.Err()
	default:
		if err = c.State.Trigger(Assign); err != nil {
			cancel()
			return fmt.Errorf("gossip.Cluster.assign(): %w", err)
		}
	}

	c.State.AssignWorkers()

	select {
	case <-ctx.Done():
		c.logger.Warn("gossip.Cluster.assign(), context canceled", zap.Error(ctx.Err()))
		return ctx.Err()
	default:
		if err = c.State.Trigger(Assigned); err != nil {
			cancel()
			return fmt.Errorf("gossip.Cluster.assign(), trigger 'Assigned' error: %w", err)
		}
	}

	return nil
}

func (c *Cluster) start(ctx context.Context, cancel context.CancelFunc) error {
	var err error

	select {
	case <-ctx.Done():
		c.logger.Warn("gossip.Cluster.start()", zap.Error(ctx.Err()))
		return ctx.Err()
	default:
		if err = c.State.Trigger(Start); err != nil {
			cancel()
			return fmt.Errorf("gossip.Cluster.start(): %w", err)
		}
	}

	c.State.StartWorkers()

	select {
	case <-ctx.Done():
		c.logger.Warn("gossip.Cluster.start()", zap.Error(ctx.Err()))
		return ctx.Err()
	default:
		if err = c.State.Trigger(Started); err != nil {
			cancel()
			return fmt.Errorf("gossip.Cluster.start(): %w", err)
		}
	}

	return nil
}

func (c *Cluster) stop(ctx context.Context, cancel context.CancelFunc) error {
	var err error

	if c.State.CurrentState() == Configuring && !c.State.LocalNodeState().Working {
		return nil
	}

	select {
	case <-ctx.Done():
		c.logger.Warn("gossip.Cluster.start()", zap.Error(ctx.Err()))
		return ctx.Err()
	default:
		if err = c.State.Trigger(Stop); err != nil {
			cancel()
			c.logger.Warn("gossip.Cluster.assign()", zap.Error(err))
			return nil
		}
	}

	c.State.StopWorkers()

	select {
	case <-ctx.Done():
		c.logger.Warn("gossip.Cluster.start()", zap.Error(ctx.Err()))
		return ctx.Err()
	default:
		if err = c.State.Trigger(Stopped); err != nil {
			cancel()
			return fmt.Errorf("gossip.Cluster.assign() error: %w", err)
		}
	}

	return nil
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

func makeContext(timeoutSec int) (context.Context, context.CancelFunc) {
	finishCh := make(chan struct{})
	ctx, cancel := context.WithTimeout(context.TODO(), time.Duration(timeoutSec)*time.Second)

	go func() {
		runtime.Gosched()

		select {
		case <-finishCh:
			cancel()
		case <-ctx.Done():
		}

		if cancel != nil {
			cancel = nil
		}
	}()

	return ctx, func() {
		select {
		case <-finishCh:
			return
		default:
			close(finishCh)
		}
	}
}
