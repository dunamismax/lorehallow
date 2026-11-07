package wire

import (
	"sync"
	"sync/atomic"
)

// Body holds one tick's encoded entities (AppendEntity chunks), shared
// read-only by every recipient's Snapshot. It is reference counted so the
// buffer can go back to a pool once the last writer has sent it: the
// zone takes one reference while it fans out, and each queued Outgoing
// holds one more.
type Body struct {
	B    []byte
	refs atomic.Int32
}

var bodies = sync.Pool{New: func() any { return &Body{B: make([]byte, 0, 4096)} }}

// NewBody returns an empty Body holding one reference (the caller's).
func NewBody() *Body {
	b := bodies.Get().(*Body)
	b.B = b.B[:0]
	b.refs.Store(1)
	return b
}

// Retain adds a reference. Call it before handing the Body to a recipient.
func (b *Body) Retain() { b.refs.Add(1) }

// Release drops a reference; the last one returns the Body to the pool.
// Never touch b.B after releasing your reference.
func (b *Body) Release() {
	switch n := b.refs.Add(-1); {
	case n == 0:
		if cap(b.B) <= 1<<20 { // don't pool buffers grown by a huge crowd
			bodies.Put(b)
		}
	case n < 0:
		panic("wire: Body released more times than retained")
	}
}

// Outgoing is one queued server message. Either Bytes is a complete,
// immutable Envelope (possibly shared by many recipients), or Body is set
// and the writer assembles a Snapshot from Tick, Ack and Body.
type Outgoing struct {
	Bytes     []byte
	Body      *Body
	Tick, Ack uint32
}

// Size is the encoded size of the message in bytes.
func (o Outgoing) Size() int {
	if o.Body != nil {
		return SnapshotSize(o.Tick, o.Ack, o.Body.B)
	}
	return len(o.Bytes)
}

// AppendTo appends the encoded message to dst.
func (o Outgoing) AppendTo(dst []byte) []byte {
	if o.Body != nil {
		return AppendSnapshot(dst, o.Tick, o.Ack, o.Body.B)
	}
	return append(dst, o.Bytes...)
}

// Release drops the message's reference to its Body, if any. Every
// Outgoing handed to a queue is released exactly once, sent or not.
func (o Outgoing) Release() {
	if o.Body != nil {
		o.Body.Release()
	}
}
