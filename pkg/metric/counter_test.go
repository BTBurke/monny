package metric

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestCounter(t *testing.T) {
	tt := []struct {
		name   string
		values []uint
		expect int
	}{
		{name: "positive", values: []uint{1, 1, 2, 3, 4}, expect: 11},
		{name: "zeros", values: []uint{1, 1, 0, 0, 0}, expect: 2},
	}
	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			c := NewCounter()
			for _, i := range tc.values {
				c.Add(i)
			}
			assert.Equal(t, tc.expect, c.Value())
			c.Reset()
			assert.Equal(t, 0, c.Value())
			c.Add(1)
			assert.Equal(t, 1, c.Value())
		})
	}
}

type f func(*WindowedCounter)

func TestWindowedCounter(t *testing.T) {
	a := func(i uint) func(*WindowedCounter) {
		return func(c *WindowedCounter) {
			c.Add(i)
		}
	}
	s := func(d time.Duration) func(*WindowedCounter) {
		return func(c *WindowedCounter) {
			time.Sleep(d)
		}
	}
	d := func(d time.Duration) func(*WindowedCounter) {
		return func(c *WindowedCounter) {
			c.MaxHistoryDuration = d
		}
	}
	h := func(i int) func(*WindowedCounter) {
		return func(c *WindowedCounter) {
			c.MaxHistory = i
		}
	}
	extract := func(counters []Counter) (i []int) {
		for _, c := range counters {
			i = append(i, c.Value())
		}
		return
	}
	tt := []struct {
		name string
		dur  string
		ops  []f
		expV int
		expH []int
	}{
		{name: "basic", dur: "1s", ops: []f{a(1), a(1), a(1), s(0)}, expV: 3, expH: []int{3}},
		{name: "multiple windows", dur: "100ms", ops: []f{a(1), a(1), s(500 * time.Millisecond), a(2), a(3)}, expV: 5, expH: []int{2, 5}},
		{name: "max history", dur: "100ms", ops: []f{h(1), a(1), a(1), s(500 * time.Millisecond), a(2), a(3)}, expV: 5, expH: []int{5}},
		{name: "max duration history", dur: "100ms", ops: []f{d(200 * time.Millisecond), a(1), a(1), s(500 * time.Millisecond), a(2), a(3)}, expV: 5, expH: []int{5}},
	}
	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			d, _ := time.ParseDuration(tc.dur)
			c := NewWindowedCounter(d)
			for _, op := range tc.ops {
				op(c)
			}
			assert.Equal(t, tc.expV, c.Value())
			assert.Equal(t, tc.expH, extract(c.HistoryInclusive()))
		})
	}
}

func TestConcurrentCounters(t *testing.T) {
	c := NewConcurrentCounter()

	var wg sync.WaitGroup
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			for j := 0; j < 10; j++ {
				c.Add(1)
			}
			wg.Done()
		}()
	}
	wg.Wait()
	assert.Equal(t, 50, c.Value())

	w := NewConcurrentWindowedCounter(1 * time.Second)

	var wg2 sync.WaitGroup
	for i := 0; i < 5; i++ {
		wg2.Add(1)
		go func() {
			for j := 0; j < 10; j++ {
				w.Add(1)
			}
			wg2.Done()
		}()
	}
	wg2.Wait()
	assert.Equal(t, 50, w.Value())
}
