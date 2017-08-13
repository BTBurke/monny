package monitor

import (
	"fmt"
	"os"
	"os/exec"
	"syscall"
	"time"

	"github.com/BTBurke/wtf/proto"
)

// ProcessHandlers is an interface for methods called based on the current
// status of the process
type ProcessHandlers interface {
	Finished(c *Command, cmd *exec.Cmd) error
	Signal(c *Command, cmd *exec.Cmd, sig os.Signal) error
	Timeout(c *Command, cmd *exec.Cmd) error
	TimeWarning(c *Command) error
	CheckMemory(c *Command, cmd *exec.Cmd) error
	KillOnHighMemory(c *Command, cmd *exec.Cmd) error
}

type handler struct{}

// Finished is called when the process ends and determines whether the process completed successfully.
// It also checks that any artifacts expected to be created exist.
func (h handler) Finished(c *Command, cmd *exec.Cmd) error {
	c.mutex.Lock()
	c.Finish = time.Now()
	c.Duration = c.Finish.Sub(c.Start)
	c.mutex.Unlock()

	switch cmd.ProcessState.Success() {
	case true:
		c.mutex.Lock()
		c.Success = true
		c.ExitCodeValid = true
		c.ReportReason = proto.Success
		c.mutex.Unlock()
		go c.report.Send(c, proto.Success)
	default:
		sysinfo, ok := cmd.ProcessState.Sys().(syscall.WaitStatus)
		c.mutex.Lock()
		if ok {
			c.ExitCode = int32(sysinfo.ExitStatus())
			c.ExitCodeValid = true
		}
		c.ReportReason = proto.Failure
		c.Success = false
		c.mutex.Unlock()
		go c.report.Send(c, proto.Failure)
	}
	handleFileCreation(c)

	// fmt.Printf("\n\nProcess finished, Received:\nStdout: %d lines\nStderr: %d lines\nDuration: %s\nMax Memory: %d\nReason: %s\n",
	// 	len(c.Stdout),
	// 	len(c.Stderr),
	// 	c.Duration.String(),
	// 	c.MaxMemory,
	// 	c.ReportReason)
	// for _, match := range c.RuleMatches {
	// 	fmt.Printf("Match: %s\n", match.Line)
	// }
	// for _, e := range c.Messages {
	// 	fmt.Printf("Error: %s\n", e)
	// }
	return nil
}

func (h handler) Signal(c *Command, cmd *exec.Cmd, sig os.Signal) error {
	c.mutex.Lock()
	c.Finish = time.Now()
	c.Duration = c.Finish.Sub(c.Start)
	c.Killed = true
	c.KillReason = proto.Signal
	c.ReportReason = proto.Killed
	c.mutex.Unlock()

	go c.report.Send(c, proto.Killed)
	if err := cmd.Process.Signal(sig); err != nil {
		return err
	}
	fmt.Printf("\n\nProcess received signal: %s\n", sig.String())
	return nil
}

func (h handler) Timeout(c *Command, cmd *exec.Cmd) error {
	c.mutex.Lock()
	c.Killed = true
	c.KillReason = proto.Timeout
	c.Finish = time.Now()
	c.Duration = c.Start.Sub(c.Finish)
	c.ReportReason = proto.Killed
	c.mutex.Unlock()

	go c.report.Send(c, proto.Killed)
	if err := cmd.Process.Signal(os.Kill); err != nil {
		return err
	}
	fmt.Printf("\n\nProcess timeout\n")
	return nil
}

func (h handler) TimeWarning(c *Command) error {
	if c.timeWarnSent {
		return nil
	}
	c.mutex.Lock()
	c.ReportReason = proto.TimeWarning
	c.timeWarnSent = true
	c.mutex.Unlock()

	go c.report.Send(c, proto.TimeWarning)

	return nil
}

func (h handler) CheckMemory(c *Command, cmd *exec.Cmd) error {
	mem := calculateMemory(cmd.Process.Pid)
	if mem > c.MaxMemory {
		c.mutex.Lock()
		c.MaxMemory = mem
		c.mutex.Unlock()
	}
	if c.Config.MemoryWarn > 0 && mem >= c.Config.MemoryWarn {
		if !c.memWarnSent {
			c.mutex.Lock()
			c.ReportReason = proto.MemoryWarning
			c.memWarnSent = true
			c.mutex.Unlock()

			go c.report.Send(c, proto.MemoryWarning)
		}
	}
	if c.Config.MemoryKill > 0 && mem >= c.Config.MemoryKill {
		return fmt.Errorf("high memory kill")
	}
	return nil
}

func (h handler) KillOnHighMemory(c *Command, cmd *exec.Cmd) error {
	c.mutex.Lock()
	c.Killed = true
	c.KillReason = proto.Memory
	c.Finish = time.Now()
	c.Duration = c.Finish.Sub(c.Start)
	c.ReportReason = proto.Killed
	c.mutex.Unlock()

	go c.report.Send(c, proto.Killed)
	if err := cmd.Process.Signal(os.Kill); err != nil {
		return err
	}
	return nil
}

func handleFileCreation(c *Command) {
	for _, f := range c.Config.Creates {
		finfo, err := os.Stat(f)
		switch {
		case os.IsNotExist(err):
			c.mutex.Lock()
			c.Success = false
			c.Messages = append(c.Messages, fmt.Sprintf("file not created: %s", f))
			c.ReportReason = proto.FileNotCreated
			c.mutex.Unlock()
			go c.report.Send(c, proto.FileNotCreated)
		case err == nil:
			c.mutex.Lock()
			c.Created = append(c.Created, File{
				Path: finfo.Name(),
				Time: finfo.ModTime(),
				Size: finfo.Size(),
			})
			c.mutex.Unlock()
		default:
			continue
		}
	}
	return
}
