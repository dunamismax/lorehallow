package session

import (
	"sync"

	"github.com/dunamismax/lorehallow/server/internal/wire"
)

// sendQueue is a connection's bounded outgoing queue: a ring buffer
// limited by message count and by bytes. The zone pushes without ever
// blocking; the writer goroutine takes everything queued at once.
//
// A push that would exceed either limit fails: the client is reading too
// slowly, and the session evicts it. The queue never grows past its
// limits, so a stalled client costs a fixed amount of memory.
type sendQueue struct {
	mu       sync.Mutex
	ring     []wire.Outgoing
	head, n  int
	bytes    int
	maxBytes int
	closed   bool
	notify   chan struct{} // capacity 1: "something was queued"
}

func newSendQueue(maxMessages, maxBytes int) *sendQueue {
	return &sendQueue{ring: make([]wire.Outgoing, maxMessages), maxBytes: maxBytes, notify: make(chan struct{}, 1)}
}

// push queues o, taking ownership of it. It returns false, and releases
// o, if the queue is full or closed.
func (q *sendQueue) push(o wire.Outgoing) bool {
	size := o.Size()
	q.mu.Lock()
	if q.closed || q.n == len(q.ring) || q.bytes+size > q.maxBytes {
		closed := q.closed
		q.mu.Unlock()
		o.Release()
		return closed // a closed queue isn't "full"; the client is already going
	}
	q.ring[(q.head+q.n)%len(q.ring)] = o
	q.n++
	q.bytes += size
	q.mu.Unlock()
	select {
	case q.notify <- struct{}{}:
	default:
	}
	return true
}

// takeAll moves every queued message into dst and returns it.
func (q *sendQueue) takeAll(dst []wire.Outgoing) []wire.Outgoing {
	q.mu.Lock()
	defer q.mu.Unlock()
	for q.n > 0 {
		dst = append(dst, q.ring[q.head])
		q.ring[q.head] = wire.Outgoing{}
		q.head = (q.head + 1) % len(q.ring)
		q.n--
	}
	q.bytes = 0
	return dst
}

// close drops and releases everything queued; later pushes are refused.
func (q *sendQueue) close() {
	q.mu.Lock()
	q.closed = true
	var drop []wire.Outgoing
	for q.n > 0 {
		drop = append(drop, q.ring[q.head])
		q.ring[q.head] = wire.Outgoing{}
		q.head = (q.head + 1) % len(q.ring)
		q.n--
	}
	q.bytes = 0
	q.mu.Unlock()
	for _, o := range drop {
		o.Release()
	}
}

// depth returns the queued message count and bytes.
func (q *sendQueue) depth() (int, int) {
	q.mu.Lock()
	defer q.mu.Unlock()
	return q.n, q.bytes
}
