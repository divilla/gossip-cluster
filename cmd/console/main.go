package main

import (
	"github.com/divilla/gossip-cluster/internal/config"
	"github.com/divilla/gossip-cluster/pkg/gossip"
	"go.uber.org/zap"
	"os"
	"os/signal"
)

func main() {
	stopCh := make(chan struct{})
	quitCh := make(chan os.Signal, 1)
	signal.Notify(quitCh, os.Interrupt)

	logger, err := zap.NewProduction()
	if err != nil {
		panic(err)
	}
	defer logger.Sync()

	cfg, err := config.New("config/config.yaml")
	if err != nil {
		panic(err)
	}

	if _, err = gossip.NewCluster(logger, cfg.Nodes[0], stopCh); err != nil {
		panic(err)
	}
	if _, err = gossip.NewCluster(logger, cfg.Nodes[1], stopCh); err != nil {
		panic(err)
	}
	if _, err = gossip.NewCluster(logger, cfg.Nodes[2], stopCh); err != nil {
		panic(err)
	}

	<-quitCh
	close(stopCh)
}
