package eventbus

import (
	"context"
	"fmt"
	"sync"
)

// EventType represents the type of event being passed on the bus.  It allows handlers receiving the event to
// to properly unmarshal the data or decide if processing is required
type EventType string

// Event is passed on the event bus to every subscriber on the channel
type Event struct {
	EventType EventType
	Data      interface{}
}

// Topic creates a group of subscribers that only receive events published to that channel
type Topic string

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
}

// New returns a new event bus.  A default topic is created, but subscribers may create other topics
// when they register.
func New() *EventBus {
	return &EventBus{
		subscribers: make(map[Topic][]chan Event),
	}
}

// Subscribe will register a subscriber to 0 or more topics.  If no topic is defined, the subscriber will added to the default channel and receive all
// events published on any channel.  The default channel acts like a multicast channel so events published on other topics
// also are received by default channel subscribers.
//
// The subscriber receives two channels, the first channel will receive events and will be closed when the event bus is shut down.
// Subscribers should detect a closed event channel and //interpret that as a shutdown signal.  When the channel is closed,
// subscribers should wait for any existing go routines to exit and then close the second channel (done channel) to indicate
// that the subscriber has finished all work.
func (e *EventBus) Subscribe(topics ...Topic) (chan Event, chan struct{}) {
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
				e.subscribers[topic] = append(e.subscribers[topic][0:i], e.subscribers[topic][i+1:]...)
			}
		}
	}

	for i, d := range e.done {
		if d == done {
			close(d)
			e.done = append(e.done[0:i], e.done[i+1:]...)
		}
	}
}

// Dispatch will send the event to 0 or more topics.  All events are broadcast to default topic subscribers, even when
// other topics may be specified.
func (e *EventBus) Dispatch(event Event, topics ...Topic) {
	e.mutex.RLock()
	defer e.mutex.RUnlock()

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
				ch <- event
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

	done := make(chan struct{})
	go shutdownNotify(done, append([]chan struct{}{}, e.done...))

	for _, chs := range e.subscribers {
		for _, ch := range chs {
			// close all subscriber channels to signal shutdown
			close(ch)
		}
	}

	select {
	case <-ctx.Done():
		return fmt.Errorf("eventbus: context timeout or cancelled before all subscribers exited")
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
