package main

import (
	"fmt"
	"github.com/divilla/gossip-cluster/internal/config"
	"github.com/divilla/gossip-cluster/pkg/gossip"
	"go.uber.org/zap"
	"os"
	"os/signal"
	"time"
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

	nc := cfg.Nodes[0]
	nc.First = true
	cfg.Nodes[0] = nc

	c1, err := gossip.NewCluster(logger, cfg.Nodes[0], stopCh)
	if err != nil {
		panic(err)
	}
	_, err = gossip.NewCluster(logger, cfg.Nodes[1], stopCh)
	if err != nil {
		panic(err)
	}
	if _, err = gossip.NewCluster(logger, cfg.Nodes[2], stopCh); err != nil {
		panic(err)
	}

	<-time.After(16 * time.Second)
	fmt.Println()
	fmt.Println("-----------------------------------------------------------------------------------------------")
	fmt.Println("*********************************** Shutting down 1 *******************************************")
	fmt.Println("-----------------------------------------------------------------------------------------------")
	fmt.Println()
	c1.Memberlist.Shutdown()

	<-time.After(32 * time.Second)
	nc.First = false
	cfg.Nodes[0] = nc
	fmt.Println()
	fmt.Println("-----------------------------------------------------------------------------------------------")
	fmt.Println("*********************************** Starting up 1 *******************************************")
	fmt.Println("-----------------------------------------------------------------------------------------------")
	fmt.Println()
	c1, err = gossip.NewCluster(logger, cfg.Nodes[0], stopCh)

	<-quitCh
	close(stopCh)
}
