package proc

import (
	"bufio"
	"context"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/BTBurke/monny/pkg/eventbus"
)

const (
	//LogLine is the EventType for logs on the LogTopic topic.  Log line processors should
	// listen for LogLine events.
	LogLine  = eventbus.EventType("log_line")
	LogTopic = eventbus.Topic("log_topic")
)

// LogEvent is the payload for the message sent on the bus
type LogEvent struct {
	Timestamp time.Time
	Line      []byte
}

// LogProcessor processes configured log sources and emits processed log lines to configured sinks.  Call Wait()
// to ensure that all log lines have been processed before shutting down.  This processor does not monitor a closed
// event bus as a signal to shut down.  It will continue to run until the scanner reaches EOF.
type LogProcessor struct {
	*logProcOpt

	wg sync.WaitGroup
}

// NewLogProcessor returns a log processor configured to the available options.  Typically it is called
// WithCommand(exec.Cmd) in order to hook into the wrapped process pipes.
func NewLogProcessor(eb *eventbus.EventBus, options ...LogProcessorOption) (*LogProcessor, error) {
	opt := &logProcOpt{
		hist: 30,
	}
	sort.Slice(options, func(i, j int) bool { return options[i].priority() < options[j].priority() })
	for _, f := range options {
		if err := f.apply(opt); err != nil {
			return nil, fmt.Errorf("error creating the log processor: %v", err)
		}
	}

	l := &LogProcessor{opt, sync.WaitGroup{}}
	done := func() { l.wg.Done() }

	for _, s := range opt.sources {
		l.wg.Add(1)

		// some special cases here to maintain pStdout->mStdout and pStderr->mStderr log sinks
		switch s.name {
		case pStdout:
			go startLogEmitter(eb, s, filterSink(opt.sinks, mStderr), done)
		case pStderr:
			go startLogEmitter(eb, s, filterSink(opt.sinks, mStdout), done)
		default:
			go startLogEmitter(eb, s, opt.sinks, done)
		}
	}
	return l, nil
}

// startLogEmitter scans the supplied source and emits each log line (newline delimited) to the LogTopic
// bus for downstream processing.  Lines are then written to the sinks, if any.  Done is called to signal
// to the LogProcessor that the scanner has closed and all logs have been emitted to the bus.
func startLogEmitter(bus eventbus.EventDispatcher, src source, sinks []sink, done func()) {
	if done != nil {
		defer done()
	}
	scanner := bufio.NewScanner(src.in)
	for scanner.Scan() {
		data := scanner.Bytes()
		src.q.Add(string(data))

		payload := LogEvent{
			Timestamp: time.Now().UTC(),
			Line:      data,
		}
		evt, err := eventbus.NewEvent(LogLine, payload)
		if err != nil {
			newError(bus, EventError{fmt.Errorf("unable to construct log event: %v", err)})
		}
		bus.Dispatch(evt, LogTopic)

		for _, s := range sinks {
			if _, err := s.out.Write(append(data, '\n')); err != nil {
				newError(bus, SinkError{fmt.Errorf("error writing to sink %s: %v", src.name, err)})
			}
		}
	}
}

// Wait will wait for all log sources to finish processing.  Context can can
// use a timeout or cancel to set a reasonable time for finishing.  An error is
// returned only if the context is cancelled before all logs are processed.
func (l *LogProcessor) Wait(ctx context.Context) error {
	defer func() {
		for _, s := range l.sinks {
			s.out.Close()
		}
	}()

	wgDone := make(chan struct{}, 1)
	go func() { l.wg.Wait(); close(wgDone) }()

	select {
	case <-wgDone:
		return nil
	case <-ctx.Done():
		return fmt.Errorf("context cancelled on log processor shutdown")
	}
}
