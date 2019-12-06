package eventbus

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestShutdownNotifier(t *testing.T) {
	chs := []chan struct{}{}
	for i := 0; i < 100; i++ {
		chs = append(chs, make(chan struct{}))
	}
	done := make(chan struct{})
	for _, ch := range chs {
		go func(c chan struct{}) {
			time.Sleep(200 * time.Millisecond)
			close(c)
		}(ch)
	}
	shutdownNotify(done, chs)

	assert.Equal(t, struct{}{}, <-done)
}

func TestSubscribe(t *testing.T) {
	contains := func(t Topic, all []Topic) bool {
		result := false
		for _, t1 := range all {
			if t == t1 {
				result = true
			}
		}
		return result
	}

	containsCh := func(c chan Event, all []chan Event) bool {
		result := false
		for _, ch1 := range all {
			if c == ch1 {
				result = true
			}
		}
		return result
	}

	tt := []struct {
		Name     string
		Topics   []Topic
		Expected []Topic
	}{
		{Name: "add default", Topics: []Topic{}, Expected: []Topic{defaultTopic}},
		{Name: "create topic on subscribe", Topics: []Topic{Topic("test")}, Expected: []Topic{Topic("test")}},
		{Name: "multi topic subscribe", Topics: []Topic{Topic("test1"), Topic("test2")}, Expected: []Topic{Topic("test1"), Topic("test2")}},
	}

	for _, tc := range tt {
		t.Run(tc.Name, func(t *testing.T) {
			event := New()
			c, d := event.subscribe(tc.Topics...)
			for topic, chs := range event.subscribers {
				switch {
				case contains(topic, tc.Expected):
					assert.True(t, containsCh(c, chs))
				default:
					assert.False(t, containsCh(c, chs))
				}
			}
			result := false
			for _, d1 := range event.done {
				if d1 == d {
					result = true
				}
			}
			assert.True(t, result)
		})
	}
}

func TestUnsubscribe(t *testing.T) {
	e := New()
	c1, d1 := e.subscribe()
	c2, d2 := e.subscribe()
	c3, d3 := e.subscribe(Topic("test"))
	c4, d4 := e.subscribe(Topic("test"))

	e.Unsubscribe(c1, d1)
	assert.Equal(t, e.subscribers[defaultTopic], []chan Event{c2})
	assert.Equal(t, e.done, []chan struct{}{d2, d3, d4})
	assert.Equal(t, e.subscribers[Topic("test")], []chan Event{c3, c4})
	e.Unsubscribe(c3, d3)
	assert.Equal(t, e.done, []chan struct{}{d2, d4})
	assert.Equal(t, e.subscribers[Topic("test")], []chan Event{c4})
}

func containsT(t Topic, all []Topic) bool {
	result := false
	for _, t1 := range all {
		if t == t1 {
			result = true
		}
	}
	return result
}

func TestDispatch(t *testing.T) {
	const (
		Topic1 Topic = "topic1"
		Topic2       = "topic2"
	)
	receiver := func(c chan Event) func() Event {
		return func() Event {
			select {
			case e := <-c:
				return e
			}
		}
	}
	event := Event{
		EventType: EventType("test"),
	}
	tt := []struct {
		Name        string
		Subscribe   []Topic
		Dispatch    []Topic
		ExpectTopic bool
	}{
		{Name: "default", Dispatch: []Topic{}, ExpectTopic: false},
		{Name: "single topic", Subscribe: []Topic{Topic1}, Dispatch: []Topic{Topic1}, ExpectTopic: true},
		{Name: "exclude topic", Subscribe: []Topic{Topic1, Topic2}, Dispatch: []Topic{defaultTopic}, ExpectTopic: false},
	}

	for _, tc := range tt {
		t.Run(tc.Name, func(t *testing.T) {
			e := New()
			cd, _ := e.subscribe()
			defaultSubscriber := receiver(cd)

			if tc.ExpectTopic {
				c, _ := e.subscribe(tc.Subscribe...)
				topicSubscriber := receiver(c)
				e.Dispatch(event, tc.Dispatch...)
				assert.Equal(t, event, topicSubscriber())
				assert.Equal(t, event, defaultSubscriber())
			} else {
				if len(tc.Subscribe) > 0 {
					c, _ := e.subscribe(tc.Subscribe...)
					topicSubscriber := receiver(c)
					e.Dispatch(event, tc.Dispatch...)
					// give time to dispatch before closing channel to prevent panic caused by
					// unrealistic test
					time.Sleep(500 * time.Millisecond)
					close(c)
					assert.NotEqual(t, event, topicSubscriber())
					assert.Equal(t, event, defaultSubscriber())
				} else {
					e.Dispatch(event, tc.Dispatch...)
					assert.Equal(t, event, defaultSubscriber())
				}
			}
		})
	}
}

func TestShutdown(t *testing.T) {
	receiver := func(c chan Event, sd ShutdownFunc) {
		select {
		case _, ok := <-c:
			if !ok {
				defer sd()
				time.Sleep(200 * time.Millisecond)
				return
			}
		}
	}

	tt := []struct {
		Name      string
		Timeout   time.Duration
		ExpectErr bool
	}{
		{Name: "no cancel", Timeout: 5 * time.Second, ExpectErr: false},
		{Name: "cancel", Timeout: 100 * time.Millisecond, ExpectErr: true},
	}

	for _, tc := range tt {
		t.Run(tc.Name, func(t *testing.T) {
			e := New()
			for i := 0; i < 100; i++ {
				c, sd := e.Subscribe()
				go receiver(c, sd)
			}
			time.Sleep(500 * time.Millisecond)
			ctx, cancel := context.WithTimeout(context.Background(), tc.Timeout)
			defer cancel()

			err := e.Shutdown(ctx)

			if tc.ExpectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}

}
