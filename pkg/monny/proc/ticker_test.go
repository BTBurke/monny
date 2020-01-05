package proc

import (
	"context"
	"log"
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
		log.Printf("starting receiver")
		for e := range ch {
			log.Printf("received %v\n", e)
			*count++
			assert.Equal(t, e, eventbus.Event{EventType: evt})
		}
		finished()
	}(c, &i)
	NewTicker(d, eb, evt)
	time.Sleep(1 * time.Second)
	_ = eb.Shutdown(context.Background())
	assert.NotZero(t, i)
}
