package wire

import (
	"bytes"
	"errors"
	"math"
	"math/rand/v2"
	"os"
	"path/filepath"
	"testing"

	"google.golang.org/protobuf/encoding/protowire"
	"google.golang.org/protobuf/proto"

	"github.com/dunamismax/lorehallow/server/internal/pb"
)

func toPB(e Entity) *pb.EntityState {
	return &pb.EntityState{Id: e.ID, X: e.X, Y: e.Y, Facing: e.Facing, Anim: e.Anim}
}

func marshalSnapshot(t testing.TB, tick, ack uint32, ents []Entity) []byte {
	t.Helper()
	snap := &pb.Snapshot{Tick: tick, AckInputSeq: ack}
	for _, e := range ents {
		snap.Entities = append(snap.Entities, toPB(e))
	}
	b, err := Encode(nil, &pb.Envelope{Payload: &pb.Envelope_Snapshot{Snapshot: snap}})
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func assemble(tick, ack uint32, ents []Entity) []byte {
	var body []byte
	for _, e := range ents {
		body = AppendEntity(body, e)
	}
	return AppendSnapshot(nil, tick, ack, body)
}

func randomEntity(r *rand.Rand) Entity {
	pick := func(vals ...int64) int64 { return vals[r.IntN(len(vals))] }
	return Entity{
		ID:     uint32(pick(0, 1, 127, 128, 16383, 16384, math.MaxUint32, int64(r.Uint32()))),
		X:      int32(pick(0, 1, -1, 63, -64, 64, math.MaxInt32, math.MinInt32, int64(int32(r.Uint32())))),
		Y:      int32(pick(0, 1, -1, 512, -512, math.MaxInt32, math.MinInt32, int64(int32(r.Uint32())))),
		Facing: pb.Facing(r.IntN(5)),
		Anim:   pb.Anim(r.IntN(3)),
	}
}

func TestAssemblyMatchesMarshal(t *testing.T) {
	r := rand.New(rand.NewPCG(1, 2))
	cases := [][]Entity{nil, {{}}, {{ID: 1}}}
	for range 500 {
		ents := make([]Entity, r.IntN(20))
		for i := range ents {
			ents[i] = randomEntity(r)
		}
		cases = append(cases, ents)
	}
	for i, ents := range cases {
		tick, ack := r.Uint32()>>uint(r.IntN(33)), r.Uint32()>>uint(r.IntN(33))
		if i < 3 {
			tick, ack = 0, 0
		}
		want := marshalSnapshot(t, tick, ack, ents)
		got := assemble(tick, ack, ents)
		if !bytes.Equal(got, want) {
			t.Fatalf("case %d: assembled\n %x\nwant\n %x", i, got, want)
		}
		var body []byte
		for _, e := range ents {
			body = AppendEntity(body, e)
		}
		if n := SnapshotSize(tick, ack, body); n != len(got) {
			t.Fatalf("case %d: SnapshotSize = %d, len = %d", i, n, len(got))
		}
	}
}

func TestEntityChunksAreSharedAcrossRecipients(t *testing.T) {
	// Encode once: the same entity bytes serve recipients with different
	// headers, and every result decodes to the right Snapshot.
	ents := []Entity{{ID: 1, X: 10, Y: 20, Facing: pb.Facing_FACING_UP, Anim: pb.Anim_ANIM_WALK}, {ID: 2, X: -5}}
	var body []byte
	for _, e := range ents {
		body = AppendEntity(body, e)
	}
	for ack := range uint32(5) {
		msg := AppendSnapshot(nil, 77, ack, body)
		var env pb.Envelope
		if err := DecodeServer(msg, &env); err != nil {
			t.Fatal(err)
		}
		s := env.GetSnapshot()
		if s.GetTick() != 77 || s.GetAckInputSeq() != ack || len(s.GetEntities()) != 2 || s.GetEntities()[1].GetX() != -5 {
			t.Fatalf("ack %d: got %v", ack, s)
		}
	}
}

func TestAppendDoesNotAllocate(t *testing.T) {
	ents := make([]Entity, 100)
	for i := range ents {
		ents[i] = Entity{ID: uint32(i + 1), X: int32(i * 512), Y: int32(-i * 300), Facing: pb.Facing_FACING_LEFT, Anim: pb.Anim_ANIM_WALK}
	}
	body := make([]byte, 0, 4096)
	out := make([]byte, 0, 8192)
	allocs := testing.AllocsPerRun(100, func() {
		body = body[:0]
		for _, e := range ents {
			body = AppendEntity(body, e)
		}
		out = AppendSnapshot(out[:0], 123456, 654321, body)
	})
	if allocs != 0 {
		t.Fatalf("encoding a 100-entity snapshot allocated %.1f times per run, want 0", allocs)
	}
}

func TestDecodeClient(t *testing.T) {
	valid, _ := Encode(nil, &pb.Envelope{Payload: &pb.Envelope_Input{Input: &pb.PlayerInput{Seq: 1, MoveX: 1}}})
	unknown := protowire.AppendTag(nil, 99, protowire.BytesType)
	unknown = protowire.AppendBytes(unknown, []byte{1, 2, 3})
	cases := []struct {
		name string
		in   []byte
		want error
	}{
		{"valid input", valid, nil},
		{"empty message", nil, ErrEmpty},
		{"empty envelope", []byte{}, ErrEmpty},
		{"unknown payload field", unknown, ErrUnknownPayload},
		{"too large", make([]byte, MaxClientMessage+1), ErrTooLarge},
		{"truncated", valid[:len(valid)-1], ErrMalformed},
		{"bad wire type", []byte{0x0f}, ErrMalformed},
		{"length beyond end", []byte{0x2a, 0x7f, 0x08}, ErrMalformed},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			var env pb.Envelope
			err := DecodeClient(c.in, &env)
			if !errors.Is(err, c.want) {
				t.Fatalf("DecodeClient = %v, want %v", err, c.want)
			}
		})
	}
}

