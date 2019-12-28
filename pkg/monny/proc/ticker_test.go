package proc

import (
	"context"
	"log"
	"sync"
	"testing"
	"time"

	"github.com/BTBurke/monny/pkg/eventbus"
	"github.com/stretchr/testify/assert"
)

func TestTicker(t *testing.T) {
	log.Printf("start")
	d := 100 * time.Millisecond
	evt := eventbus.EventType("test_tick")

	eb := eventbus.New()
	c, finished := eb.Subscribe()

	i := 0
	//start := time.Now()
	var wg sync.WaitGroup
	wg.Add(1)
	go func(ch chan eventbus.Event, count *int) {
		log.Printf("starting receiver")
		for {
			select {
			case e, open := <-ch:
				*count++
				if !open {
					finished()
					wg.Done()
					return
				}
				log.Printf("got: %v\n", e)
				assert.Equal(t, e, eventbus.Event{EventType: evt})
				//assert.WithinDuration(t, start.Add(time.Duration(i*d)*time.Millisecond), start, time.Duration(d/4)*time.Millisecond)
			}
		}
	}(c, &i)
	NewTicker(d, eb, evt)
	wg.Add(1)
	go func(count *int) {
		log.Printf("starting watcher")
		for {
			if *count >= 3 {
				log.Printf("trying shutdown")
				_ = eb.Shutdown(context.Background())
				break
			}
		}
		wg.Done()
	}(&i)
	wg.Wait()

}
