package monitor

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/BTBurke/wtf/constants"
)

type Command struct {
	Config        Config
	UserCommand   []string
	Stdout        []string
	Stderr        []string
	Success       bool
	AlertMatches  []AlertMatch
	Killed        bool
	KillReason    constants.KillReason
	Created       []File
	MaxMemory     uint64
	ReportReason  constants.ReportReason
	Start         time.Time
	Finish        time.Time
	Duration      time.Duration
	ExitCode      int
	ExitCodeValid bool
	Errors        []string

	mutex         sync.Mutex
	memWarnSent   bool
	alertLastSent time.Time
}

type File struct {
	Path string
	Size int64
	Time time.Time
}

type AlertMatch struct {
	Time time.Time
	Line string
}

func New(usercmd []string, id string, options ...ConfigOption) (*Command, []error) {
	cfg, err := newConfig(id, options...)
	if err != nil {
		return nil, err
	}
	return &Command{
		Config:      cfg,
		UserCommand: usercmd,
	}, nil
}

func (c *Command) Exec() error {
	fmt.Println("Starting...")
	var cmd *exec.Cmd
	switch len(c.UserCommand) {
	case 1:
		cmd = exec.Command(c.UserCommand[0])
	default:
		cmd = exec.Command(c.UserCommand[0], c.UserCommand[1:]...)
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

	var wg sync.WaitGroup
	go func() {
		wg.Add(1)
		defer wg.Done()
		for stdoutScanner.Scan() {
			fmt.Fprintln(os.Stdout, stdoutScanner.Text())
			c.processStdout(stdoutScanner.Bytes())
		}
	}()
	go func() {
		wg.Add(1)
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

	c.Start = time.Now()
	if err := cmd.Start(); err != nil {
		return err
	}
	go func() {
		if err := cmd.Wait(); err != nil {
			wg.Wait()
			c.Finish = time.Now()
			c.Duration = c.Finish.Sub(c.Start)
			runFinished <- true
		}
		wg.Wait()
		c.Finish = time.Now()
		c.Duration = c.Finish.Sub(c.Start)
		runFinished <- true
	}()

	for {
		select {
		case <-runFinished:
			return handleFinished(c, cmd)
		case sig := <-signals:
			return handleSignal(c, cmd, sig)
		case <-timeout:
			return handleTimeout(c, cmd)
		case <-timenotify:
			handleTimeWarning(c)
		case <-profileMemory:
			if err := checkMemory(c, cmd.Process.Pid); err != nil {
				return killOnHighMemory(c, cmd)
			}
		}
	}
}

// checkAlert finds a regular expression match to a line from either Stdout or Stderr.
func checkAlert(line []byte, alerts []alert) []AlertMatch {
	var matches []AlertMatch
	for _, alert := range alerts {
		var text []byte
		switch {
		case len(alert.Field) > 0:
			text = extractTextFromJSON(line, alert.Field)
		default:
			text = line
		}

		found := alert.Regex.Find(text)
		if found != nil {
			matches = append(matches, AlertMatch{
				Time: time.Now(),
				Line: string(line),
			})
			// TODO: in the alert rate case, should save intermediate results, then clear
			// once the notification is sent
			// TODO: for daemon case, send notification only once every 30 mins, saving intermediate
			// results, then clear
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
		case int:
			return []byte(strconv.Itoa(value.(int)))
		default:
			return []byte{}
		}
	}
}

func (c *Command) processStdout(line []byte) {
	matches := checkAlert(line, c.Config.Alerts)
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.AlertMatches = append(c.AlertMatches, matches...)
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
	matches := checkAlert(line, c.Config.Alerts)
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.AlertMatches = append(c.AlertMatches, matches...)
	history := len(c.Stderr)
	switch {
	case history >= c.Config.StderrHistory:
		c.Stderr = append(c.Stderr[2:], string(line))
	default:
		c.Stderr = append(c.Stderr, string(line))
	}
	return
}
