package protovectors

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"google.golang.org/protobuf/proto"

	"github.com/dunamismax/lorehallow/server/internal/pb"
)

const testdata = "../../../proto/testdata"

// TestGoldenRoundTrip decodes every committed vector, re-encodes it and
// requires identical bytes, and checks the JSON description matches.
func TestGoldenRoundTrip(t *testing.T) {
	names, err := Names(testdata)
	if err != nil {
		t.Fatal(err)
	}
	if len(names) == 0 {
		t.Fatal("no vectors in proto/testdata; run make proto")
	}
	for _, name := range names {
		t.Run(name, func(t *testing.T) {
			raw, err := os.ReadFile(filepath.Join(testdata, name+".bin"))
			if err != nil {
				t.Fatal(err)
			}
			var env pb.Envelope
			if err := proto.Unmarshal(raw, &env); err != nil {
				t.Fatalf("decode: %v", err)
			}
			again, err := Encode(&env)
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(raw, again) {
				t.Errorf("re-encoded bytes differ:\n got %x\nwant %x", again, raw)
			}
			wantJSON, err := os.ReadFile(filepath.Join(testdata, name+".json"))
			if err != nil {
				t.Fatal(err)
			}
			gotJSON, err := Describe(&env)
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(gotJSON, wantJSON) {
				t.Errorf("description differs:\n got %s\nwant %s", gotJSON, wantJSON)
			}
		})
	}
}

// TestCommittedVectorsAreCurrent fails when proto/testdata is out of date
// with All(); `make proto` regenerates it.
func TestCommittedVectorsAreCurrent(t *testing.T) {
	dir := t.TempDir()
	if err := Write(dir); err != nil {
		t.Fatal(err)
	}
	fresh, _ := filepath.Glob(filepath.Join(dir, "*"))
	committed, _ := filepath.Glob(filepath.Join(testdata, "*.*"))
	var committedVectors []string
	for _, c := range committed {
		if ext := filepath.Ext(c); ext == ".bin" || ext == ".json" {
			committedVectors = append(committedVectors, c)
		}
	}
	if len(fresh) != len(committedVectors) {
		t.Fatalf("proto/testdata has %d vector files, want %d; run make proto", len(committedVectors), len(fresh))
	}
	for _, f := range fresh {
		want, _ := os.ReadFile(f)
		got, err := os.ReadFile(filepath.Join(testdata, filepath.Base(f)))
		if err != nil || !bytes.Equal(got, want) {
			t.Errorf("%s is stale; run make proto", filepath.Base(f))
		}
	}
}

// TestEveryPayloadCovered requires at least one vector per Envelope payload.
func TestEveryPayloadCovered(t *testing.T) {
	oneof := (&pb.Envelope{}).ProtoReflect().Descriptor().Oneofs().ByName("payload")
	covered := map[string]bool{}
	for _, v := range All() {
		if f := v.Msg.ProtoReflect().WhichOneof(oneof); f != nil {
			covered[string(f.Name())] = true
		}
	}
	fields := oneof.Fields()
	for i := range fields.Len() {
		name := string(fields.Get(i).Name())
		if !covered[name] {
			t.Errorf("Envelope payload %q has no golden vector; add one in protovectors.All", name)
		}
	}
}

func TestVectorNamesUnique(t *testing.T) {
	seen := map[string]bool{}
	for _, v := range All() {
		if seen[v.Name] {
			t.Errorf("duplicate vector name %q", v.Name)
		}
		seen[v.Name] = true
	}
}
