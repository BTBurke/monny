package proc

import (
	"fmt"
	"io"
	"os"
	"os/exec"
)

type logProcOpt struct {
	hist    int
	sources []source
	sinks   []sink
}

type sourceOrSink int

// all possible sources and sinks, pX means the wrapped or piped process channels
// mX indicates the monny process's input/output.  Default would be to process logs from
// pStdout->mStdout, pStderr->mStderr, but can be configured different ways depending on
// how monny is run in relation to the monitored process
const (
	_ sourceOrSink = iota
	pStdout
	pStderr
	pStdin
	mStdout
	mStderr
	mStdin
	logfile
)

// option priority matters for overriding default behavior
type priority int

const (
	history priority = iota
	command
	noStdoutOut
	noStderrOut
	noOutput
	noStdoutIn
	noStderrIn
)

type source struct {
	name sourceOrSink
	q    *Queue
	in   io.Reader
}

type sink struct {
	name sourceOrSink
	out  io.WriteCloser
}

// LogProcessorOption overrides default behavior.  Options are applied in a defined order
// no matter the order in which they are passed to the constructor so that principle of least
// surprise applies.
type LogProcessorOption interface {
	apply(*logProcOpt) error
	priority() priority
}

// WithHistory controls how much of the most recent log source is retained and reported when
// an alert is generated.  It applies by default to all configured log sources (e.g., to Stdout and
// Stderr if both sources are configured)
func WithHistory(hist int) LogProcessorOption {
	return optF{
		f:   func(l *logProcOpt) error { l.hist = hist; return nil },
		pri: history,
	}
}

// options should return an optF struct to apply the option and declare its priority to the constructor
type optF struct {
	f   func(*logProcOpt) error
	pri priority
}

func (o optF) apply(l *logProcOpt) error { return o.f(l) }
func (o optF) priority() priority        { return o.pri }

// WithCommand will configure log sources and sinks based on the defined command.  This will determine which
// log sources should be parsed depending on whether monny wraps a forked process or is run as a piped process
// from an earlier command.  Other options may override default behavior.  Default behavior is to copy sources
// directly to the same sink (Stdout->Stdout, Stderr->Stderr) when wrapping a forked process.  When run via a previous
// piped command, monny will parse logs on Stdin and sink to Stdout.  Note that piped processes lose Stderr information.
// Output sinks can be changed by other options, such as using rotated log files.
func WithCommand(c *exec.Cmd) LogProcessorOption {
	f := func(l *logProcOpt) error {
		switch {
		// empty path indicates monny running as piped final process, stdin should be treated as the source
		// and defaults to a stdout sink unless overridden subsequently by a sink-related option
		case c.Path == "":
			l.sources = append(l.sources, source{
				name: mStdin,
				q:    NewQueue(l.hist),
				in:   os.Stdin,
			})
			l.sinks = append(l.sinks, sink{
				name: mStdout,
				out:  os.Stdout,
			})
		// TODO: must handle the piped process to pass previous command stdin to wrapped command
		// in the command processor
		default:
			outPipe, err := c.StdoutPipe()
			if err != nil {
				return fmt.Errorf("unable to get process stdout: %v", err)
			}
			errPipe, err := c.StderrPipe()
			if err != nil {
				return fmt.Errorf("unable to get process stderr: %v", err)
			}
			l.sources = append(l.sources, source{
				name: pStdout,
				q:    NewQueue(l.hist),
				in:   outPipe,
			}, source{
				name: pStderr,
				q:    NewQueue(l.hist),
				in:   errPipe,
			})
			l.sinks = append(l.sinks, sink{
				name: mStdout,
				out:  os.Stdout,
			}, sink{
				name: mStderr,
				out:  os.Stderr,
			})
		}
		return nil
	}
	return optF{
		f:   f,
		pri: command,
	}
}

// WithNoOutput prevents monny from echoing processed logs to either Stdout
// or Stderr
func WithNoOutput() LogProcessorOption {
	return optF{
		f:   func(l *logProcOpt) error { l.sinks = []sink{}; return nil },
		pri: noOutput,
	}
}

// WithNoStderrOutput prevents monny from echoing processed logs on Stderr
func WithNoStderrOutput() LogProcessorOption {
	return optF{
		f:   func(l *logProcOpt) error { l.sinks = filterSink(l.sinks, mStderr); return nil },
		pri: noStderrOut,
	}
}

// WithNoStdoutOutput prevents monny from echoing processed logs on Stdout
func WithNoStdoutOutput() LogProcessorOption {
	return optF{
		f:   func(l *logProcOpt) error { l.sinks = filterSink(l.sinks, mStdout); return nil },
		pri: noStdoutOut,
	}
}

// WithNoStdoutInput prevents monny from processing logs from your process's Stdout
func WithNoStdoutInput() LogProcessorOption {
	return optF{
		f:   func(l *logProcOpt) error { l.sources = filterSource(l.sources, pStdout); return nil },
		pri: noStdoutIn,
	}
}

// WithNoStderrInput prevents monny from processing logs from your process's Stderr
func WithNoStderrInput() LogProcessorOption {
	return optF{
		f:   func(l *logProcOpt) error { l.sources = filterSource(l.sources, pStderr); return nil },
		pri: noStderrIn,
	}
}

// filter one sink from the default configured sinks
func filterSink(sinks []sink, target sourceOrSink) []sink {
	fSinks := []sink{}
	for _, s := range sinks {
		switch s.name {
		case target:
			continue
		default:
			fSinks = append(fSinks, s)
		}
	}
	return fSinks
}

// filter one source from the default configured sources
func filterSource(sources []source, target sourceOrSink) []source {
	fSources := []source{}
	for _, s := range sources {
		switch s.name {
		case target:
			continue
		default:
			fSources = append(fSources, s)
		}
	}
	return fSources
}
