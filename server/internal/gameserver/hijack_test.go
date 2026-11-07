package gameserver

import (
	"bufio"
	"bytes"
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/coder/websocket"
)

// fakeHijacker returns a pipe with net/http-sized buffers, some of them
// possibly holding data already.
type fakeHijacker struct {
	http.ResponseWriter
	conn      net.Conn
	readAhead []byte // bytes the client sent before the upgrade finished
	unwritten []byte
}

func (f *fakeHijacker) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	r := bufio.NewReaderSize(bytes.NewReader(f.readAhead), 4096)
	if len(f.readAhead) > 0 {
		_, _ = r.Peek(len(f.readAhead))
	}
	w := bufio.NewWriterSize(f.conn, 4096)
	_, _ = w.Write(f.unwritten)
	return f.conn, bufio.NewReadWriter(r, w), nil
}

func TestSmallBuffers(t *testing.T) {
	cases := []struct {
		name                string
		early, unwritten    []byte
		wantRead, wantWrite int
	}{
		{"the usual upgrade gets small buffers", nil, nil, ReadBufferSize, WriteBufferSize},
		{"bytes sent before the 101 stay with the original reader", []byte("early frame"), nil, 4096, WriteBufferSize},
		{"an unflushed writer is kept", nil, []byte("pending"), ReadBufferSize, 4096},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			a, b := net.Pipe()
			defer func() { _ = a.Close(); _ = b.Close() }()
			w := withSmallBuffers(&fakeHijacker{conn: a, readAhead: c.early, unwritten: c.unwritten})
			_, brw, err := w.(http.Hijacker).Hijack()
			if err != nil {
				t.Fatal(err)
			}
			if got := brw.Reader.Size(); got != c.wantRead {
				t.Errorf("read buffer %d, want %d", got, c.wantRead)
			}
			if got := brw.Writer.Size(); got != c.wantWrite {
				t.Errorf("write buffer %d, want %d", got, c.wantWrite)
			}
			if len(c.early) > 0 {
				got, _ := brw.Peek(len(c.early))
				if !bytes.Equal(got, c.early) {
					t.Errorf("early bytes %q, want %q", got, c.early)
				}
			}
		})
	}
}

func TestSmallBuffersPassThroughNonHijackers(t *testing.T) {
	w := httptest.NewRecorder()
	if got := withSmallBuffers(w); got != http.ResponseWriter(w) {
		t.Fatalf("withSmallBuffers wrapped a writer that can't hijack: %T", got)
	}
}

// Messages larger than either buffer go through intact in both
// directions over a real socket.
func TestSmallBuffersCarryLargeMessages(t *testing.T) {
	big := bytes.Repeat([]byte("lorehallow"), 4000) // 40 KB, 20× the write buffer
	in := bytes.Repeat([]byte{7}, 3*ReadBufferSize)
	got := make(chan []byte, 1)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c, err := websocket.Accept(withSmallBuffers(w), r, nil)
		if err != nil {
			t.Error(err)
			return
		}
		defer func() { _ = c.CloseNow() }()
		c.SetReadLimit(int64(len(in)))
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_, msg, err := c.Read(ctx)
		if err != nil {
			t.Error(err)
			return
		}
		got <- msg
		for range 3 {
			if err := c.Write(ctx, websocket.MessageBinary, big); err != nil {
				t.Error(err)
				return
			}
		}
		_ = c.Close(websocket.StatusNormalClosure, "")
	}))
	defer srv.Close()

	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()
	c, _, err := websocket.Dial(ctx, "ws"+srv.URL[len("http"):], nil)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = c.CloseNow() }()
	c.SetReadLimit(int64(len(big)))
	if err := c.Write(ctx, websocket.MessageBinary, in); err != nil {
		t.Fatal(err)
	}
	if msg := <-got; !bytes.Equal(msg, in) {
		t.Fatalf("server read %d bytes, want %d", len(msg), len(in))
	}
	for i := range 3 {
		_, msg, err := c.Read(ctx)
		if err != nil {
			t.Fatalf("message %d: %v", i, err)
		}
		if !bytes.Equal(msg, big) {
			t.Fatalf("message %d: %d bytes, want %d intact", i, len(msg), len(big))
		}
	}
}
