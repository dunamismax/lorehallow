package mapdata

import (
	"encoding/json"
	"errors"
	"fmt"
	"image/png"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// tsj is the subset of Tiled's external tileset JSON that maps use.
type tsj struct {
	Type        string `json:"type"`
	Name        string `json:"name"`
	TileWidth   int    `json:"tilewidth"`
	TileHeight  int    `json:"tileheight"`
	TileCount   int    `json:"tilecount"`
	Columns     int    `json:"columns"`
	Image       string `json:"image"`
	ImageWidth  int    `json:"imagewidth"`
	ImageHeight int    `json:"imageheight"`
	Margin      int    `json:"margin"`
	Spacing     int    `json:"spacing"`
}

// Validate loads the map at path and checks it against the files around
// it: every tileset exists, is named after processed art under
// root/art/processed/tiles/, matches its image, and covers every tile id
// the map uses. root is the repository root.
func Validate(root, path string) error {
	m, err := Load(path)
	if err != nil {
		return err
	}
	var errs []error
	bad := func(format string, args ...any) {
		errs = append(errs, fmt.Errorf("map %s: "+format, append([]any{m.ID}, args...)...))
	}
	if len(m.Tilesets) == 0 {
		bad("no tilesets")
	}
	refs := append([]TilesetRef(nil), m.Tilesets...)
	sort.Slice(refs, func(i, j int) bool { return refs[i].FirstGID < refs[j].FirstGID })
	var next uint32 = 1
	for _, ref := range refs {
		if ref.Source == "" {
			bad("embedded tilesets are not allowed; use content/tilesets/<name>.tsj")
			continue
		}
		if ref.FirstGID != next {
			bad("tileset %s starts at gid %d, want %d (gaps or overlaps)", ref.Source, ref.FirstGID, next)
		}
		count, err := checkTileset(root, filepath.Join(filepath.Dir(path), ref.Source))
		if err != nil {
			errs = append(errs, fmt.Errorf("map %s: %w", m.ID, err))
			continue
		}
		next = ref.FirstGID + uint32(count)
	}
	if m.maxGID >= next && len(errs) == 0 {
		bad("tile id %d is beyond the last tileset (ids end at %d)", m.maxGID, next-1)
	}
	return errors.Join(errs...)
}

func checkTileset(root, path string) (int, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return 0, fmt.Errorf("tileset: %w", err)
	}
	var t tsj
	if err := json.Unmarshal(data, &t); err != nil {
		return 0, fmt.Errorf("tileset %s: %w", path, err)
	}
	name := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	var errs []error
	bad := func(format string, args ...any) {
		errs = append(errs, fmt.Errorf("tileset %s: "+format, append([]any{name}, args...)...))
	}
	if t.Type != "tileset" {
		bad("type is %q, want \"tileset\"", t.Type)
	}
	if t.Name != name {
		bad("name is %q, want %q (the file name and the processed art id)", t.Name, name)
	}
	if t.TileWidth != TileSize || t.TileHeight != TileSize || t.Margin != 0 || t.Spacing != 0 {
		bad("tiles must be %dx%d with no margin or spacing", TileSize, TileSize)
	}
	if t.Columns <= 0 || t.TileCount <= 0 || t.TileCount%t.Columns != 0 {
		bad("tilecount %d and columns %d don't make a full grid", t.TileCount, t.Columns)
	}
	img := filepath.Clean(filepath.Join(filepath.Dir(path), t.Image))
	want := filepath.Join(root, "art", "processed", "tiles", name+".png")
	if abs, err := filepath.Abs(img); err != nil || abs != mustAbs(want) {
		bad("image %q must be art/processed/tiles/%s.png", t.Image, name)
	}
	f, err := os.Open(want)
	if err != nil {
		bad("image: %v", err)
		return 0, errors.Join(errs...)
	}
	defer func() { _ = f.Close() }()
	cfg, err := png.DecodeConfig(f)
	if err != nil {
		bad("image: %v", err)
		return 0, errors.Join(errs...)
	}
	if t.Columns > 0 {
		rows := t.TileCount / t.Columns
		if cfg.Width != t.Columns*TileSize || cfg.Height != rows*TileSize || cfg.Width != t.ImageWidth || cfg.Height != t.ImageHeight {
			bad("image is %dx%d, tileset says %dx%d with %d columns and %d rows", cfg.Width, cfg.Height, t.ImageWidth, t.ImageHeight, t.Columns, rows)
		}
	}
	return t.TileCount, errors.Join(errs...)
}

func mustAbs(p string) string {
	a, err := filepath.Abs(p)
	if err != nil {
		return p
	}
	return a
}
