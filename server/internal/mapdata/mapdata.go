// Package mapdata loads maps from Tiled JSON (.tmj) files: the size, the
// collision grid and the spawn points the server needs. The conventions
// are in docs/decisions/0006-tiled-map-format.md.
package mapdata

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strings"
)

// Units. Positions in the simulation and on the wire are subpixels.
const (
	TileSize          = 32 // pixels per tile side
	SubpixelsPerPixel = 16
	TileSubpixels     = TileSize * SubpixelsPerPixel // 512
)

// Layer names with a meaning to the server.
const (
	CollisionLayer = "collision"
	SpawnLayer     = "spawns"
	SpawnType      = "spawn"
)

// MaxTiles bounds a map's area so a hostile or broken file can't make the
// server allocate without limit (a 1024×1024 map; v1 zones are ~200×150).
const MaxTiles = 1 << 20

// Point is a position in subpixels.
type Point struct{ X, Y int32 }

// Map is what the server needs from a map file.
type Map struct {
	ID            string
	Width, Height int // in tiles
	solid         []bool
	Spawns        []Point
	// Tilesets lists the tileset references in file order.
	Tilesets []TilesetRef
	// TileLayers lists the drawn tile layer names, bottom to top.
	TileLayers []string
	// maxGID is the largest tile id used in any tile layer (flip bits removed).
	maxGID uint32
}

// TilesetRef is one entry of a map's tilesets list.
type TilesetRef struct {
	FirstGID uint32
	Source   string // relative to the map file
}

// WidthSubpixels is the map width in subpixels.
func (m *Map) WidthSubpixels() int32 { return int32(m.Width * TileSubpixels) }

// HeightSubpixels is the map height in subpixels.
func (m *Map) HeightSubpixels() int32 { return int32(m.Height * TileSubpixels) }

// Solid reports whether tile (tx, ty) blocks movement. Everything outside
// the map is solid, so the map edge is a wall.
func (m *Map) Solid(tx, ty int) bool {
	if tx < 0 || ty < 0 || tx >= m.Width || ty >= m.Height {
		return true
	}
	return m.solid[ty*m.Width+tx]
}

// SolidAt reports whether the subpixel point (x, y) is in a solid tile.
func (m *Map) SolidAt(x, y int32) bool {
	return m.Solid(floorDiv(x, TileSubpixels), floorDiv(y, TileSubpixels))
}

func floorDiv(a, b int32) int {
	q := a / b
	if (a%b != 0) && ((a < 0) != (b < 0)) {
		q--
	}
	return int(q)
}

// New builds a map directly (for tests and generated maps). solid is
// row-major, width*height long.
func New(id string, width, height int, solid []bool, spawns []Point) (*Map, error) {
	if width <= 0 || height <= 0 || width*height > MaxTiles {
		return nil, fmt.Errorf("map %s: bad size %dx%d", id, width, height)
	}
	if len(solid) != width*height {
		return nil, fmt.Errorf("map %s: solid has %d cells, want %d", id, len(solid), width*height)
	}
	m := &Map{ID: id, Width: width, Height: height, solid: solid, Spawns: spawns}
	return m, m.checkSpawns()
}

// Load reads and parses a .tmj file. The map id is the file name without
// its extension.
func Load(path string) (*Map, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return Parse(strings.TrimSuffix(filepath.Base(path), filepath.Ext(path)), data)
}

// The subset of Tiled's JSON format Lorehallow uses.
type tmj struct {
	Type        string     `json:"type"`
	Orientation string     `json:"orientation"`
	Infinite    bool       `json:"infinite"`
	Width       int        `json:"width"`
	Height      int        `json:"height"`
	TileWidth   int        `json:"tilewidth"`
	TileHeight  int        `json:"tileheight"`
	Layers      []tmjLayer `json:"layers"`
	Tilesets    []struct {
		FirstGID uint32 `json:"firstgid"`
		Source   string `json:"source"`
	} `json:"tilesets"`
}

type tmjLayer struct {
	Name        string      `json:"name"`
	Type        string      `json:"type"`
	Width       int         `json:"width"`
	Height      int         `json:"height"`
	Data        []uint32    `json:"data"`
	Encoding    string      `json:"encoding"`
	Compression string      `json:"compression"`
	Objects     []tmjObject `json:"objects"`
	Layers      []tmjLayer  `json:"layers"`
}

type tmjObject struct {
	Type  string  `json:"type"`
	Class string  `json:"class"`
	X     float64 `json:"x"`
	Y     float64 `json:"y"`
	Point bool    `json:"point"`
}

// Tiled stores flip and rotation flags in the top bits of a tile id.
const gidFlags = 0xF0000000

