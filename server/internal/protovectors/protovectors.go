// Package protovectors defines the golden protocol test vectors.
//
// Each vector is an Envelope encoded with deterministic marshaling. The
// lorehallow-protovectors command writes them to proto/testdata/ as a .bin
// file (the exact bytes) and a .json file (protojson with proto field
// names and enum numbers) describing the content. Go and GDScript tests decode every .bin,
// check it against the .json, re-encode it, and require identical bytes.
//
// When you add a message type to Envelope, add vectors for it here and run
// `make proto`. TestEveryPayloadCovered fails until you do.
package protovectors

import (
	"bytes"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"

	"github.com/dunamismax/lorehallow/server/internal/pb"
)

// Vector is one named golden message.
type Vector struct {
	Name string
	Msg  *pb.Envelope
}

// All returns every golden vector, in a stable order.
func All() []Vector {
	return []Vector{
		{"envelope_empty", &pb.Envelope{}},
		{"ping_zero", &pb.Envelope{Payload: &pb.Envelope_Ping{Ping: &pb.Ping{}}}},
		{"ping_basic", &pb.Envelope{Payload: &pb.Envelope_Ping{Ping: &pb.Ping{Nonce: 7, SentAtMs: 1_700_000_000_000}}}},
		{"ping_max", &pb.Envelope{Payload: &pb.Envelope_Ping{Ping: &pb.Ping{Nonce: math.MaxUint32, SentAtMs: math.MaxInt64}}}},
		{"pong_basic", &pb.Envelope{Payload: &pb.Envelope_Pong{Pong: &pb.Pong{Nonce: 7, PingSentAtMs: 1_700_000_000_000, ReceivedAtMs: 1_700_000_000_042}}}},
		{"pong_negative", &pb.Envelope{Payload: &pb.Envelope_Pong{Pong: &pb.Pong{Nonce: 1, PingSentAtMs: -1, ReceivedAtMs: math.MinInt64}}}},

		{"client_hello_basic", &pb.Envelope{Payload: &pb.Envelope_ClientHello{ClientHello: &pb.ClientHello{ProtocolVersion: 1, ClientVersion: "v0.2.0", DisplayName: "Wren"}}}},
		{"client_hello_unicode", &pb.Envelope{Payload: &pb.Envelope_ClientHello{ClientHello: &pb.ClientHello{ProtocolVersion: math.MaxUint32, ClientVersion: "v0.2.0-3-gabcdef0-dirty", DisplayName: "Élodie 森"}}}},
		{"server_hello_basic", &pb.Envelope{Payload: &pb.Envelope_ServerHello{ServerHello: &pb.ServerHello{
			PlayerId: 42, TickRate: 20, ServerTick: 123_456,
			Spawn: &pb.EntityState{Id: 42, X: 5 * 512, Y: 7 * 512, Facing: pb.Facing_FACING_DOWN, Anim: pb.Anim_ANIM_IDLE},
			MapId: "starter", ServerVersion: "v0.2.0",
		}}}},
		{"input_basic", &pb.Envelope{Payload: &pb.Envelope_Input{Input: &pb.PlayerInput{Seq: 1, ClientTick: 100, MoveX: 1, MoveY: -1}}}},
		{"input_extremes", &pb.Envelope{Payload: &pb.Envelope_Input{Input: &pb.PlayerInput{Seq: math.MaxUint32, ClientTick: math.MaxUint32, MoveX: math.MinInt32, MoveY: math.MaxInt32}}}},
		{"input_still", &pb.Envelope{Payload: &pb.Envelope_Input{Input: &pb.PlayerInput{Seq: 7, ClientTick: 8}}}},
		{"snapshot_empty", &pb.Envelope{Payload: &pb.Envelope_Snapshot{Snapshot: &pb.Snapshot{Tick: 1}}}},
		{"snapshot_three", &pb.Envelope{Payload: &pb.Envelope_Snapshot{Snapshot: &pb.Snapshot{Tick: 9_000, AckInputSeq: 8_999, Entities: []*pb.EntityState{
			{Id: 1, X: 2560, Y: 3584, Facing: pb.Facing_FACING_DOWN, Anim: pb.Anim_ANIM_IDLE},
			{Id: 2, X: -16, Y: 0, Facing: pb.Facing_FACING_LEFT, Anim: pb.Anim_ANIM_WALK},
			{Id: math.MaxUint32, X: math.MaxInt32, Y: math.MinInt32, Facing: pb.Facing_FACING_RIGHT, Anim: pb.Anim_ANIM_WALK},
		}}}}},
		{"entity_joined_basic", &pb.Envelope{Payload: &pb.Envelope_EntityJoined{EntityJoined: &pb.EntityJoined{
			State: &pb.EntityState{Id: 3, X: 512, Y: 1024, Facing: pb.Facing_FACING_UP, Anim: pb.Anim_ANIM_WALK}, DisplayName: "Bram",
		}}}},
		{"entity_left_basic", &pb.Envelope{Payload: &pb.Envelope_EntityLeft{EntityLeft: &pb.EntityLeft{Id: 3}}}},
		{"disconnect_too_slow", &pb.Envelope{Payload: &pb.Envelope_Disconnect{Disconnect: &pb.Disconnect{Reason: pb.DisconnectReason_DISCONNECT_REASON_TOO_SLOW, Text: "Your connection couldn't keep up."}}}},
		{"disconnect_protocol", &pb.Envelope{Payload: &pb.Envelope_Disconnect{Disconnect: &pb.Disconnect{Reason: pb.DisconnectReason_DISCONNECT_REASON_PROTOCOL_MISMATCH, Text: "Update required"}}}},
	}
}

