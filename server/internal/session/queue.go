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
// limits, so a stalled client costs a bounded amount of memory.
//
// The ring starts small and doubles up to maxMessages only when a burst
// needs it, so a quiet connection holds a few hundred bytes of queue, not
// the worst case (measured: preallocating 256 slots was ~10 KiB per idle
// connection).
type sendQueue struct {
	mu       sync.Mutex
	ring     []wire.Outgoing
	head, n  int
	bytes    int
	maxMsgs  int
	maxBytes int
	closed   bool
	notify   chan struct{} // capacity 1: "something was queued"
}

const initialQueueSlots = 8

func newSendQueue(maxMessages, maxBytes int) *sendQueue {
	return &sendQueue{
		ring:    make([]wire.Outgoing, min(initialQueueSlots, maxMessages)),
		maxMsgs: maxMessages, maxBytes: maxBytes, notify: make(chan struct{}, 1),
	}
}

// grow doubles the ring (up to maxMsgs), keeping the queued order.
func (q *sendQueue) grow() {
	bigger := make([]wire.Outgoing, min(len(q.ring)*2, q.maxMsgs))
	for i := range q.n {
		bigger[i] = q.ring[(q.head+i)%len(q.ring)]
	}
	q.ring, q.head = bigger, 0
}

// push queues o, taking ownership of it. It returns false, and releases
// o, if the queue is full or closed.
func (q *sendQueue) push(o wire.Outgoing) bool {
	size := o.Size()
	q.mu.Lock()
	if q.n == len(q.ring) && len(q.ring) < q.maxMsgs && !q.closed {
		q.grow()
	}
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
	if len(q.ring) > 4*initialQueueSlots {
		q.ring, q.head = make([]wire.Outgoing, initialQueueSlots), 0 // shrink back after a burst
	}
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
