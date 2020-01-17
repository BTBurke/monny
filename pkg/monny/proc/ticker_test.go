package proc

import (
	"context"
	"testing"
	"time"

	"github.com/BTBurke/monny/pkg/eventbus"
	"github.com/stretchr/testify/assert"
)

func TestTicker(t *testing.T) {
	d := 100 * time.Millisecond
	evt := eventbus.EventType("test_tick")

	eb := eventbus.New()
	c, finished := eb.Subscribe()

	i := 0
	go func(ch chan eventbus.Event, count *int) {
		for e := range ch {
			*count++
			assert.Equal(t, e, eventbus.Event{EventType: evt})
		}
		finished()
	}(c, &i)
	NewTicker(d, eb, evt)
	time.Sleep(1 * time.Second)
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()
	_ = eb.Shutdown(ctx)
	assert.NotZero(t, i)
}
