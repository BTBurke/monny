package eventbus

import (
	"context"
	"sync"
)

var _ EventDispatcher = &EventBus{}

// Topic creates a group of subscribers that only receive events published to that channel
type Topic string

// EventDispatcher is an interface for functions that only emit events to the bus
type EventDispatcher interface {
	Dispatch(e Event, t ...Topic)
}

const (
	defaultTopic Topic = "__default__"
)

// EventBus dispatches events to all subcribers on one or more topics.  If no topic is set, a default
// channel is created that dispatches events to every subscriber.  Subscribers can use the EventType to
// filter which events they respond to rather than configuring multiple topics.
type EventBus struct {
	subscribers map[Topic][]chan Event
	done        []chan struct{}
	mutex       sync.RWMutex
	sdStarted   bool
}

// New returns a new event bus.  A default topic is created, but subscribers may create other topics
// when they register.
func New() *EventBus {
	return &EventBus{
		subscribers: make(map[Topic][]chan Event),
	}
}

// ShutdownFunc tells the event bus that this subscriber has finished the shutdown process and it is safe to exit
type ShutdownFunc func()

type doneCloser struct {
	once sync.Once
	d    chan struct{}
}

// atomic close to prevent race condition on shutdown and receive value at same time
func (s *doneCloser) close() {
	s.once.Do(func() { close(s.d) })
}

// Subscribe will register a subscriber to 0 or more topics.  If no topic is defined, the subscriber will added to the default channel and receive all
// events published on any channel.  The default channel acts like a multicast channel so events published on other topics
// also are received by default channel subscribers.
//
// The subscriber receives a channel to receive events and a shutdown function. The event channel will be closed when the event bus is shut down.
// Subscribers should detect a closed event channel and interpret that as a shutdown signal.  When the channel is closed,
// subscribers should wait for any existing go routines to exit and then call the ShutdownFunc to indicate
// that the subscriber has finished all work.  It is safe to defer the ShutdownFunc at the top of your event handler to ensure that
// the event bus knows when your subscriber has exited.
func (e *EventBus) Subscribe(topics ...Topic) (chan Event, ShutdownFunc) {
	c, d := e.subscribe(topics...)
	s := &doneCloser{d: d}
	return c, func() { s.close() }
}

func (e *EventBus) subscribe(topics ...Topic) (chan Event, chan struct{}) {
	e.mutex.Lock()
	defer e.mutex.Unlock()

	c := make(chan Event, 1)
	done := make(chan struct{})
	e.done = append(e.done, done)

	// subscribe to the default topic if no topics defined
	if len(topics) == 0 {
		topics = []Topic{defaultTopic}
	}

	for _, topic := range topics {
		ch, ok := e.subscribers[topic]
		switch {
		case ok:
			e.subscribers[topic] = append(ch, c)
		default:
			e.subscribers[topic] = append([]chan Event{}, c)
		}
	}
	return c, done
}

// Unsubscribe removes the subscriber from receiving any more events and closes all its channels
func (e *EventBus) Unsubscribe(c chan Event, done chan struct{}) {
	e.mutex.Lock()
	defer e.mutex.Unlock()

	for topic, chs := range e.subscribers {
		for i, ch := range chs {
			if ch == c {
				close(ch)
				recover()
				e.subscribers[topic] = append(e.subscribers[topic][0:i], e.subscribers[topic][i+1:]...)
			}
		}
	}

	for i, d := range e.done {
		if d == done {
			close(d)
			recover()
			e.done = append(e.done[0:i], e.done[i+1:]...)
		}
	}
}

// Dispatch will send the event to 0 or more topics.  All events are broadcast to default topic subscribers, even when
// other topics may be specified.
func (e *EventBus) Dispatch(event Event, topics ...Topic) {
	e.mutex.RLock()
	defer e.mutex.RUnlock()

	// if shutdown already started prior to this lock but subscribers have not closed yet, return early
	if e.sdStarted {
		return
	}

	// always send to the defaultTopic even if other topics specified
	topics = append(topics, defaultTopic)

	for _, topic := range topics {
		// it no subscribers on the topic, silently drop message.  This is probably the behavior we want since
		// it should be ok to emit events on specialized channels where there may not be subscribers in some cases
		channels, ok := e.subscribers[topic]
		if len(channels) == 0 || !ok {
			continue
		}

		// make a copy of the channels to preserve locking
		chs := append([]chan Event{}, channels...)

		go func(event Event, chs []chan Event) {
			for _, ch := range chs {
				// run in go func so that if channel is closed by subscriber improperly or
				// blocks because channel buffer is full it won't prevent other subscribers
				// from receiving the event

				// this will pessimistically lock the send channel in case the event bus is behind in sending
				// events and shutdown has started.  Events will be silently dropped if shutdown is called and there
				// are still pending events because subscribers are blocking.
				go func(evt Event, c chan Event) {
					e.mutex.RLock()
					defer e.mutex.RUnlock()
					if e.sdStarted {
						return
					}
					defer recover()
					c <- evt
				}(event, ch)
			}
		}(event, chs)
	}
}

// Shutdown will send the shutdown signal to all subscribers and block until they exit.  Best practice is to
// use a context timeout to prevent shutdown from hanging if a go routine cannot finish processing all events in
// a reasonable time.  Shutdown returns an error if it reaches context timeout or cancel to distinguish from a completely
// successful shutdown.
func (e *EventBus) Shutdown(ctx context.Context) error {
	e.mutex.Lock()
	defer e.mutex.Unlock()
	e.sdStarted = true

	done := make(chan struct{})
	go shutdownNotify(done, append([]chan struct{}{}, e.done...))

	for _, chs := range e.subscribers {
		for _, ch := range chs {
			// close all subscriber channels to signal shutdown, recover above in case
			// one of the channels is closed improperly by subscriber which would cause a panic
			close(ch)
			if r := recover(); r != nil {
				continue
			}
		}
	}

	select {
	case <-ctx.Done():
		return ErrShutdownTimeout
	case <-done:
		return nil
	}
}

// shutdownNotify will watch each channel for it to be closed on the subscriber end and sends the notification on the done
// channel.  This should be called on the eventbus list of done channels. Subscribers should detect a closed send channel,
// do cleanup, then close their done channel when all go routines have exited.
func shutdownNotify(done chan struct{}, all []chan struct{}) {
	var wg sync.WaitGroup

	// launch one go routine for each non-nil channel and wait until all return to know
	// that all subscribers have shut down
	for _, ch := range all {
		wg.Add(1)
		go func(c chan struct{}) {
			select {
			case <-c:
				wg.Done()
				return
			}
		}(ch)
	}
	wg.Wait()
	close(done)
}
