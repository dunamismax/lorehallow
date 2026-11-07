package mapdata

import (
	"encoding/json"
	"fmt"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// fixture builds Tiled JSON for a w×h map. solid lists solid tiles as
// "x,y"; spawns are tile coordinates (placed at the tile center).
func fixture(w, h int, solid []string, spawns [][2]int, edit func(m map[string]any)) []byte {
	ground := make([]uint32, w*h)
	coll := make([]uint32, w*h)
	for i := range ground {
		ground[i] = 1
	}
	for _, s := range solid {
		var x, y int
		if _, err := fmt.Sscanf(s, "%d,%d", &x, &y); err != nil {
			panic(err)
		}
		coll[y*w+x] = 2
	}
	var objs []map[string]any
	for _, s := range spawns {
		objs = append(objs, map[string]any{"type": "spawn", "x": s[0]*32 + 16, "y": s[1]*32 + 16, "point": true})
	}
	m := map[string]any{
		"type": "map", "orientation": "orthogonal", "infinite": false,
		"width": w, "height": h, "tilewidth": 32, "tileheight": 32,
		"layers": []any{
			map[string]any{"name": "ground", "type": "tilelayer", "width": w, "height": h, "data": ground},
			map[string]any{"name": "collision", "type": "tilelayer", "width": w, "height": h, "data": coll},
			map[string]any{"name": "spawns", "type": "objectgroup", "objects": objs},
		},
		"tilesets": []any{map[string]any{"firstgid": 1, "source": "../tilesets/tiles_test.tsj"}},
	}
	if edit != nil {
		edit(m)
	}
	b, err := json.Marshal(m)
	if err != nil {
		panic(err)
	}
	return b
}

func layer(m map[string]any, i int) map[string]any { return m["layers"].([]any)[i].(map[string]any) }

func TestParseValid(t *testing.T) {
	m, err := Parse("test", fixture(4, 3, []string{"1,1", "3,2"}, [][2]int{{0, 0}, {2, 1}}, nil))
	if err != nil {
		t.Fatal(err)
	}
	if m.Width != 4 || m.Height != 3 || m.WidthSubpixels() != 4*512 {
		t.Fatalf("size %dx%d", m.Width, m.Height)
	}
	if !m.Solid(1, 1) || !m.Solid(3, 2) || m.Solid(0, 0) || m.Solid(2, 1) {
		t.Fatal("collision grid wrong")
	}
	if len(m.Spawns) != 2 || m.Spawns[1] != (Point{X: (2*32 + 16) * 16, Y: (1*32 + 16) * 16}) {
		t.Fatalf("spawns %v", m.Spawns)
	}
	if len(m.TileLayers) != 1 || m.TileLayers[0] != "ground" {
		t.Fatalf("tile layers %v", m.TileLayers)
	}
}

func TestOutsideTheMapIsSolid(t *testing.T) {
	m, err := New("t", 2, 2, make([]bool, 4), []Point{{X: 100, Y: 100}})
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range [][2]int{{-1, 0}, {0, -1}, {2, 0}, {0, 2}, {-100, -100}} {
		if !m.Solid(c[0], c[1]) {
			t.Errorf("Solid(%d, %d) = false outside the map", c[0], c[1])
		}
	}
	// Negative subpixels round toward negative infinity: -1 is tile -1.
	if !m.SolidAt(-1, 0) || m.SolidAt(0, 0) || m.SolidAt(511, 511) || m.SolidAt(512, 511) || !m.SolidAt(1024, 0) {
		t.Error("SolidAt tile rounding wrong")
	}
}

func TestFlipFlagsAreIgnored(t *testing.T) {
	b := fixture(2, 1, nil, [][2]int{{0, 0}}, func(m map[string]any) {
		layer(m, 1)["data"] = []uint32{0, 0x80000000 | 2}
	})
	m, err := Parse("t", b)
	if err != nil {
		t.Fatal(err)
	}
	if !m.Solid(1, 0) || m.maxGID != 2 {
		t.Fatalf("flipped tile: solid=%v maxGID=%d", m.Solid(1, 0), m.maxGID)
	}
}

func TestParseRejects(t *testing.T) {
	cases := []struct {
		name string
		edit func(m map[string]any)
		want string
	}{
		{"not a map", func(m map[string]any) { m["type"] = "tileset" }, `want "map"`},
		{"isometric", func(m map[string]any) { m["orientation"] = "isometric" }, "orthogonal"},
		{"infinite", func(m map[string]any) { m["infinite"] = true }, "infinite"},
		{"16px tiles", func(m map[string]any) { m["tilewidth"] = 16 }, "want 32x32"},
		{"huge", func(m map[string]any) { m["width"], m["height"] = 4096, 4096 }, "bad size"},
		{"no collision", func(m map[string]any) { layer(m, 1)["name"] = "walls" }, `no "collision" tile layer`},
		{"short layer", func(m map[string]any) { layer(m, 0)["data"] = []uint32{1} }, `layer "ground"`},
		{"compressed", func(m map[string]any) { layer(m, 0)["compression"] = "zlib" }, "uncompressed"},
		{"no spawn layer", func(m map[string]any) { layer(m, 2)["name"] = "objects" }, `no "spawns" object layer`},
		{"no spawns", func(m map[string]any) { layer(m, 2)["objects"] = []any{} }, "no spawn points"},
		{"spawn not a point", func(m map[string]any) {
			layer(m, 2)["objects"] = []any{map[string]any{"type": "spawn", "x": 16, "y": 16}}
		}, "point object"},
		{"spawn outside", func(m map[string]any) {
			layer(m, 2)["objects"] = []any{map[string]any{"type": "spawn", "x": 9999, "y": 16, "point": true}}
		}, "outside the map"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, err := Parse("bad", fixture(3, 3, nil, [][2]int{{1, 1}}, c.edit))
			if err == nil || !strings.Contains(err.Error(), c.want) {
				t.Fatalf("Parse error = %v, want it to mention %q", err, c.want)
			}
		})
	}
}

