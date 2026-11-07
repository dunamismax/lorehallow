// Package wire encodes and decodes Envelopes for the WebSocket transport,
// with the size limits from docs/design/protocol.md.
//
// It also builds Snapshots the way ADR 0005 asks: each entity's state is
// encoded once per tick (AppendEntity) and every recipient's Snapshot is
// assembled from those shared bytes plus a few bytes of its own header
// (AppendSnapshot). The result is byte-for-byte what proto.Marshal with
// Deterministic set would produce, which the tests check.
package wire

import (
	"errors"
	"fmt"

	"google.golang.org/protobuf/encoding/protowire"
	"google.golang.org/protobuf/proto"

	"github.com/dunamismax/lorehallow/server/internal/pb"
)

// Size limits, in bytes, of one WebSocket message.
const (
	// MaxClientMessage bounds anything a client sends. The largest client
	// message is a ClientHello (a 16-character name and a version string).
	MaxClientMessage = 512
	// MaxServerMessage bounds anything the server sends. Clients must
	// accept messages up to this size.
	MaxServerMessage = 64 << 10
)

// Errors returned by Decode.
var (
	ErrTooLarge       = errors.New("wire: message too large")
	ErrEmpty          = errors.New("wire: envelope has no payload")
	ErrUnknownPayload = errors.New("wire: unknown payload")
	ErrMalformed      = errors.New("wire: malformed message")
)

var unmarshal = proto.UnmarshalOptions{RecursionLimit: 8}

// DecodeClient decodes one client message into env, which is reset first.
// It rejects messages over MaxClientMessage before parsing, malformed
// bytes, and envelopes with no payload or a payload this server doesn't
// know (a newer client, or a forged field number).
func DecodeClient(b []byte, env *pb.Envelope) error {
	return decode(b, MaxClientMessage, env)
}

// DecodeServer decodes one server message (used by bots and tests).
func DecodeServer(b []byte, env *pb.Envelope) error {
	return decode(b, MaxServerMessage, env)
}

func decode(b []byte, limit int, env *pb.Envelope) error {
	if len(b) > limit {
		return fmt.Errorf("%w: %d bytes, limit %d", ErrTooLarge, len(b), limit)
	}
	env.Reset()
	if err := unmarshal.Unmarshal(b, env); err != nil {
		return fmt.Errorf("%w: %w", ErrMalformed, err)
	}
	if env.GetPayload() == nil {
		if len(env.ProtoReflect().GetUnknown()) > 0 {
			return ErrUnknownPayload
		}
		return ErrEmpty
	}
	return nil
}

var marshal = proto.MarshalOptions{Deterministic: true}

// Encode returns the deterministic wire bytes of env, appended to dst.
// Use it for messages sent rarely; Snapshots use AppendSnapshot.
func Encode(dst []byte, env *pb.Envelope) ([]byte, error) {
	return marshal.MarshalAppend(dst, env)
}

// Field numbers used by the hand-assembled encoders below. They must match
// proto/lorehallow/v1/envelope.proto; TestAssemblyMatchesMarshal guards them.
const (
	envelopeSnapshotField = 6
	snapshotTickField     = 1
	snapshotAckField      = 2
	snapshotEntitiesField = 3
	entityIDField         = 1
	entityXField          = 2
	entityYField          = 3
	entityFacingField     = 4
	entityAnimField       = 5
)

// Entity is the visible state of one entity on one tick.
type Entity struct {
	ID     uint32
	X, Y   int32
	Facing pb.Facing
	Anim   pb.Anim
}

// AppendEntity appends e encoded as one element of Snapshot.entities
// (field tag, length, body) to dst. Concatenating these chunks gives the
// entities part of any Snapshot that contains them.
func AppendEntity(dst []byte, e Entity) []byte {
	n := entityBodySize(e)
	dst = protowire.AppendTag(dst, snapshotEntitiesField, protowire.BytesType)
	dst = protowire.AppendVarint(dst, uint64(n))
	dst = appendUint(dst, entityIDField, uint64(e.ID))
	dst = appendSint(dst, entityXField, e.X)
	dst = appendSint(dst, entityYField, e.Y)
	dst = appendUint(dst, entityFacingField, uint64(e.Facing))
	dst = appendUint(dst, entityAnimField, uint64(e.Anim))
	return dst
}

func entityBodySize(e Entity) int {
	return uintSize(entityIDField, uint64(e.ID)) +
		sintSize(entityXField, e.X) +
		sintSize(entityYField, e.Y) +
		uintSize(entityFacingField, uint64(e.Facing)) +
		uintSize(entityAnimField, uint64(e.Anim))
}

// SnapshotSize is the length of the Envelope AppendSnapshot writes.
func SnapshotSize(tick, ack uint32, entities []byte) int {
	n := snapshotBodySize(tick, ack, entities)
	return protowire.SizeTag(envelopeSnapshotField) + protowire.SizeBytes(n)
}

func snapshotBodySize(tick, ack uint32, entities []byte) int {
	return uintSize(snapshotTickField, uint64(tick)) + uintSize(snapshotAckField, uint64(ack)) + len(entities)
}

// AppendSnapshot appends a complete Envelope holding a Snapshot to dst.
// entities is a concatenation of AppendEntity chunks, shared by every
// recipient; only tick and ack are per recipient. It does not allocate
// when dst has room.
func AppendSnapshot(dst []byte, tick, ack uint32, entities []byte) []byte {
	n := snapshotBodySize(tick, ack, entities)
	dst = protowire.AppendTag(dst, envelopeSnapshotField, protowire.BytesType)
	dst = protowire.AppendVarint(dst, uint64(n))
	dst = appendUint(dst, snapshotTickField, uint64(tick))
	dst = appendUint(dst, snapshotAckField, uint64(ack))
	return append(dst, entities...)
}

// proto3 omits fields at their zero value; so do these helpers.

func appendUint(dst []byte, field protowire.Number, v uint64) []byte {
	if v == 0 {
		return dst
	}
	dst = protowire.AppendTag(dst, field, protowire.VarintType)
	return protowire.AppendVarint(dst, v)
}

func appendSint(dst []byte, field protowire.Number, v int32) []byte {
	if v == 0 {
		return dst
	}
	dst = protowire.AppendTag(dst, field, protowire.VarintType)
	return protowire.AppendVarint(dst, protowire.EncodeZigZag(int64(v)))
}

func uintSize(field protowire.Number, v uint64) int {
	if v == 0 {
		return 0
	}
	return protowire.SizeTag(field) + protowire.SizeVarint(v)
}

func sintSize(field protowire.Number, v int32) int {
	if v == 0 {
		return 0
	}
	return protowire.SizeTag(field) + protowire.SizeVarint(protowire.EncodeZigZag(int64(v)))
}
