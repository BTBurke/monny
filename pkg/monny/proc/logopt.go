package proc

import (
	"io"
	"os/exec"
)

type logProcOpt struct {
	hist    int
	sources []source
	sinks   []sink
}

type sourceOrSink int

const (
	_ sourceOrSink = iota
	stdout
	stderr
	stdin
	logfile
)

type source struct {
	name sourceOrSink
	q    *Queue
	in   io.Reader
}

type sink struct {
	name sourceOrSink
	out  []io.WriteCloser
}

type LogProcessorOption interface {
	apply(*logProcOpt)
	priority() int
}

func WithHistory(hist int) LogProcessorOption {
	return optF{
		f:   func(l *logProcOpt) { l.hist = hist },
		pri: 0,
	}
}

type optF struct {
	f   func(*logProcOpt)
	pri int
}

func (o optF) apply(l *logProcOpt) { o.f(l) }
func (o optF) priority() int       { return o.pri }

func WithCommand(c *exec.Cmd) LogProcessorOption {
	f := func(l *logProcOpt) {
		// empty path indicates monny running as piped final process, stdin pipe should be treated as the source
		// and defaults to a stdout sink unless overridden subsequently by a sink-related option
	}
	return optF{
		f:   f,
		pri: 0,
	}
}
