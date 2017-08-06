package monitor

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"os/signal"
	"regexp"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/BTBurke/wtf/proto"
)

// Command represents the current state of process execution
type Command struct {
	Config        Config
	UserCommand   []string
	Stdout        []string
	Stderr        []string
	Success       bool
	RuleMatches   []RuleMatch
	Killed        bool
	KillReason    proto.KillReason
	Created       []File
	MaxMemory     uint64
	ReportReason  proto.ReportReason
	Start         time.Time
	Finish        time.Time
	Duration      time.Duration
	ExitCode      int32
	ExitCodeValid bool
	Messages      []string

	mutex        sync.Mutex
	pid          int
	memWarnSent  bool
	timeWarnSent bool
	handler      ProcessHandlers
	report       ReportSender
	cleanup      []func() error
}

// File represents an artifact that is produced by the process.
// When configured, the failure to create the file is treated as
// a trigger for a report.
type File struct {
	Path string
	Size int64
	Time time.Time
}

type RuleMatch struct {
	Time  time.Time
	Line  string
	Index [][]int
}

func New(usercmd []string, options ...ConfigOption) (*Command, []error) {
	cfg, err := newConfig(options...)
	if len(err) > 0 {
		return nil, err
	}
	return &Command{
		Config:      cfg,
		UserCommand: usercmd,
		handler:     handler{},
		report: &Report{
			Host: cfg.host,
			Port: cfg.port,
		},
	}, nil
}

func (c *Command) Exec() error {
	var cmd *exec.Cmd
	wrappedCmd, cleanup, err := wrapComplexCommand(c.Config.Shell, c.UserCommand)
	if err != nil {
		return err
	}
	c.cleanup = append(c.cleanup, cleanup)

	switch len(wrappedCmd) {
	case 1:
		cmd = exec.Command(wrappedCmd[0])
	default:
		cmd = exec.Command(wrappedCmd[0], wrappedCmd[1:]...)
	}
	stdoutReader, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	stderrReader, err := cmd.StderrPipe()
	if err != nil {
		return err
	}
	stdoutScanner := bufio.NewScanner(stdoutReader)
	stderrScanner := bufio.NewScanner(stderrReader)

	c.Start = time.Now()
	if err := cmd.Start(); err != nil {
		return err
	}
	c.pid = os.Getpid()

	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		for stdoutScanner.Scan() {
			fmt.Fprintln(os.Stdout, stdoutScanner.Text())
			c.processStdout(stdoutScanner.Bytes())
		}
	}()
	go func() {
		defer wg.Done()
		for stderrScanner.Scan() {
			fmt.Fprintln(os.Stderr, stderrScanner.Text())
			c.processStderr(stderrScanner.Bytes())
		}
	}()

	runFinished := make(chan bool, 1)
	timeout := make(<-chan time.Time, 1)
	timenotify := make(<-chan time.Time, 1)
	signals := make(chan os.Signal, 1)
	profileMemory := make(<-chan time.Time, 1)
	signal.Notify(signals, os.Interrupt, os.Kill)

	if c.Config.KillTimeout > 0 {
		timeout = time.After(c.Config.KillTimeout)
	}
	if c.Config.NotifyTimeout > 0 {
		timenotify = time.After(c.Config.NotifyTimeout)
	}
	if runtime.GOOS == "linux" {
		switch c.Config.Daemon {
		case true:
			profileMemory = time.Tick(30 * time.Second)
		default:
			profileMemory = time.Tick(1 * time.Second)
		}
	}

	go func() {
		wg.Wait()
		cmd.Wait()
		runFinished <- true
	}()

	for {
		select {
		case <-runFinished:
			return c.handler.Finished(c, cmd)
		case sig := <-signals:
			return c.handler.Signal(c, cmd, sig)
		case <-timeout:
			return c.handler.Timeout(c, cmd)
		case <-timenotify:
			c.handler.TimeWarning(c)
		case <-profileMemory:
			if err := c.handler.CheckMemory(c, cmd); err != nil {
				return c.handler.KillOnHighMemory(c, cmd)
			}
		}
	}
}

