// Package protovectors defines the golden protocol test vectors.
//
// Each vector is an Envelope encoded with deterministic marshaling. The
// lorehallow-protovectors command writes them to proto/testdata/ as a .bin
// file (the exact bytes) and a .json file (protojson with proto field
// names) describing the content. Go and GDScript tests decode every .bin,
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
	}
}

// Encode returns the deterministic wire bytes of m.
func Encode(m proto.Message) ([]byte, error) {
	return proto.MarshalOptions{Deterministic: true}.Marshal(m)
}

// Describe returns the protojson description of m, with proto field
// names and a trailing newline, formatted stably.
func Describe(m proto.Message) ([]byte, error) {
	raw, err := protojson.MarshalOptions{UseProtoNames: true}.Marshal(m)
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
