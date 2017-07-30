package monitor

import (
	"fmt"
	"os"
	"os/exec"
	"syscall"
	"time"

	"github.com/BTBurke/wtf/constants"
)

func handleFinished(c *Command, cmd *exec.Cmd) error {
	switch cmd.ProcessState.Success() {
	case true:
		c.Success = true
		c.ReportReason = constants.Success
		c.ExitCodeValid = true
	default:
		sysinfo, ok := cmd.ProcessState.Sys().(syscall.WaitStatus)
		if ok {
			c.ExitCode = sysinfo.ExitStatus()
			c.ExitCodeValid = true
		}
		c.Success = false
		c.ReportReason = constants.Failure
	}
	handleFileCreation(c)

	fmt.Printf("\n\nProcess finished, Received:\nStdout: %d lines\nStderr: %d lines\nDuration: %s\nMax Memory: %d\nReason: %s\n",
		len(c.Stdout),
		len(c.Stderr),
		c.Duration.String(),
		c.MaxMemory,
		c.ReportReason)
	for _, match := range c.AlertMatches {
		fmt.Printf("Match: %s\n", match.Line)
	}
	for _, e := range c.Errors {
		fmt.Printf("Error: %s\n", e)
	}
	return nil
}

func handleSignal(c *Command, cmd *exec.Cmd, sig os.Signal) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	c.Finish = time.Now()
	c.Duration = c.Start.Sub(c.Finish)
	c.Killed = true
	c.KillReason = constants.Signal
	c.ReportReason = constants.Killed
	if err := cmd.Process.Signal(sig); err != nil {
		return err
	}
	fmt.Printf("\n\nProcess received signal: %s\n", sig.String())
	return nil
}

func handleTimeout(c *Command, cmd *exec.Cmd) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	c.Killed = true
	c.KillReason = constants.Timeout
	c.Finish = time.Now()
	c.Duration = c.Start.Sub(c.Finish)
	c.ReportReason = constants.Killed
	if err := cmd.Process.Signal(os.Kill); err != nil {
		return err
	}
	fmt.Printf("\n\nProcess timeout\n")
	return nil
}

func handleTimeWarning(c *Command) {
	fmt.Println("TODO: send time warning")
	return
}

func checkMemory(c *Command, pid int) error {
	mem := calculateMemory(pid)
	if mem > c.MaxMemory {
		c.mutex.Lock()
		defer c.mutex.Unlock()
		c.MaxMemory = mem
	}
	if c.Config.MemoryWarn > 0 && mem >= c.Config.MemoryWarn {
		fmt.Println("Memory warning exceeded")
		if !c.memWarnSent {
			c.mutex.Lock()
			defer c.mutex.Unlock()
			c.memWarnSent = true
			fmt.Println("TODO: send the warning")
		}
	}
	if c.Config.MemoryKill > 0 && mem >= c.Config.MemoryKill {
		fmt.Println("Memory kill")
		return fmt.Errorf("kill on high memory")
	}
	return nil
}

func killOnHighMemory(c *Command, cmd *exec.Cmd) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	c.Killed = true
	c.KillReason = constants.Memory
	c.Finish = time.Now()
	c.Duration = c.Start.Sub(c.Finish)
	if err := cmd.Process.Kill(); err != nil {
		fmt.Printf("Kill error: %v", err)
		return err
	}
	return nil
}

func handleFileCreation(c *Command) {
	for _, f := range c.Config.Creates {
		finfo, err := os.Stat(f)
		switch {
		case os.IsNotExist(err):
			c.ReportReason = constants.FileNotCreated
			c.Success = false
			c.Errors = append(c.Errors, fmt.Sprintf("file not created: %s", f))
		case err == nil:
			c.Created = append(c.Created, File{
				Path: finfo.Name(),
				Time: finfo.ModTime(),
				Size: finfo.Size(),
			})
		default:
			continue
		}
	}
	return
}