// SendReport will send a report based on the current run status
// of the command.  This is safe to call in a go routine to send
// in the background.  It will attempt to send a report for 1hr
// using exponential backoff if there is a problem.
func (c *Command) SendReport(reason proto.ReportReason) {
	c.mutex.Lock()
	c.report.Create(c, reason)
	c.mutex.Unlock()

	result := make(chan error, 1)
	cancel := make(chan bool, 1)
	timeout := time.After(1 * time.Hour)

	go c.report.Send(result, cancel)

	select {
	case err := <-result:
		fmt.Println(err)
	case <-timeout:
		cancel <- true
	}
	close(result)
	close(cancel)
}

// checkRule finds a regular expression match to a line from either Stdout or Stderr.
func checkRule(line []byte, rules []rule) []RuleMatch {
	var matches []RuleMatch
	for _, rule := range rules {
		var text []byte
		switch {
		case len(rule.Field) > 0:
			text = extractTextFromJSON(line, rule.Field)
		default:
			text = line
		}

		found := rule.Regex.FindAllIndex(text, -1)
		if found != nil {
			matches = append(matches, RuleMatch{
				Time:  time.Now(),
				Line:  string(line),
				Index: found,
			})
		}
	}
	return matches
}

func extractTextFromJSON(raw []byte, field string) []byte {
	fieldPath := strings.Split(field, ".")
	switch {
	case len(fieldPath) > 1:
		res := make(map[string]json.RawMessage)
		if err := json.Unmarshal(raw, &res); err != nil {
			return []byte{}
		}
		return extractTextFromJSON(res[fieldPath[0]], strings.Join(fieldPath[1:], "."))
	default:
		res := make(map[string]interface{})
		if err := json.Unmarshal(raw, &res); err != nil {
			return []byte{}
		}
		value := res[field]

		switch value.(type) {
		case string:
			return []byte(value.(string))
		case float32:
			return []byte(fmt.Sprintf("%f", value.(float32)))
		case float64:
			return []byte(fmt.Sprintf("%f", value.(float64)))
		case bool:
			return []byte(fmt.Sprintf("%v", value.(bool)))
		case []interface{}:
			var out []string
			vals, ok := value.([]interface{})
			if !ok {
				return []byte{}
			}
			for _, val := range vals {
				switch val.(type) {
				case string:
					out = append(out, val.(string))
				case float32:
					out = append(out, fmt.Sprintf("%f", value.(float32)))
				case float64:
					out = append(out, fmt.Sprintf("%f", value.(float64)))
				case bool:
					out = append(out, fmt.Sprintf("%v", val.(bool)))
				default:
				}
			}
			return []byte(strings.Join(out, "\n"))
		default:
			return []byte{}
		}
	}
}

func (c *Command) processStdout(line []byte) {
	matches := checkRule(line, c.Config.Rules)
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.RuleMatches = append(c.RuleMatches, matches...)
	history := len(c.Stdout)
	switch {
	case history >= c.Config.StdoutHistory:
		c.Stdout = append(c.Stdout[2:], string(line))
	default:
		c.Stdout = append(c.Stdout, string(line))
	}
	return
}

func (c *Command) processStderr(line []byte) {
	matches := checkRule(line, c.Config.Rules)
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.RuleMatches = append(c.RuleMatches, matches...)
	history := len(c.Stderr)
	switch {
	case history >= c.Config.StderrHistory:
		c.Stderr = append(c.Stderr[2:], string(line))
	default:
		c.Stderr = append(c.Stderr, string(line))
	}
	return
}

func wrapComplexCommand(shell string, args []string) ([]string, func() error, error) {
	r := regexp.MustCompile("(&&|\x7C|<|>)")

	var match []byte
	for _, arg := range args {
		match = r.Find([]byte(arg))
		if match != nil {
			break
		}
	}

	switch match {
	case nil:
		return args, nil, nil
	default:
		wd, err := os.Getwd()
		if err != nil {
			return args, nil, err
		}
		f, err := ioutil.TempFile(wd, "xrayo")
		if err != nil {
			return args, nil, err
		}
		if _, err := f.Write([]byte(strings.Join(args, " "))); err != nil {
			return args, nil, err
		}
		if err := f.Chmod(os.ModePerm); err != nil {
			return args, nil, err
		}
		if err := f.Close(); err != nil {
			return args, nil, err
		}
		return []string{shell, f.Name()}, func() error { return os.Remove(f.Name()) }, nil
	}
}

func (c *Command) Cleanup() (errs []error) {
	for _, f := range c.cleanup {
		if err := f(); err != nil {
			errs = append(errs, err)
		}
	}
	return
}
