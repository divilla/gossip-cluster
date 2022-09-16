package config

import (
	"errors"
	"fmt"
	"gopkg.in/yaml.v3"
	"io/ioutil"
	"os"
)

type Config struct {
	ClearDB               bool     `yaml:"clear_db"`
	ReportAfter           int      `yaml:"report_after"`
	TotalWorkers          int      `yaml:"total_workers"`
	OperationsTimeout     int64    `yaml:"operations_timeout"`
	ExcludeTables         []string `yaml:"exclude_tables"`
	SeedRows              int      `yaml:"seed_rows"`
	InsertSkipNullableOdd int64    `yaml:"insert_skip_nullable_odd"`
	UpdateFieldOdd        int64    `yaml:"update_field_odd"`
	AlwaysUpdate          []string `yaml:"always_update"`
	SkipFields            []string `yaml:"skip_fields"`

	DSNList []struct {
		Name string `yaml:"name"`
		URL  string `yaml:"url"`
	} `yaml:"dsn_list"`

	CommandOdds struct {
		Insert uint64 `yaml:"insert"`
		Update uint64 `yaml:"update"`
		Delete uint64 `yaml:"delete"`
	} `yaml:"command_odds"`
	TotalCommandOdds uint64
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

		cfg.TotalCommandOdds = cfg.CommandOdds.Insert + cfg.CommandOdds.Update + cfg.CommandOdds.Delete

		return cfg, nil
	}

	return nil, os.ErrNotExist
}
