package monitor

import (
	"fmt"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"
)

const api string = "https://notify.lmkwtf.com"

type Config struct {
	ID              string
	Alerts          []alert
	AlertQuantity   int
	AlertPeriod     time.Duration
	NotifyTimeout   time.Duration
	KillTimeout     time.Duration
	MemoryWarn      uint64
	MemoryKill      uint64
	Daemon          bool
	Creates         []string
	StdoutHistory   int
	StderrHistory   int
	NotifyOnSuccess bool
	NotifyOnFailure bool
	Shell           string

	url string
}

type alert struct {
	Field string
	Regex *regexp.Regexp
}

type ConfigOption func(c *Config) error

func newConfig(id string, options ...ConfigOption) (Config, []error) {
	c := Config{
		ID:              id,
		StdoutHistory:   50,
		StderrHistory:   50,
		NotifyOnSuccess: true,
		NotifyOnFailure: true,
		url:             api,
	}

	var errors []error
	for _, option := range options {
		err := option(&c)
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
		return Config{}, errors
	}
	return c, nil
}

func Alert(regex string) ConfigOption {
	return func(c *Config) error {
		reg, err := regexp.Compile(regex)
		c.Alerts = append(c.Alerts, alert{Regex: reg})
		return err
	}
}

func JSONAlert(field string, regex string) ConfigOption {
	return func(c *Config) error {
		reg, err := regexp.Compile(regex)
		c.Alerts = append(c.Alerts, alert{
			Field: field,
			Regex: reg,
		})
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

func StdoutHistory(h string) ConfigOption {
	return func(c *Config) error {
		hist, err := strconv.Atoi(h)
		if err != nil {
			return err
		}
		c.StdoutHistory = hist
		return nil
	}
}

func StderrHistory(h string) ConfigOption {
	return func(c *Config) error {
		hist, err := strconv.Atoi(h)
		if err != nil {
			return err
		}
		c.StderrHistory = hist
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

func Daemon() ConfigOption {
	return func(c *Config) error {
		c.Daemon = true
		return nil
	}
}

func MemoryWarn(mem string) ConfigOption {
	return func(c *Config) error {
		var err error
		var warn int
		switch {
		case strings.HasSuffix(mem, "K"):
			warn, err = strconv.Atoi(mem[0 : len(mem)-1])
		case strings.HasSuffix(mem, "M"):
			warn, err = strconv.Atoi(mem[0 : len(mem)-1])
			warn = warn * 1000
		case strings.HasSuffix(mem, "G"):
			warn, err = strconv.Atoi(mem[0 : len(mem)-1])
			warn = warn * 1000000
		default:
			warn, err = strconv.Atoi(mem)
		}
		if err != nil {
			return fmt.Errorf("could not parse memory warning limit: %s", mem)
		}
		c.MemoryWarn = uint64(warn)
		return nil
	}
}

func MemoryKill(mem string) ConfigOption {
	return func(c *Config) error {
		var err error
		var kill int
		switch {
		case strings.HasSuffix(mem, "K"):
			kill, err = strconv.Atoi(mem[0 : len(mem)-1])
		case strings.HasSuffix(mem, "M"):
			kill, err = strconv.Atoi(mem[0 : len(mem)-1])
			kill = kill * 1000
		case strings.HasSuffix(mem, "G"):
			kill, err = strconv.Atoi(mem[0 : len(mem)-1])
			kill = kill * 1000000
		default:
			kill, err = strconv.Atoi(mem)
		}
		if err != nil {
			return fmt.Errorf("could not parse memory warning limit: %s", mem)
		}
		c.MemoryKill = uint64(kill)
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
