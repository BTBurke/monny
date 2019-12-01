package monny

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"
)

const api string = "https://report.lmkwtf.com"
const port string = "443"

// Config stores configuration data for the monitoring service.  Functional options are
// used to modify the configuration based on command-line flags or optional YAML configuration.
// See documentation of individual functional options for descriptions.
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

	host   string
	port   string
	useTLS bool
	out    io.WriteCloser
	err    io.WriteCloser
}

type rule struct {
	Field string
	Regex *regexp.Regexp
}

// ConfigOption is a function for validating and setting configuration values
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
		out:             os.Stdout,
		err:             os.Stderr,
	}

	var errors []error
	for _, option := range options {
		err := option(&c)
		if err != nil {
			errors = append(errors, err)
		}
	}

	shell, err := findDefaultShell()
	if err != nil {
		errors = append(errors, err)
	}
	c.Shell = shell
	if len(c.ID) == 0 {
		errors = append(errors, fmt.Errorf("id is required, use monny -i <id>; new ids are created with monctl create or pass your email address to get a notifications via email without an account"))
	}

	if len(errors) > 0 {
		return Config{}, errors
	}
	return c, nil
}

func findDefaultShell() (string, error) {
	shell := os.Getenv("SHELL")
	if len(shell) == 0 {
		return shell, fmt.Errorf("could not determine default shell, set with --shell=<full path to shell>")
	}
	return shell, nil
}

// ID of this monitor, used to connect the report with the notification
// channels configured via monctl.  ID can simply be an email address to receive
// notifications without a configured account on the server.  The server must be configured
// to allow anonymous reporting with only an email address (default: disabled).
func ID(id string) ConfigOption {
	return func(c *Config) error {
		c.ID = id
		return nil
	}
}

// Rule that reports on regex match to stdout or stderr
func Rule(regex string) ConfigOption {
	return func(c *Config) error {
		reg, err := regexp.Compile(regex)
		c.Rules = append(c.Rules, rule{Regex: reg})
		return err
	}
}

// JSONRule is like Rule except the stdout or stderr is unmarshaled to a JSON object and
// the regex match is applied to a particular field.  Nested fields are selected by flattening
// the path.
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

// RuleQuantity creates reports when the total number of rule matches exceeds this value.  To
// report on a rate, set RulePeriod to a duration and reports are generated when the rate exceeds
// RuleQuantity/RulePeriod
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

// RulePeriod is used in conjunction with RuleQuantity to send reports when the rate of rule matches
// exceceds RuleQuantity/RulePeriod. Expects a time.Duration in string format (e.g. 10m, 1h)
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

// StdoutHistory sets the max number of lines of stdout to send with the report (default 30)
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

// StderrHistory sets the max number of lines of stderr to send with the report (default 30)
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

// NoNotifyOnSuccess prevents sending success reports which are necessary for deadman's switch
// notifications and command completion history
func NoNotifyOnSuccess() ConfigOption {
	return func(c *Config) error {
		c.NotifyOnSuccess = false
		return nil
	}
}

// NoNotifyOnFailure prevents sending failure reports.  This can be useful if the process does
// not use standard exit return values and the failure reports are false positives.
func NoNotifyOnFailure() ConfigOption {
	return func(c *Config) error {
		c.NotifyOnFailure = false
		return nil
	}
}

// Daemon indicates that this is a long-running process so that rule matches and other reports
// are sent immediately instead of waiting for process termination.
func Daemon() ConfigOption {
	return func(c *Config) error {
		c.Daemon = true
		return nil
	}
}

// MemoryWarn sends a report when process memory exceeds this value.  Expects a string with
// units in K, M, or G.  (Linux only, memory measurements on Darwin or Windows is a no-op)
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

// MemoryKill kills the process and sends a report when process memory exceeds this value.  Expects a string with
// units in K, M, or G.  (Linux only, memory measurements on Darwin or Windows is a no-op)
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

// KillTimeout kills the process and sends a report when process run time exceeds the duration set.  Duration
// is expressed as a string with unit ns, us, ms, s, m, h.
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

// NotifyTimeout sends a report when process run time exceeds the duration set.  Duration
// is expressed as a string with unit ns, us, ms, s, m, h.
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

// Creates generates a report when an expected file is not created as a result of the process.
// Expects a filepath that will be checked on process completion.
func Creates(filepath string) ConfigOption {
	return func(c *Config) error {
		c.Creates = append(c.Creates, filepath)
		return nil
	}
}

// Host sets the url and port when using a private reporting server.  Expects host:port.
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

// Insecure allows a non-TLS connection to a private reporting server.  This option should only
// be used when the reporting server and the monitor communicate over a private internal network.
func Insecure() ConfigOption {
	return func(c *Config) error {
		c.useTLS = false
		return nil
	}
}

// NoErrorReports prevents unhandled errors from being reported to monny.dev to improve the quality
// and stability of the software.  No private data is sent (e.g., no stdout, stderr, or any config data).
// The only information sent is the text of the error and a stack trace.
func NoErrorReports() ConfigOption {
	return func(c *Config) error {
		SuppressErrorReporting = true
		return nil
	}
}

// Shell sets the shell that will execute the command.  If an absolute path is not specified, the search
// path will be checked for the executable.
func Shell(shell string) ConfigOption {
	return func(c *Config) error {
		path, err := exec.LookPath(shell)
		if err != nil {
			return err
		}
		c.Shell = path
		return nil
	}
}

// LogFile sends Stdout and Stderr to log rotated files in the given directory.  It will create the
// directory if it does not exist.  An error will be returned if the user does not have write permission
// to create (if the directory does not already exist) or write to the directory.
func LogFile(dir string) ConfigOption {
	return func(c *Config) error {
		// TODO: add log rotator
		return nil
	}
}

// logOut redirects Stdout to out
func logOut(out io.WriteCloser) ConfigOption {
	return func(c *Config) error {
		c.out = out
		return nil
	}
}

// logErr redirects Stderr to err
func logErr(err io.WriteCloser) ConfigOption {
	return func(c *Config) error {
		c.err = err
		return nil
	}
}
