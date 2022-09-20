package main

import (
	"fmt"
	"github.com/divilla/gossip-cluster/internal/memlistconf"
	"github.com/gookit/gcli/v3"
	"github.com/hashicorp/memberlist"
	"go.uber.org/zap"
	"log"
	"os"
	"os/signal"
	"time"
)

type (
	options struct {
		DNSName            string
		BindIPAddress      string
		BindPort           int
		AdvertiseIPAddress string
		AdvertisePort      int
	}
)

func main() {
	//stopCh := make(chan struct{})
	quitCh := make(chan os.Signal, 1)
	signal.Notify(quitCh, os.Interrupt)

	logger, err := zap.NewProduction()
	if err != nil {
		panic(err)
	}
	defer logger.Sync()

	opt := options{}

	app := gcli.NewApp()
	app.Version = "0.1"
	app.Desc = "Gossip Cluster"
	// app.SetVerbose(gcli.VerbDebug)

	//app.Add(cmd.Example)

	app.Add(&gcli.Command{
		Name:       "start",
		Desc:       "Start a cluster with first node.",
		Examples:   "gc start --name localhost --ip 127.0.0.1 --port 8081",
		Flags:      makeFlags(),
		Func:       makeStartCommand(logger, &opt, quitCh),
		Help:       "this is help",
		HelpRender: nil,
		Config:     makeConfig(&opt),
	})

	app.Add(&gcli.Command{
		Name:       "join",
		Desc:       "Join a cluster with nodes passed as arguments.",
		Examples:   "gc start --name localhost --ip 127.0.0.1 --port 8082 127.0.0.1:8081",
		Flags:      makeFlags(),
		Func:       makeJoinCommand(logger, &opt, quitCh),
		Help:       "this is help",
		HelpRender: nil,
		Config:     makeConfig(&opt),
	})

	app.Run(nil)

	<-quitCh
}

func makeFlags() gcli.Flags {
	flags := gcli.NewFlags()
	flags.SetConfig(&gcli.FlagsConfig{
		WithoutType: true,
		DescNewline: false,
		Alignment:   gcli.AlignLeft,
		TagName:     gcli.FlagTagName,
	})

	return *flags
}

func makeStartCommand(logger *zap.Logger, opt *options, quitCh chan os.Signal) func(*gcli.Command, []string) error {
	return func(cmd *gcli.Command, args []string) error {
		cfg := memberlist.DefaultLANConfig()
		parseOptions(cfg, opt)

		ml, err := memberlist.Create(cfg)
		if err != nil {
			return fmt.Errorf("failed to create first node: %w", err)
		}

		tlq := memlistconf.NewTLQ(ml)
		gs := memlistconf.NewState(logger, ml.LocalNode())
		ms := memlistconf.NewMessenger(logger, ml, tlq)
		cfg.Delegate = memlistconf.NewDelegate(logger, ml, tlq, gs)

		i := 0
		for {
			if ml.NumMembers() == 3 {
				if err = gs.Event(memlistconf.Assemble); err != nil {
					return err
				}
				break
			}

			select {
			case <-quitCh:
				return err
			case <-time.After(time.Second):
				i++
				logger.Warn("assemble failed", zap.Int("attempt", i+1), zap.Int("num_members", ml.NumMembers()))
			}
		}

		logger.Info("local_node", zap.String("state", gs.LocalNode()[memlistconf.State].(string)))
		ms.Broadcast("ln", []byte("assembled"))
		fmt.Println()
		<-quitCh

		return nil
	}
}

func makeJoinCommand(logger *zap.Logger, opt *options, quitCh chan os.Signal) func(*gcli.Command, []string) error {
	return func(cmd *gcli.Command, args []string) error {
		cfg := memberlist.DefaultLANConfig()
		cfg.Logger = log.Default()
		parseOptions(cfg, opt)

		ml, err := memberlist.Create(cfg)
		if err != nil {
			return fmt.Errorf("failed to create follower node: %w", err)
		}

		tlq := memlistconf.NewTLQ(ml)
		gs := memlistconf.NewState(logger, ml.LocalNode())
		ms := memlistconf.NewMessenger(logger, ml, tlq)
		cfg.Delegate = memlistconf.NewDelegate(logger, ml, tlq, gs)

		if err = gs.Event(memlistconf.Join); err != nil {
			return err
		}
		for i := 0; i < 30; i++ {
			if _, err = ml.Join(args); err == nil {
				if err = gs.Event(memlistconf.Joined); err != nil {
					return err
				}
				break
			}

			select {
			case <-quitCh:
				return nil
			case <-time.After(time.Second):
				logger.Warn("join failed", zap.Int("attempt", i+1), zap.Int("nr", ml.NumMembers()), zap.Error(err))
			}
		}

		i := 0
		if err = gs.Event(memlistconf.Assemble); err != nil {
			return err
		}
		for {
			if ml.NumMembers() == 3 {
				if err = gs.Event(memlistconf.Assembled); err != nil {
					return err
				}
				break
			}

			select {
			case <-quitCh:
				return nil
			case <-time.After(time.Second):
				i++
				logger.Warn("assemble failed", zap.Int("attempt", i+1), zap.Int("num_members", ml.NumMembers()))
			}
		}

		ms.Broadcast("join", []byte("joined"))
		logger.Info("local_node", zap.String("state", gs.LocalNode()[memlistconf.State].(string)))
		fmt.Println()
		<-quitCh

		return nil
	}
}

func parseOptions(cfg *memberlist.Config, opt *options) {
	if opt.DNSName != "" {
		cfg.Name = opt.DNSName
	}

	if opt.BindIPAddress != "" {
		cfg.BindAddr = opt.BindIPAddress
		cfg.AdvertiseAddr = opt.BindIPAddress
	}
	if opt.AdvertiseIPAddress != "" {
		cfg.AdvertiseAddr = opt.AdvertiseIPAddress
	}

	if opt.BindPort != 0 {
		cfg.BindPort = opt.BindPort
		cfg.AdvertisePort = opt.BindPort
	}
	if opt.AdvertisePort != 0 {
		cfg.AdvertisePort = opt.AdvertisePort
	}
}

func makeConfig(opt *options) func(*gcli.Command) {
	return func(c *gcli.Command) {
		c.AddArg("nodes", "Cluster nodes list.", false, true)

		c.StrVar(&opt.DNSName, &gcli.FlagMeta{
			Name:     "name",
			Desc:     "Nodes's dns name.",
			Shorts:   []string{"n"},
			Required: false,
		})
		c.StrVar(&opt.BindIPAddress, &gcli.FlagMeta{
			Name:     "ip",
			Desc:     "Address to bind to. The port is used for both UDP and TCP gossip.",
			Shorts:   []string{"i"},
			Required: false,
		})
		c.IntVar(&opt.BindPort, &gcli.FlagMeta{
			Name:     "port",
			Desc:     "Port to listen on. The port is used for both UDP and TCP gossip.",
			Shorts:   []string{"p"},
			Required: false,
		})
		c.StrVar(&opt.AdvertiseIPAddress, &gcli.FlagMeta{
			Name:     "advertise-ip",
			Desc:     "Address to advertise to other cluster members. Used for nat traversal.",
			Shorts:   []string{"ai"},
			Required: false,
		})
		c.IntVar(&opt.AdvertisePort, &gcli.FlagMeta{
			Name:     "advertise-port",
			Desc:     "Port to advertise to other cluster members. Used for nat traversal.",
			Shorts:   []string{"ap"},
			Required: false,
		})
	}
}
