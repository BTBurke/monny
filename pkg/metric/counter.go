package metric

import (
	"sync"
	"time"
)

var _ CounterI = &Counter{}
var _ CounterI = &WindowedCounter{}

// CounterI is the basic interface for a counter that returns its current value and adds new observations
type CounterI interface {
	Value() int
	Add(i uint)
	Reset()
}

// Counter is a monotonically increasing counter
type Counter struct {
	start    time.Time
	duration time.Duration
	value    int
}

// Value returns the current value of the counter
func (c *Counter) Value() int {
	return c.value
}

// Add will increase the current count by i
func (c *Counter) Add(i uint) {
	c.value += int(i)
}

// Reset sets the value of the counter to zero
func (c *Counter) Reset() {
	c.value = 0
}

// Start returns the time this counter was started, which may be the time.Time null value for non-windowed counters.  This
// function is only useful when operating on a history of counters returned by the History() function on windowed counters.
func (c Counter) Start() time.Time {
	return c.start
}

// Duration returns the duration of the counter, which will be zero for non-windowed counters. This function is only useful
// when operating on a history of counters returned by the History() function on windowed counters.
func (c Counter) Duration() time.Duration {
	return c.duration
}

// NewCounter returns a new monotonically increasing counter
func NewCounter() *Counter {
	return &Counter{}
}

// WindowedCounter returns a counter that keeps track of counts within a set duration.  A new counter is allocated
// after the duration has elapsed and keeps track of the history of counts in each interval over time.  Note that if
// no observations are added during a window, the history will not keep track of 0 values so there may be gaps in the
// the complete timeline.  Use NewWindowedCounter to initialize a windowed counter.  If you instatiate this as an empty
// struct it will act like a monotonically increasing counter without windowing.
type WindowedCounter struct {
	hist    []Counter
	current *Counter
}

// Value returns the current value of the counter in the most recent window
func (c *WindowedCounter) Value() int {
	now := time.Now().UTC()
	end := c.current.start.Add(c.current.duration)

	// check if the current window has closed, if so return 0
	switch {
	case now.After(end) && c.current.duration >= 0:
		return 0
	default:
		return c.current.Value()
	}
}

// Add will increment the current counter value within the window by i
func (c *WindowedCounter) Add(i uint) {
	now := time.Now().UTC()
	end := c.current.start.Add(c.current.duration)
	switch {
	case now.Before(end) || c.current.duration == 0:
		c.current.Add(i)
	default:
		c.hist = append(c.hist, *c.current)
		c.current = &Counter{start: time.Now().UTC(), duration: c.current.duration}
		c.current.Add(i)
	}
}

// History will return the history of counters not including the current value if the window is still open
func (c *WindowedCounter) History() []Counter {
	now := time.Now().UTC()
	end := c.current.start.Add(c.current.duration)
	switch {
	case now.After(end) || c.current.duration == 0:
		return append(c.hist, *c.current)
	default:
		return c.hist
	}
}

// HistoryInclusive will return the history of all counters, including the current value even if the window
// is still open on the current counter
func (c *WindowedCounter) HistoryInclusive() []Counter {
	return append(c.hist, *c.current)
}

// Reset will clear the current counters history and start a new zero-valued counter with the same window duration
func (c *WindowedCounter) Reset() {
	c.hist = []Counter{}
	c.current = &Counter{start: time.Now().UTC(), duration: c.current.duration}
}

// NewWindowedCounter creates a new windowed counter with a window size of duration
func NewWindowedCounter(duration time.Duration) *WindowedCounter {
	return &WindowedCounter{
		current: &Counter{start: time.Now().UTC(), duration: duration},
	}
}

// ConcurrentCounter is a Counter that is safe for concurrent use
type ConcurrentCounter struct {
	mu sync.RWMutex
	c  *Counter
}

func (c *ConcurrentCounter) Value() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.c.Value()
}

func (c *ConcurrentCounter) Add(i uint) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.c.Add(i)
}

func (c *ConcurrentCounter) Reset() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.c.Reset()
}

func NewConcurrentCounter() *ConcurrentCounter {
	return &ConcurrentCounter{
		c: NewCounter(),
	}
}

// ConcurrentWindowedCounter is a WindowedCounter safe for concurrent use from multiple goroutines
type ConcurrentWindowedCounter struct {
	mu sync.RWMutex
	c  *WindowedCounter
}

func (c *ConcurrentWindowedCounter) Value() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.c.Value()
}

func (c *ConcurrentWindowedCounter) Add(i uint) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.c.Add(i)
}

func (c *ConcurrentWindowedCounter) Reset() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.c.Reset()
}

func (c *ConcurrentWindowedCounter) History() []Counter {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.c.History()
}

func (c *ConcurrentWindowedCounter) HistoryInclusive() []Counter {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.c.HistoryInclusive()

}

func NewConcurrentWindowedCounter(duration time.Duration) *ConcurrentWindowedCounter {
	return &ConcurrentWindowedCounter{
		c: NewWindowedCounter(duration),
	}
}
