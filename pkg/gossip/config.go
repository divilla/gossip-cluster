package gossip

const (
	defaultMinNodesNum      = 3
	defaultJoinTimeoutS     = 10
	defaultAssembleTimeoutS = 30
)

type (
	Config struct {
		Name               string `yaml:"name"`
		BindAddr           string `yaml:"bind_addr"`
		BindPort           int    `yaml:"bind_port"`
		AdvertiseAddr      string `yaml:"advertise_addr"`
		AdvertisePort      int    `yaml:"advertise_port"`
		PushPullIntervalMS int    `yaml:"push_pull_interval_ms"`

		JoinNodes        []string `yaml:"join_nodes"`
		JoinNodesNum     int      `yaml:"join_nodes_num"`
		JoinTimeoutS     int      `yaml:"join_timeout_s"`
		AssembleTimeoutS int      `yaml:"assemble_timeout_s"`
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

	return c
}
