package config

import (
	"errors"
	"fmt"
	"github.com/divilla/gossip-cluster/pkg/gossip"
	"gopkg.in/yaml.v3"
	"io/ioutil"
	"os"
)

type Config struct {
	Nodes []*gossip.Config `yaml:"nodes"`
}

func New(paths ...string) (*Config, error) {
	var file []byte
	var err error

	cfg := new(Config)
	for _, path := range paths {
		if _, err = os.Stat(path); errors.Is(err, os.ErrNotExist) {
			continue
		}

		if file, err = ioutil.ReadFile(path); err != nil {
			return nil, fmt.Errorf("ioutil.ReadFile() %w", err)
		}

		if err = yaml.Unmarshal(file, cfg); err != nil {
			return nil, fmt.Errorf("yaml.Unmarshal() %w", err)
		}

		return cfg, nil
	}

	return nil, os.ErrNotExist
}
