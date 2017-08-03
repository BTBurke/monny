package monitor

import (
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"
)

const api string = "https://report.lmkwtf.com"
const port string = "443"

type Config struct {
	ID              string
	Rules           []rule
	RuleQuantity    int
	RulePeriod      time.Duration
	Hostname        string
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

	host      string
	port      string
	useTLS    bool
	comingled []string
}

type rule struct {
	Field string
	Regex *regexp.Regexp
}

type ConfigOption func(c *Config) error

func newConfig(options ...ConfigOption) (Config, []error) {
	host, err := os.Hostname()
	if err != nil {
		host = ""
	}
	c := Config{
		StdoutHistory:   30,
		StderrHistory:   30,
		NotifyOnSuccess: true,
		NotifyOnFailure: true,
		Hostname:        host,
		host:            api,
		port:            port,
		useTLS:          true,
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
	if len(c.comingled) > 0 {
		errors = append(errors, fmt.Errorf("unknown options: %s\n\nIf these are command-line options for your process add a blank flag separator (--) between the commands like:\nwtf -c config.yaml -- mycommand.sh --myoption", strings.Join(c.comingled, ",")))
	}
	if len(c.ID) == 0 {
		errors = append(errors, fmt.Errorf("id is required, use wtf -i <id>; new ids are created with wtfctl create"))
	}

	if len(errors) > 0 {
		return Config{}, errors
	}
	return c, nil
}

func ID(id string) ConfigOption {
	return func(c *Config) error {
		c.ID = id
		return nil
	}
}

func Rule(regex string) ConfigOption {
	return func(c *Config) error {
		reg, err := regexp.Compile(regex)
		c.Rules = append(c.Rules, rule{Regex: reg})
		return err
	}
}

func JSONRule(field string, regex string) ConfigOption {
	return func(c *Config) error {
		reg, err := regexp.Compile(regex)
		c.Rules = append(c.Rules, rule{
			Field: field,
			Regex: reg,
		})
		return err
	}
}

func RuleQuantity(quantity string) ConfigOption {
	return func(c *Config) error {
		qty, err := strconv.Atoi(quantity)
		if err != nil {
			return fmt.Errorf("could not convert rule-quantity to integer")
		}
		c.RuleQuantity = qty
		return nil
	}
}

func RulePeriod(period string) ConfigOption {
	return func(c *Config) error {
		duration, err := time.ParseDuration(period)
		if err != nil {
			return fmt.Errorf("could not convert rule-period to time")
		}
		c.RulePeriod = duration
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
			return fmt.Errorf("unrecognized kill timeout duration: %s", timeout)
		}
		c.KillTimeout = duration
		return nil
	}
}

func NotifyTimeout(timeout string) ConfigOption {
	return func(c *Config) error {
		duration, err := time.ParseDuration(timeout)
		if err != nil {
			return fmt.Errorf("unrecognized notify timeout duration: %s", timeout)
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

func Host(pathWithPort string) ConfigOption {
	return func(c *Config) error {
		h := strings.Split(pathWithPort, ":")
		if len(h) != 2 {
			return fmt.Errorf("unknown host, use host:port")
		}
		c.host = h[0]
		c.port = h[1]
		return nil
	}
}

func Insecure() ConfigOption {
	return func(c *Config) error {
		c.useTLS = false
		return nil
	}
}

func NoErrorReports() ConfigOption {
	return func(c *Config) error {
		SuppressErrorReporting = true
		return nil
	}
}

func comingledOption(option string) ConfigOption {
	return func(c *Config) error {
		c.comingled = append(c.comingled, option)
		return nil
	}
}
