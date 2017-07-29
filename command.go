//go:generate stringer -type=KillReason,ReportReason -output const_string.go
package wtf

import (
	"sync"
	"time"
)

type KillReason int

const (
	Timeout KillReason = iota
	Memory
	Signal
)

type ReportReason int

const (
	Success ReportReason = iota
	Failure
	AlertRate
	Killed
)

type Command struct {
	Config       Config
	UserCommand  string
	Stdout       []string
	Stderr       []string
	Success      bool
	GrepMatches  []GrepMatch
	Duration     time.Duration
	Killed       bool
	KillReason   KillReason
	Created      []File
	MaxMemory    uint64
	ReportReason ReportReason

	mutex sync.Mutex
}

type File struct {
	Path string
	Size string
	Time time.Time
}

type GrepMatch struct {
	Time time.Time
	Line string
}

func NewCommand(usercmd string, cfg Config) *Command {
	return &Command{
		Config:      cfg,
		UserCommand: usercmd,
	}
}

func (c *Command) Exec() error {
	return nil
}

// checkAlert finds a regular expression match to a line from either Stdout or Stderr.
func (c *Command) checkAlert(line []byte) {
	return

}

func (c *Command) checkMemory(pid int) {
	return
}

func (c *Command) processStdout(line []byte) {
	return
}

func (c *Command) processStderr(line []byte) {
	return
}
