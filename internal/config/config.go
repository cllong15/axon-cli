package config

import (
	"errors"
	"strings"
	"time"
)

type Config struct {
	Targets      []string      `mapstructure:"targets"`
	WorkerCount  int           `mapstructure:"workers"`
	Timeout      time.Duration `mapstructure:"timeout"`
	OutputFormat string        `mapstructure:"format"`
	Method       string        `mapstructure:"method"`
	FailOnError  bool          `mapstructure:"fail_on_error"`
	OnlyErrors   bool          `mapstructure:"only_errors"`
	Threshold    time.Duration `mapstructure:"threshold"`
}

func DefaultConfig() *Config {
	c := &Config{
		WorkerCount:  10,
		Timeout:      5 * time.Second,
		OutputFormat: "table",
		Method:       "GET",
		FailOnError:  false,
	}
	return c
}

func (c *Config) Validate() error {
	if c.Threshold < 0 {
		return errors.New("threshold must be positive")
	}
	if c.WorkerCount <= 0 {
		return errors.New("worker count must be greater than 0")
	}
	if c.Timeout <= 0 {
		return errors.New("timeout must be greater than 0")
	}
	if c.OutputFormat != "table" && c.OutputFormat != "json" {
		return errors.New("output format must be table or json")
	}
	c.Method = strings.ToUpper(strings.TrimSpace(c.Method))
	switch c.Method {
	case "GET", "HEAD", "POST", "PUT", "DELETE", "OPTIONS":
		//valid
	default:
		return errors.New("unsupported HTTP method; supported: GET, HEAD, POST, PUT, DELETE, OPTIONS")
	}
	return nil
}
