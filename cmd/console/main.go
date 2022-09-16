package main

import (
	"fmt"
	"github.com/divilla/gossip-cluster/internal/xml"
	"github.com/gookit/gcli/v3"
	"github.com/gookit/gcli/v3/_examples/cmd"
	"github.com/hashicorp/memberlist"
	"go.uber.org/zap"
	"os"
	"os/signal"
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

	cfg := memberlist.DefaultLocalConfig()
	ml, err := memberlist.Create(memberlist.DefaultLocalConfig())
	if err != nil {
		panic("Failed to create member list: " + err.Error())
	}
	cfg.Delegate = xml.NewDelegate(logger, ml)

	options := struct {
		Alias string
	}{}

	app := gcli.NewApp()
	app.Version = "1.0.3"
	app.Desc = "Gossip Cluster"
	// app.SetVerbose(gcli.VerbDebug)

	app.Add(cmd.Example)
	app.Add(&gcli.Command{
		Name: "demo",
		// allow color tag and {$cmd} will be replace to 'demo'
		Desc: "this is a description <info>message</> for {$cmd}",
		Subs: []*gcli.Command{
			// ... allow add subcommands
		},
		Aliases: []string{"dm"},
		Func: func(cmd *gcli.Command, args []string) error {
			gcli.Print("hello, in the demo command\n")
			return nil
		},
	})
	app.Add(&gcli.Command{
		Name:     "start",
		Desc:     "start leader",
		Examples: "gc start --alias host-1 127.0.0.1:8081",
		Func: func(cmd *gcli.Command, args []string) error {
			//if len(args) == 0 {
			//	gcli.Print("missing argument(s), run gc start --help\n")
			//}
			//
			//ln := ml.LocalNode()
			//ln.Name = options.Alias
			//
			//address := strings.Split(args[0], ":")
			//if len(address) == 1 {
			//	ip := net.ParseIP(address[0])
			//	if ip == nil {
			//		return errors.New("invalid ip address, expected format: 127.0.0.1")
			//	}
			//
			//	ln.Addr = ip
			//}
			//if len(address) == 2 {
			//	ip := net.ParseIP(address[0])
			//	if ip == nil {
			//		return errors.New("invalid ip address, expected format: 127.0.0.1:8001")
			//	}
			//
			//	ln.Addr = ip
			//
			//	if port, err := strconv.Atoi(address[1]); err != nil {
			//		return errors.New("invalid port, expected format: 127.0.0.1:8001")
			//	} else {
			//		ln.Port = uint16(port)
			//	}
			//}

			for _, member := range ml.Members() {
				fmt.Printf("Member: %s %s:%d\n", member.Name, member.Addr, member.Port)
			}

			<-quitCh

			return nil
		},
		Help:       "this is help",
		HelpRender: nil,
		Config: func(c *gcli.Command) {
			c.AddArg("address", "host:port", true)
			c.StrVar(&options.Alias, &gcli.FlagMeta{
				Name:     "alias",
				Desc:     "alias to be used instead of address:port",
				Shorts:   []string{"a"},
				Required: true,
			})
		},
	})

	app.Run(nil)

	<-quitCh
}