func TestDecodeResetsTheEnvelope(t *testing.T) {
	env := &pb.Envelope{Payload: &pb.Envelope_Ping{Ping: &pb.Ping{Nonce: 9}}}
	if err := DecodeClient(nil, env); !errors.Is(err, ErrEmpty) {
		t.Fatal(err)
	}
	if env.GetPing() != nil {
		t.Fatal("old payload survived a failed decode")
	}
}

// goldenSeeds adds every committed golden vector to a fuzz corpus.
func goldenSeeds(f *testing.F) {
	paths, err := filepath.Glob("../../../proto/testdata/*.bin")
	if err != nil || len(paths) == 0 {
		f.Fatalf("no golden vectors found: %v", err)
	}
	for _, p := range paths {
		b, err := os.ReadFile(p)
		if err != nil {
			f.Fatal(err)
		}
		f.Add(b)
	}
}

// FuzzDecodeClient: any bytes either fail cleanly or decode to a message
// that survives a re-encode and decode unchanged. It must never panic.
func FuzzDecodeClient(f *testing.F) {
	goldenSeeds(f)
	f.Add([]byte{0x2a, 0xff, 0xff, 0xff, 0xff, 0x0f})
	f.Fuzz(func(t *testing.T, b []byte) {
		var env pb.Envelope
		if err := DecodeClient(b, &env); err != nil {
			return
		}
		again, err := Encode(nil, &env)
		if err != nil {
			t.Fatalf("re-encode: %v", err)
		}
		var env2 pb.Envelope
		if err := DecodeServer(again, &env2); err != nil {
			t.Fatalf("decode of re-encoded message: %v", err)
		}
		if !proto.Equal(&env, &env2) {
			t.Fatalf("round trip changed the message: %v vs %v", &env, &env2)
		}
	})
}

// FuzzSnapshotAssembly: hand-assembled snapshots always equal proto.Marshal.
func FuzzSnapshotAssembly(f *testing.F) {
	f.Add(uint32(0), uint32(0), uint32(0), int32(0), int32(0), uint8(0), uint8(0), uint8(0))
	f.Add(uint32(1), uint32(2), uint32(math.MaxUint32), int32(math.MinInt32), int32(math.MaxInt32), uint8(4), uint8(2), uint8(3))
	f.Fuzz(func(t *testing.T, tick, ack, id uint32, x, y int32, facing, anim, count uint8) {
		ents := make([]Entity, int(count%8))
		for i := range ents {
			ents[i] = Entity{ID: id + uint32(i), X: x - int32(i), Y: y + int32(i), Facing: pb.Facing(facing % 5), Anim: pb.Anim(anim % 3)}
		}
		if got, want := assemble(tick, ack, ents), marshalSnapshot(t, tick, ack, ents); !bytes.Equal(got, want) {
			t.Fatalf("assembled %x, want %x", got, want)
		}
	})
}

func benchEntities(n int) []Entity {
	ents := make([]Entity, n)
	for i := range ents {
		ents[i] = Entity{ID: uint32(i + 1), X: int32(i * 511), Y: int32(i * 300), Facing: pb.Facing_FACING_DOWN, Anim: pb.Anim_ANIM_WALK}
	}
	return ents
}

// BenchmarkSnapshotEncodeOnce: encode 100 entities once, then assemble a
// snapshot for each of 100 recipients (the Phase 2 broadcast).
func BenchmarkSnapshotEncodeOnce(b *testing.B) {
	ents := benchEntities(100)
	body := make([]byte, 0, 4096)
	out := make([]byte, 0, 8192)
	b.ReportAllocs()
	for b.Loop() {
		body = body[:0]
		for _, e := range ents {
			body = AppendEntity(body, e)
		}
		for r := range uint32(100) {
			out = AppendSnapshot(out[:0], 1000, r, body)
		}
	}
}

// BenchmarkSnapshotMarshalPerRecipient is the naive alternative: build
// and marshal a Snapshot message for every recipient.
func BenchmarkSnapshotMarshalPerRecipient(b *testing.B) {
	ents := benchEntities(100)
	states := make([]*pb.EntityState, len(ents))
	for i, e := range ents {
		states[i] = toPB(e)
	}
	out := make([]byte, 0, 8192)
	b.ReportAllocs()
	for b.Loop() {
		for r := range uint32(100) {
			env := &pb.Envelope{Payload: &pb.Envelope_Snapshot{Snapshot: &pb.Snapshot{Tick: 1000, AckInputSeq: r, Entities: states}}}
			out, _ = marshal.MarshalAppend(out[:0], env)
		}
	}
}

func BenchmarkDecodeInput(b *testing.B) {
	msg, _ := Encode(nil, &pb.Envelope{Payload: &pb.Envelope_Input{Input: &pb.PlayerInput{Seq: 123456, ClientTick: 654321, MoveX: -1, MoveY: 1}}})
	var env pb.Envelope
	b.ReportAllocs()
	for b.Loop() {
		if err := DecodeClient(msg, &env); err != nil {
			b.Fatal(err)
		}
	}
}
