package gameserver

import (
	"bufio"
	"net"
	"net/http"
)

// Transport buffer sizes for game connections (ADR 0010).
//
// net/http hands a hijacked connection its own 4 KiB read and write
// buffers, and coder/websocket keeps both for the life of the
// connection: 8 KiB per player, most of an idle connection's memory.
// Game messages are small. A client message is at most
// wire.MaxClientMessage (512 bytes) plus an 8-byte frame header, so a
// 1 KiB read buffer still holds two of the largest. Server messages are
// usually a Snapshot of a few hundred bytes; bufio writes anything
// larger than its buffer straight to the socket, so a big message costs
// one extra write, not a copy.
const (
	ReadBufferSize  = 1 << 10
	WriteBufferSize = 2 << 10
)

// smallBuffers wraps the handler's ResponseWriter so that Hijack hands
// the WebSocket library smaller buffers than net/http's. The originals
// become garbage once the HTTP handler returns, which is why the session
// runs on its own goroutine (see Handler).
type smallBuffers struct {
	http.ResponseWriter
	hj http.Hijacker
}

func (w smallBuffers) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	conn, brw, err := w.hj.Hijack()
	if err != nil {
		return conn, brw, err
	}
	r := brw.Reader
	if r.Buffered() == 0 {
		// The usual case: a client waits for 101 Switching Protocols
		// before it sends a frame, so nothing is buffered yet. If a client
		// did send early, the original reader keeps those bytes.
		r = bufio.NewReaderSize(conn, ReadBufferSize)
	}
	bw := brw.Writer
	if bw.Buffered() == 0 { // net/http flushes the 101 response before hijacking
		bw = bufio.NewWriterSize(conn, WriteBufferSize)
	}
	return conn, bufio.NewReadWriter(r, bw), nil
}

// withSmallBuffers returns w wrapped, or w itself if it can't hijack.
func withSmallBuffers(w http.ResponseWriter) http.ResponseWriter {
	hj, ok := w.(http.Hijacker)
	if !ok {
		return w
	}
	return smallBuffers{ResponseWriter: w, hj: hj}
}
