// Package pixelfont reads Lorehallow's hand-drawn pixel font source
// (web/static/fonts/lorehallow-pixel.txt), compiles it into a TrueType font,
// and draws text with it onto images (for the social cards).
//
// The source format is described at the top of that file: one grid of "#"
// and "." per glyph, capital height 7 rows, descenders in rows 8 and 9.
package pixelfont

import (
	"bufio"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"slices"
	"strconv"
	"strings"
	"unicode/utf8"
)

// Rows above the baseline (capital height) and the most rows below it.
const (
	CapRows     = 7
	DescentRows = 2
	Rows        = CapRows + DescentRows
)

// Glyph is one character's pixels: Bits[row][col], row 0 at the top of
// the capital height.
type Glyph struct {
	Rune  rune
	Width int
	Bits  [Rows][]bool
}

// Advance is the glyph's width plus one pixel of spacing.
func (g *Glyph) Advance() int { return g.Width + 1 }

// Font is a parsed pixel font.
type Font struct {
	Name   string
	Glyphs map[rune]*Glyph
}

// Parse reads the font source. Every error names its line.
func Parse(src string) (*Font, error) {
	f := &Font{Glyphs: map[rune]*Glyph{}}
	var cur *Glyph
	var rows []string
	start := 0
	finish := func() error {
		if cur == nil {
			return nil
		}
		if len(rows) != CapRows && len(rows) != Rows {
			return fmt.Errorf("line %d: glyph %q has %d rows; want %d or %d", start, cur.Rune, len(rows), CapRows, Rows)
		}
		cur.Width = len(rows[0])
		for i := range Rows {
			cur.Bits[i] = make([]bool, cur.Width)
			if i >= len(rows) {
				continue
			}
			if len(rows[i]) != cur.Width {
				return fmt.Errorf("line %d: glyph %q row %d is %d wide; want %d", start+i+1, cur.Rune, i+1, len(rows[i]), cur.Width)
			}
			for j, c := range rows[i] {
				switch c {
				case '#':
					cur.Bits[i][j] = true
				case '.':
				default:
					return fmt.Errorf("line %d: glyph %q has %q (use # or a dot)", start+i+1, cur.Rune, c)
				}
			}
		}
		if _, dup := f.Glyphs[cur.Rune]; dup {
			return fmt.Errorf("line %d: glyph %q defined twice", start, cur.Rune)
		}
		f.Glyphs[cur.Rune] = cur
		cur, rows = nil, nil
		return nil
	}
	sc := bufio.NewScanner(strings.NewReader(src))
	n := 0
	for sc.Scan() {
		n++
		line := strings.TrimRight(sc.Text(), " \t\r")
		switch {
		case strings.HasPrefix(line, "#") && cur == nil:
			continue // comment
		case line == "":
			if err := finish(); err != nil {
				return nil, err
			}
		case strings.HasPrefix(line, "name: "):
			f.Name = strings.TrimSpace(strings.TrimPrefix(line, "name: "))
		case strings.HasPrefix(line, "glyph "):
			if err := finish(); err != nil {
				return nil, err
			}
			r, err := parseRune(strings.TrimPrefix(line, "glyph "))
			if err != nil {
				return nil, fmt.Errorf("line %d: %w", n, err)
			}
			cur, start = &Glyph{Rune: r}, n
		case cur != nil:
			rows = append(rows, line)
		default:
			return nil, fmt.Errorf("line %d: unexpected %q outside a glyph", n, line)
		}
	}
	if err := finish(); err != nil {
		return nil, err
	}
	if f.Name == "" {
		return nil, fmt.Errorf("missing \"name:\" line")
	}
	if f.Glyphs[' '] == nil {
		return nil, fmt.Errorf("missing a space glyph (glyph U+0020)")
	}
	return f, nil
}

func parseRune(s string) (rune, error) {
	if strings.HasPrefix(s, "U+") {
		v, err := strconv.ParseUint(s[2:], 16, 32)
		if err != nil {
			return 0, fmt.Errorf("bad code point %q", s)
		}
		return rune(v), nil
	}
	if utf8.RuneCountInString(s) != 1 {
		return 0, fmt.Errorf("glyph name %q must be one character or U+XXXX", s)
	}
	r, _ := utf8.DecodeRuneInString(s)
	return r, nil
}

// Runes lists the font's characters in code point order.
func (f *Font) Runes() []rune {
	rs := make([]rune, 0, len(f.Glyphs))
	for r := range f.Glyphs {
		rs = append(rs, r)
	}
	slices.Sort(rs)
	return rs
}

// Has reports whether every character of s has a glyph.
func (f *Font) Has(s string) bool {
	for _, r := range s {
		if f.Glyphs[r] == nil {
			return false
		}
	}
	return true
}

// Measure is the width of s in pixels (without the trailing spacing).
func (f *Font) Measure(s string) int {
	w := 0
	for _, r := range s {
		if g := f.Glyphs[r]; g != nil {
			w += g.Advance()
		}
	}
	if w > 0 {
		w--
	}
	return w
}

// Draw paints s onto dst with its capital top-left at (x, y), each font
// pixel a scale×scale square. Characters without a glyph are skipped.
func (f *Font) Draw(dst draw.Image, s string, x, y, scale int, c color.Color) {
	src := image.NewUniform(c)
	for _, r := range f.Rects(s, x, y, scale) {
		draw.Draw(dst, r, src, image.Point{}, draw.Over)
	}
}

// Rects lists the pixel squares for s at (x, y) and scale, so callers can
// paint shadows and outlines as well as the text itself.
func (f *Font) Rects(s string, x, y, scale int) []image.Rectangle {
	var out []image.Rectangle
	for _, ch := range s {
		g := f.Glyphs[ch]
		if g == nil {
			continue
		}
		for row := range Rows {
			for col, on := range g.Bits[row] {
				if on {
					px, py := x+col*scale, y+row*scale
					out = append(out, image.Rect(px, py, px+scale, py+scale))
				}
			}
		}
		x += g.Advance() * scale
	}
	return out
}