// Parse parses Tiled map JSON.
func Parse(id string, data []byte) (*Map, error) {
	var t tmj
	if err := json.Unmarshal(data, &t); err != nil {
		return nil, fmt.Errorf("map %s: %w", id, err)
	}
	var errs []error
	bad := func(format string, args ...any) {
		errs = append(errs, fmt.Errorf("map %s: "+format, append([]any{id}, args...)...))
	}
	if t.Type != "map" {
		bad("type is %q, want \"map\"", t.Type)
	}
	if t.Orientation != "orthogonal" {
		bad("orientation is %q, want orthogonal", t.Orientation)
	}
	if t.Infinite {
		bad("infinite maps are not supported")
	}
	if t.TileWidth != TileSize || t.TileHeight != TileSize {
		bad("tiles are %dx%d, want %dx%d", t.TileWidth, t.TileHeight, TileSize, TileSize)
	}
	if t.Width <= 0 || t.Height <= 0 || t.Width*t.Height > MaxTiles {
		bad("bad size %dx%d", t.Width, t.Height)
	}
	if len(errs) > 0 {
		return nil, errors.Join(errs...)
	}

	m := &Map{ID: id, Width: t.Width, Height: t.Height}
	for _, ts := range t.Tilesets {
		m.Tilesets = append(m.Tilesets, TilesetRef{FirstGID: ts.FirstGID, Source: ts.Source})
	}
	var collision *tmjLayer
	var spawnLayers int
	layers := flatten(t.Layers)
	for i := range layers {
		l := &layers[i]
		switch l.Type {
		case "tilelayer":
			if l.Encoding != "" && l.Encoding != "csv" || l.Compression != "" {
				bad("layer %q: use uncompressed JSON array data", l.Name)
				continue
			}
			if l.Width != t.Width || l.Height != t.Height || len(l.Data) != t.Width*t.Height {
				bad("layer %q: is %dx%d with %d tiles, want %dx%d", l.Name, l.Width, l.Height, len(l.Data), t.Width, t.Height)
				continue
			}
			for _, gid := range l.Data {
				m.maxGID = max(m.maxGID, gid&^gidFlags)
			}
			if l.Name == CollisionLayer {
				if collision != nil {
					bad("more than one %q layer", CollisionLayer)
				}
				collision = l
			} else {
				m.TileLayers = append(m.TileLayers, l.Name)
			}
		case "objectgroup":
			if l.Name != SpawnLayer {
				continue
			}
			spawnLayers++
			for _, o := range l.Objects {
				if o.Type != SpawnType && o.Class != SpawnType {
					continue
				}
				if !o.Point {
					bad("spawn at (%g, %g) must be a point object", o.X, o.Y)
					continue
				}
				x, y := o.X*SubpixelsPerPixel, o.Y*SubpixelsPerPixel
				if x < math.MinInt32 || x > math.MaxInt32 || y < math.MinInt32 || y > math.MaxInt32 {
					bad("spawn at (%g, %g) is out of range", o.X, o.Y)
					continue
				}
				m.Spawns = append(m.Spawns, Point{X: int32(math.Round(x)), Y: int32(math.Round(y))})
			}
		}
	}
	if collision == nil {
		bad("no %q tile layer", CollisionLayer)
	} else {
		m.solid = make([]bool, t.Width*t.Height)
		for i, gid := range collision.Data {
			m.solid[i] = gid&^gidFlags != 0
		}
	}
	if spawnLayers == 0 {
		bad("no %q object layer", SpawnLayer)
	}
	if len(errs) == 0 {
		if err := m.checkSpawns(); err != nil {
			errs = append(errs, err)
		}
	}
	if len(errs) > 0 {
		return nil, errors.Join(errs...)
	}
	return m, nil
}

func (m *Map) checkSpawns() error {
	if len(m.Spawns) == 0 {
		return fmt.Errorf("map %s: no spawn points", m.ID)
	}
	var errs []error
	for _, s := range m.Spawns {
		if s.X < 0 || s.Y < 0 || s.X >= m.WidthSubpixels() || s.Y >= m.HeightSubpixels() {
			errs = append(errs, fmt.Errorf("map %s: spawn %v is outside the map", m.ID, s))
		} else if m.SolidAt(s.X, s.Y) {
			errs = append(errs, fmt.Errorf("map %s: spawn %v is inside a solid tile", m.ID, s))
		}
	}
	return errors.Join(errs...)
}

// flatten returns layers with groups expanded, in draw order.
func flatten(layers []tmjLayer) []tmjLayer {
	var out []tmjLayer
	for _, l := range layers {
		if l.Type == "group" {
			out = append(out, flatten(l.Layers)...)
			continue
		}
		out = append(out, l)
	}
	return out
}
