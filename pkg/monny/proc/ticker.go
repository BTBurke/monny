package proc

import (
	"time"

	"github.com/BTBurke/monny/pkg/eventbus"
)

// NewTicker returns a ticking handler that will emit a message with the passed EventType every specified duration.
// This is the underlying mechanism to tell other handlers when to sample metrics or send accumulated metrics.
func NewTicker(d time.Duration, e *eventbus.EventBus, evt eventbus.EventType) {
	if d == 0 {
		return
	}
	ch, finished := e.Subscribe()
	tick := time.NewTicker(d)

	go func(c chan eventbus.Event, t *time.Ticker, evt eventbus.EventType) {
		for {
			select {
			case _, open := <-c:
				if !open {
					t.Stop()
					finished()
				}

			case <-t.C:
				evt, _ := eventbus.NewEvent(evt, nil)
				e.Dispatch(evt)
			}
		}
	}(ch, tick, evt)
}
