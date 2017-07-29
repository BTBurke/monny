package wtf

import (
	"fmt"
	"os/exec"
	"regexp"
	"strconv"
	"time"
)

const api string = "https://api.lmkwtf.com"

type Config struct {
	ID              string
	Alert           []*regexp.Regexp
	AlertQuantity   int
	AlertPeriod     time.Duration
	NotifyTimeout   time.Duration
	KillTimeout     time.Duration
	Creates         []string
	StdoutHistory   int
	StderrHistory   int
	NotifyOnSuccess bool
	NotifyOnFailure bool
	Shell           string

	api string
}

type ConfigOption func(c *Config) error

func NewConfig(id string, options ...ConfigOption) (*Config, []error) {
	c := &Config{
		ID:              id,
		StdoutHistory:   50,
		StderrHistory:   50,
		NotifyOnSuccess: true,
		NotifyOnFailure: true,
		api:             api,
	}

	var errors []error
	for _, option := range options {
		err := option(c)
		if err != nil {
			errors = append(errors, err)
		}
	}
	if c.Shell == "" {
		shell, err := exec.LookPath("bash")
		if err != nil {
			errors = append(errors, fmt.Errorf("no default shell found, specify path to shell using option --shell"))
		}
		c.Shell = shell
	}

	if len(errors) > 0 {
		return nil, errors
	}
	return c, nil
}

func Alert(regex string) ConfigOption {
	return func(c *Config) error {
		reg, err := regexp.Compile(regex)
		c.Alert = append(c.Alert, reg)
		return err
	}
}

func AlertQuantity(quantity string) ConfigOption {
	return func(c *Config) error {
		qty, err := strconv.Atoi(quantity)
		if err != nil {
			return fmt.Errorf("could not convert alert-quantity to integer")
		}
		c.AlertQuantity = qty
		return nil
	}
}

func AlertPeriod(period string) ConfigOption {
	return func(c *Config) error {
		duration, err := time.ParseDuration(period)
		if err != nil {
			return fmt.Errorf("could not convert alert-period to time")
		}
		c.AlertPeriod = duration
		return nil
	}
}

func StdoutHistory(h int) ConfigOption {
	return func(c *Config) error {
		c.StdoutHistory = h
		return nil
	}
}

func NoNotifyOnSuccess() ConfigOption {
	return func(c *Config) error {
		c.NotifyOnSuccess = false
		return nil
	}
}

func NoNotifyOnFailure() ConfigOption {
	return func(c *Config) error {
		c.NotifyOnFailure = false
		return nil
	}
}

func Shell(shellPath string) ConfigOption {
	return func(c *Config) error {
		shell, err := exec.LookPath(shellPath)
		if err != nil {
			return fmt.Errorf("no shell found at path %s", shellPath)
		}
		c.Shell = shell
		return nil
	}
}

func KillTimeout(timeout string) ConfigOption {
	return func(c *Config) error {
		duration, err := time.ParseDuration(timeout)
		if err != nil {
			return fmt.Errorf("unrecognized kill timeout duration: %s")
		}
		c.KillTimeout = duration
		return nil
	}
}

func NotifyTimeout(timeout string) ConfigOption {
	return func(c *Config) error {
		duration, err := time.ParseDuration(timeout)
		if err != nil {
			return fmt.Errorf("unrecognized notify timeout duration: %s")
		}
		c.NotifyTimeout = duration
		return nil
	}
}

func Creates(filepath string) ConfigOption {
	return func(c *Config) error {
		c.Creates = append(c.Creates, filepath)
		return nil
	}
}