// Encode returns the deterministic wire bytes of m.
func Encode(m proto.Message) ([]byte, error) {
	return proto.MarshalOptions{Deterministic: true}.Marshal(m)
}

// Describe returns the protojson description of m, with proto field
// names and a trailing newline, formatted stably.
func Describe(m proto.Message) ([]byte, error) {
	raw, err := protojson.MarshalOptions{UseProtoNames: true, UseEnumNumbers: true}.Marshal(m)
	if err != nil {
		return nil, err
	}
	// protojson output whitespace is deliberately unstable; re-indent it.
	var out bytes.Buffer
	if err := indentJSON(&out, raw); err != nil {
		return nil, err
	}
	out.WriteByte('\n')
	return out.Bytes(), nil
}

// Write writes every vector into dir as <name>.bin and <name>.json and
// removes stale vector files that no longer correspond to a vector.
func Write(dir string) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	want := map[string]bool{}
	for _, v := range All() {
		b, err := Encode(v.Msg)
		if err != nil {
			return fmt.Errorf("encode %s: %w", v.Name, err)
		}
		j, err := Describe(v.Msg)
		if err != nil {
			return fmt.Errorf("describe %s: %w", v.Name, err)
		}
		for name, data := range map[string][]byte{v.Name + ".bin": b, v.Name + ".json": j} {
			want[name] = true
			if err := os.WriteFile(filepath.Join(dir, name), data, 0o644); err != nil {
				return err
			}
		}
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}
	for _, e := range entries {
		ext := filepath.Ext(e.Name())
		if (ext == ".bin" || ext == ".json") && !want[e.Name()] {
			if err := os.Remove(filepath.Join(dir, e.Name())); err != nil {
				return err
			}
		}
	}
	return nil
}

// Names returns the sorted vector names found in dir (by .bin file).
func Names(dir string) ([]string, error) {
	matches, err := filepath.Glob(filepath.Join(dir, "*.bin"))
	if err != nil {
		return nil, err
	}
	names := make([]string, 0, len(matches))
	for _, m := range matches {
		names = append(names, filepath.Base(m[:len(m)-len(".bin")]))
	}
	sort.Strings(names)
	return names, nil
}
