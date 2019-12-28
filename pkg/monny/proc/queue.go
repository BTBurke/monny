package proc

import (
	"sync"
)

// Queue is a FIFO string queue used principally by the log parser to maintain a limited
// log history
type Queue struct {
	q        []string
	capacity int
	mu       sync.Mutex
}

// NewQueue returns a new FIFO string queue with capacity cap.  Capacity is not fixed as subsequent
// calls to add without pop will grow the size of the queue.  Use Enqueue to maintain a fixed capacity
// queue.
func NewQueue(capacity int) *Queue {
	q := make([]string, 0, capacity+1)
	return &Queue{q: q, capacity: capacity}
}

func (q *Queue) add(s string) {
	q.q = append(q.q, s)
}

func (q *Queue) pop() string {
	q.q = q.q[1:]
	return q.q[0]
}

// Add puts the string in the queue, popping the head if the queue is already filled to capacity
func (q *Queue) Add(s string) {
	q.mu.Lock()
	defer q.mu.Unlock()

	switch {
	case len(q.q) < q.capacity:
		q.add(s)
	default:
		_ = q.pop()
		q.add(s)
	}
}

// Copy will lock the queue from further writes and copy the current queue into a new slice.  The new slice length
// will be less than or equal to the initial capacity if the queue is not completely full.
func (q *Queue) Copy() []string {
	q.mu.Lock()
	defer q.mu.Unlock()

	s := make([]string, len(q.q))
	copy(s, q.q)

	return s
}

// Clear will discard everything in the queue and initialize a new queue with the same capacity
func (q *Queue) Clear() {
	q.mu.Lock()
	defer q.mu.Unlock()

	q.q = make([]string, 0, q.capacity+1)
}
