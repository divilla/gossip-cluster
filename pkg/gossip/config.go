package gossip

const (
	defaultMinNodesNum      = 3
	defaultJoinTimeoutS     = 10
	defaultAssembleTimeoutS = 30
	defaultElectLeaderS     = 30
)

type (
	Config struct {
		NodeID             uint16 `yaml:"node_id"`
		BindAddr           string `yaml:"bind_addr"`
		BindPort           int    `yaml:"bind_port"`
		AdvertiseAddr      string `yaml:"advertise_addr"`
		AdvertisePort      int    `yaml:"advertise_port"`
		PushPullIntervalMS int    `yaml:"push_pull_interval_ms"`

		First            bool     `yaml:"first"`
		JoinNodes        []string `yaml:"join_nodes"`
		JoinNodesNum     int      `yaml:"join_nodes_num"`
		JoinTimeoutS     int      `yaml:"join_timeout_s"`
		AssembleTimeoutS int      `yaml:"assemble_timeout_s"`
		ElectLeaderS     int      `yaml:"elect_leader_s"`

		Debug bool `yaml:"debug"`
	}
)

func parseDefaults(c *Config) *Config {
	if c.JoinNodesNum == 0 {
		c.JoinNodesNum = defaultMinNodesNum
	}

	if c.JoinTimeoutS == 0 {
		c.JoinTimeoutS = defaultJoinTimeoutS
	}

	if c.AssembleTimeoutS == 0 {
		c.AssembleTimeoutS = defaultAssembleTimeoutS
	}

	if c.ElectLeaderS == 0 {
		c.ElectLeaderS = defaultElectLeaderS
	}

	return c
}
