package config

import (
	"errors"
	"time"
)

type Config struct {
	Targets      []string      `mapstructure:"targets"`
	WorkerCount  int           `mapstructure:"workers"`
	Timeout      time.Duration `mapstructure:"timeout"`
	OutputFormat string        `mapstructure:"format"`
}

func DefaultConfig() *Config {
	c := &Config{
		WorkerCount:  10,
		Timeout:      5 * time.Second,
		OutputFormat: "table",
	}
	return c
}

func (c *Config) Validate() error {
	if c.WorkerCount <= 0 {
		return errors.New("worker count must be greater than 0")
	}
	if c.Timeout <= 0 {
		return errors.New("timeout must be greater than 0")
	}
	if c.OutputFormat != "table" && c.OutputFormat != "json" {
		return errors.New("output format must be table or json")
	}
	return nil
}