func TestSpawnInsideWallIsRejected(t *testing.T) {
	_, err := Parse("bad", fixture(3, 3, []string{"1,1"}, [][2]int{{1, 1}}, nil))
	if err == nil || !strings.Contains(err.Error(), "inside a solid tile") {
		t.Fatalf("err = %v", err)
	}
}

func TestSpawnClassIsAccepted(t *testing.T) {
	// Tiled 1.9+ can write "class" instead of "type".
	b := fixture(3, 3, nil, nil, func(m map[string]any) {
		layer(m, 2)["objects"] = []any{map[string]any{"class": "spawn", "x": 16, "y": 16, "point": true}}
	})
	if _, err := Parse("ok", b); err != nil {
		t.Fatal(err)
	}
}

// repo writes a minimal repository layout with one 2×1 tileset.
func repo(t *testing.T, mapJSON []byte, tileset map[string]any, imgW, imgH int) (root, mapPath string) {
	t.Helper()
	root = t.TempDir()
	for _, d := range []string{"content/maps", "content/tilesets", "art/processed/tiles"} {
		if err := os.MkdirAll(filepath.Join(root, d), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	f, err := os.Create(filepath.Join(root, "art/processed/tiles/tiles_test.png"))
	if err != nil {
		t.Fatal(err)
	}
	if err := png.Encode(f, image.NewNRGBA(image.Rect(0, 0, imgW, imgH))); err != nil {
		t.Fatal(err)
	}
	_ = f.Close()
	ts, _ := json.Marshal(tileset)
	if err := os.WriteFile(filepath.Join(root, "content/tilesets/tiles_test.tsj"), ts, 0o644); err != nil {
		t.Fatal(err)
	}
	mapPath = filepath.Join(root, "content/maps/test.tmj")
	if err := os.WriteFile(mapPath, mapJSON, 0o644); err != nil {
		t.Fatal(err)
	}
	return root, mapPath
}

func goodTileset() map[string]any {
	return map[string]any{
		"type": "tileset", "name": "tiles_test", "tilewidth": 32, "tileheight": 32, "tilecount": 2, "columns": 2,
		"image": "../../art/processed/tiles/tiles_test.png", "imagewidth": 64, "imageheight": 32,
	}
}

func TestValidate(t *testing.T) {
	good := fixture(3, 3, []string{"0,0"}, [][2]int{{1, 1}}, nil)
	t.Run("ok", func(t *testing.T) {
		root, p := repo(t, good, goodTileset(), 64, 32)
		if err := Validate(root, p); err != nil {
			t.Fatal(err)
		}
	})
	cases := []struct {
		name    string
		mapJSON []byte
		edit    func(ts map[string]any)
		imgW    int
		want    string
	}{
		{"tile id beyond tileset", fixture(3, 3, nil, [][2]int{{1, 1}}, func(m map[string]any) {
			layer(m, 0)["data"] = []uint32{1, 1, 1, 1, 3, 1, 1, 1, 1}
		}), nil, 64, "beyond the last tileset"},
		{"name mismatch", good, func(ts map[string]any) { ts["name"] = "other" }, 64, "want \"tiles_test\""},
		{"wrong image path", good, func(ts map[string]any) { ts["image"] = "elsewhere.png" }, 64, "must be art/processed/tiles"},
		{"image size mismatch", good, nil, 96, "image is 96x32"},
		{"ragged grid", good, func(ts map[string]any) { ts["tilecount"] = 3 }, 64, "full grid"},
		{"missing tileset file", fixture(3, 3, nil, [][2]int{{1, 1}}, func(m map[string]any) {
			m["tilesets"] = []any{map[string]any{"firstgid": 1, "source": "../tilesets/nope.tsj"}}
		}), nil, 64, "no such file"},
		{"embedded tileset", fixture(3, 3, nil, [][2]int{{1, 1}}, func(m map[string]any) {
			m["tilesets"] = []any{map[string]any{"firstgid": 1, "name": "inline"}}
		}), nil, 64, "embedded tilesets"},
		{"gid gap", fixture(3, 3, nil, [][2]int{{1, 1}}, func(m map[string]any) {
			m["tilesets"] = []any{map[string]any{"firstgid": 5, "source": "../tilesets/tiles_test.tsj"}}
		}), nil, 64, "starts at gid 5, want 1"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			ts := goodTileset()
			if c.edit != nil {
				c.edit(ts)
			}
			root, p := repo(t, c.mapJSON, ts, c.imgW, 32)
			err := Validate(root, p)
			if err == nil || !strings.Contains(err.Error(), c.want) {
				t.Fatalf("Validate error = %v, want it to mention %q", err, c.want)
			}
		})
	}
}

func TestContentMapsValidate(t *testing.T) {
	maps, _ := filepath.Glob("../../../content/maps/*.tmj")
	if len(maps) == 0 {
		t.Fatal("no maps in content/maps")
	}
	for _, p := range maps {
		if err := Validate("../../..", p); err != nil {
			t.Error(err)
		}
	}
}

func FuzzParse(f *testing.F) {
	f.Add(fixture(3, 3, []string{"0,0"}, [][2]int{{1, 1}}, nil))
	if b, err := os.ReadFile("../../../content/maps/starter.tmj"); err == nil {
		f.Add(b)
	}
	f.Fuzz(func(t *testing.T, b []byte) {
		m, err := Parse("fuzz", b)
		if err != nil {
			return
		}
		if m.Width*m.Height > MaxTiles || len(m.Spawns) == 0 {
			t.Fatalf("accepted an invalid map: %dx%d, %d spawns", m.Width, m.Height, len(m.Spawns))
		}
		for _, s := range m.Spawns {
			if m.SolidAt(s.X, s.Y) {
				t.Fatalf("accepted a spawn in a wall: %v", s)
			}
		}
	})
}
