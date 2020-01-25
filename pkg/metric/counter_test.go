package metric

import (
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
