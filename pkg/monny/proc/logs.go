package proc

import (
	"sort"
	"sync"

	"github.com/BTBurke/monny/pkg/eventbus"
)

const (
	LogLine  = eventbus.EventType("log_line")
	LogTopic = eventbus.Topic("log_topic")
)

type LogProcessor struct {
	eb *eventbus.EventBus
	mu sync.Mutex

	sources []source
	sinks   []sink
}

func NewLogProcessor(eb *eventbus.EventBus, options ...LogProcessorOption) (*LogProcessor, error) {
	opt := &logProcOpt{
		hist: 30,
	}
	sort.Slice(options, func(i, j int) bool { return options[i].priority() < options[j].priority() })
	for _, f := range options {
		f.apply(opt)
	}

	return nil, nil
}
