package pixelfont

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"slices"
	"unicode/utf16"
)

// Font units: one pixel is 100 units and the em is 10 pixels.
const (
	Unit       = 100
	UnitsPerEm = 1000
	ascender   = 800 // capital height plus one pixel of headroom
	descender  = -200
)

type point struct{ x, y int }

// contours traces the outline of a glyph's pixels: one closed loop per
// boundary, clockwise around filled areas and counter-clockwise around
// holes (TrueType's convention, y up), with straight runs merged.
func (g *Glyph) contours() [][]point {
	filled := func(r, c int) bool {
		return r >= 0 && r < Rows && c >= 0 && c < g.Width && g.Bits[r][c]
	}
	// Corner (c, r) of the grid is at x = c*Unit, y = (CapRows-r)*Unit.
	next := map[point][]point{}
	add := func(a, b point) { next[a] = append(next[a], b) }
	for r := range Rows {
		for c := range g.Width {
			if !filled(r, c) {
				continue
			}
			tl, tr := point{c, r}, point{c + 1, r}
			bl, br := point{c, r + 1}, point{c + 1, r + 1}
			if !filled(r-1, c) {
				add(tl, tr) // top edge, left to right
			}
			if !filled(r, c+1) {
				add(tr, br) // right edge, downward
			}
			if !filled(r+1, c) {
				add(br, bl) // bottom edge, right to left
			}
			if !filled(r, c-1) {
				add(bl, tl) // left edge, upward
			}
		}
	}
	// Walk the edges into loops. Where two loops touch at a corner, turn
	// right so each stays a simple polygon.
	starts := make([]point, 0, len(next))
	for p := range next {
		starts = append(starts, p)
	}
	slices.SortFunc(starts, func(a, b point) int {
		if a.y != b.y {
			return a.y - b.y
		}
		return a.x - b.x
	})
	var loops [][]point
	for _, s := range starts {
		for len(next[s]) > 0 {
			loop := []point{s}
			prev, cur := s, next[s][0]
			next[s] = next[s][1:]
			for cur != s {
				loop = append(loop, cur)
				outs := next[cur]
				pick := 0
				if len(outs) > 1 {
					dx, dy := cur.x-prev.x, cur.y-prev.y
					// Right turn in grid coordinates (y down): (dx, dy) -> (-dy, dx).
					for i, o := range outs {
						if o.x-cur.x == -dy && o.y-cur.y == dx {
							pick = i
						}
					}
				}
				prev, cur = cur, outs[pick]
				next[prev] = slices.Delete(outs, pick, pick+1)
			}
			loops = append(loops, simplify(loop))
		}
	}
	// Grid to font units.
	for _, l := range loops {
		for i, p := range l {
			l[i] = point{p.x * Unit, (CapRows - p.y) * Unit}
		}
	}
	return loops
}

// simplify drops points in the middle of straight runs.
func simplify(loop []point) []point {
	n := len(loop)
	var out []point
	for i, p := range loop {
		a, b := loop[(i+n-1)%n], loop[(i+1)%n]
		if (a.x == p.x && p.x == b.x) || (a.y == p.y && p.y == b.y) {
			continue
		}
		out = append(out, p)
	}
	return out
}

type glyphData struct {
	r       rune
	advance int
	data    []byte
	xMin    int
	bbox    [4]int // xMin, yMin, xMax, yMax
	points  int
	loops   int
	empty   bool
}

func encodeGlyph(loops [][]point) glyphData {
	gd := glyphData{loops: len(loops)}
	if len(loops) == 0 {
		gd.empty = true
		return gd
	}
	bb := [4]int{1 << 30, 1 << 30, -1 << 30, -1 << 30}
	for _, l := range loops {
		for _, p := range l {
			bb[0], bb[1] = min(bb[0], p.x), min(bb[1], p.y)
			bb[2], bb[3] = max(bb[2], p.x), max(bb[3], p.y)
			gd.points++
		}
	}
	gd.bbox, gd.xMin = bb, bb[0]
	var b bytes.Buffer
	w := func(v any) { _ = binary.Write(&b, binary.BigEndian, v) }
	w(int16(len(loops)))
	for _, v := range bb {
		w(int16(v))
	}
	end := -1
	for _, l := range loops {
		end += len(l)
		w(uint16(end))
	}
	w(uint16(0)) // no instructions
	for range gd.points {
		b.WriteByte(0x01) // on-curve, x and y as int16 deltas
	}
	var px, py int
	var ys []int16
	for _, l := range loops {
		for _, p := range l {
			w(int16(p.x - px))
			ys = append(ys, int16(p.y-py))
			px, py = p.x, p.y
		}
	}
	for _, y := range ys {
		w(y)
	}
	for b.Len()%4 != 0 {
		b.WriteByte(0)
	}
	gd.data = b.Bytes()
	return gd
}

// TTF compiles the font into a TrueType file. The output depends only on
// the source, so rebuilding it gives the same bytes.
func (f *Font) TTF() ([]byte, error) {
	runes := f.Runes()
	if len(runes) > 1000 {
		return nil, fmt.Errorf("too many glyphs")
	}
	// Glyph 0 is .notdef: a hollow box.
	notdef := &Glyph{Width: 4}
	for r := range Rows {
		notdef.Bits[r] = make([]bool, 4)
		for c := range 4 {
			notdef.Bits[r][c] = r < CapRows && (r == 0 || r == CapRows-1 || c == 0 || c == 3)
		}
	}
	glyphs := []glyphData{}
	add := func(g *Glyph) {
		gd := encodeGlyph(g.contours())
		gd.r, gd.advance = g.Rune, g.Advance()*Unit
		glyphs = append(glyphs, gd)
	}
	add(notdef)
	for _, r := range runes {
		add(f.Glyphs[r])
	}

	// glyf and loca (long offsets).
	var glyf, loca bytes.Buffer
	be := binary.BigEndian
	put := func(b *bytes.Buffer, vs ...any) {
		for _, v := range vs {
			_ = binary.Write(b, be, v)
		}
	}
	bb := [4]int{1 << 30, 1 << 30, -1 << 30, -1 << 30}
	maxPoints, maxContours, advMax := 0, 0, 0
	minLSB, minRSB, maxExtent := 1<<30, 1<<30, -1<<30
	var advSum int
	for _, g := range glyphs {
		put(&loca, uint32(glyf.Len()))
		glyf.Write(g.data)
		advMax = max(advMax, g.advance)
		advSum += g.advance
		if g.empty {
			continue
		}
		bb[0], bb[1] = min(bb[0], g.bbox[0]), min(bb[1], g.bbox[1])
		bb[2], bb[3] = max(bb[2], g.bbox[2]), max(bb[3], g.bbox[3])
		maxPoints, maxContours = max(maxPoints, g.points), max(maxContours, g.loops)
		minLSB = min(minLSB, g.xMin)
		minRSB = min(minRSB, g.advance-g.bbox[2])
		maxExtent = max(maxExtent, g.bbox[2])
	}
	put(&loca, uint32(glyf.Len()))
	n := len(glyphs)

	var head bytes.Buffer
	put(&head, uint32(0x00010000), uint32(0x00010000), uint32(0), uint32(0x5F0F3CF5),
		uint16(0x0001), uint16(UnitsPerEm),
		int64(3_870_000_000), int64(3_870_000_000), // fixed dates keep the build reproducible
		int16(bb[0]), int16(bb[1]), int16(bb[2]), int16(bb[3]),
		uint16(0), uint16(8), int16(2), int16(1), int16(0))

	var hhea bytes.Buffer
	put(&hhea, uint32(0x00010000), int16(ascender), int16(descender), int16(0),
		uint16(advMax), int16(minLSB), int16(minRSB), int16(maxExtent),
		int16(1), int16(0), int16(0), int16(0), int16(0), int16(0), int16(0),
		int16(0), uint16(n))

	var maxp bytes.Buffer
	put(&maxp, uint32(0x00010000), uint16(n), uint16(maxPoints), uint16(maxContours),
		uint16(0), uint16(0), uint16(2), uint16(0), uint16(0), uint16(0), uint16(0),
		uint16(0), uint16(0), uint16(0), uint16(0))

	var hmtx bytes.Buffer
	for _, g := range glyphs {
		put(&hmtx, uint16(g.advance), int16(g.xMin))
	}

	first, last := runes[0], runes[len(runes)-1]
	var os2 bytes.Buffer
	put(&os2, uint16(4), int16(advSum/n), uint16(400), uint16(5), uint16(0),
		int16(650), int16(600), int16(0), int16(100), int16(650), int16(600), int16(0), int16(400),
		int16(Unit), int16(300), int16(0))
	os2.Write(make([]byte, 10))                                      // panose
	put(&os2, uint32(1|1<<1|1<<31), uint32(0), uint32(0), uint32(0)) // Basic Latin, Latin-1, General Punctuation
	os2.WriteString("LORE")
	put(&os2, uint16(0x0040|0x0080), uint16(min(first, 0xFFFF)), uint16(min(last, 0xFFFF)),
		int16(ascender), int16(descender), int16(0), uint16(ascender), uint16(-descender),
		uint32(1), uint32(0), int16(500), int16(CapRows*Unit), uint16(0), uint16(' '), uint16(1))

	// cmap: format 4, one segment per character, shared by the Unicode
	// (0, 3) and Windows (3, 1) encodings.
	var sub bytes.Buffer
	segs := len(runes) + 1
	searchRange, entrySelector := 1, 0
	for searchRange*2 <= segs {
		searchRange *= 2
		entrySelector++
	}
	searchRange *= 2
	put(&sub, uint16(4), uint16(16+segs*8), uint16(0), uint16(segs*2), uint16(searchRange),
		uint16(entrySelector), uint16(segs*2-searchRange))
	for _, r := range runes {
		put(&sub, uint16(r))
	}
	put(&sub, uint16(0xFFFF), uint16(0))
	for _, r := range runes {
		put(&sub, uint16(r))
	}
	put(&sub, uint16(0xFFFF))
	for i, r := range runes {
		put(&sub, uint16((i+1-int(r))&0xFFFF))
	}
	put(&sub, uint16(1))
	for range segs {
		put(&sub, uint16(0))
	}
	var cmap bytes.Buffer
	put(&cmap, uint16(0), uint16(2), uint16(0), uint16(3), uint32(20), uint16(3), uint16(1), uint32(20))
	cmap.Write(sub.Bytes())

	family := f.Name
	ps := "LorehallowPixel-Regular"
	names := []struct {
		id uint16
		s  string
	}{
		{0, "Hand-drawn for lorehallow.com"},
		{1, family}, {2, "Regular"}, {3, ps + ";1.000"}, {4, family + " Regular"},
		{5, "Version 1.000"}, {6, ps},
		{13, "Creative Commons Attribution-NonCommercial 4.0 International"},
		{14, "https://creativecommons.org/licenses/by-nc/4.0/"},
	}
	var nameRecs, nameStr bytes.Buffer
	for _, nm := range names {
		u := utf16.Encode([]rune(nm.s))
		put(&nameRecs, uint16(3), uint16(1), uint16(0x0409), nm.id, uint16(len(u)*2), uint16(nameStr.Len()))
		for _, c := range u {
			put(&nameStr, c)
		}
	}
	var name bytes.Buffer
	put(&name, uint16(0), uint16(len(names)), uint16(6+12*len(names)))
	name.Write(nameRecs.Bytes())
	name.Write(nameStr.Bytes())

	var post bytes.Buffer
	put(&post, uint32(0x00030000), uint32(0), int16(-Unit), int16(Unit), uint32(0),
		uint32(0), uint32(0), uint32(0), uint32(0))

	return assemble(map[string][]byte{
		"OS/2": os2.Bytes(), "cmap": cmap.Bytes(), "glyf": glyf.Bytes(), "head": head.Bytes(),
		"hhea": hhea.Bytes(), "hmtx": hmtx.Bytes(), "loca": loca.Bytes(), "maxp": maxp.Bytes(),
		"name": name.Bytes(), "post": post.Bytes(),
	}), nil
}

func checksum(b []byte) uint32 {
	var sum uint32
	for i := 0; i < len(b); i += 4 {
		var w [4]byte
		copy(w[:], b[i:min(i+4, len(b))])
		sum += binary.BigEndian.Uint32(w[:])
	}
	return sum
}

// assemble writes the sfnt container and fixes head.checkSumAdjustment.
func assemble(tables map[string][]byte) []byte {
	tags := make([]string, 0, len(tables))
	for t := range tables {
		tags = append(tags, t)
	}
	slices.Sort(tags)
	n := len(tags)
	sr, es := 1, 0
	for sr*2 <= n {
		sr *= 2
		es++
	}
	var out bytes.Buffer
	be := binary.BigEndian
	hdr := make([]byte, 12+16*n)
	be.PutUint32(hdr[0:], 0x00010000)
	be.PutUint16(hdr[4:], uint16(n))
	be.PutUint16(hdr[6:], uint16(sr*16))
	be.PutUint16(hdr[8:], uint16(es))
	be.PutUint16(hdr[10:], uint16(n*16-sr*16))
	offset := len(hdr)
	var body bytes.Buffer
	headAt := 0
	for i, t := range tags {
		data := tables[t]
		rec := hdr[12+16*i:]
		copy(rec, t)
		be.PutUint32(rec[4:], checksum(data))
		be.PutUint32(rec[8:], uint32(offset+body.Len()))
		be.PutUint32(rec[12:], uint32(len(data)))
		if t == "head" {
			headAt = offset + body.Len()
		}
		body.Write(data)
		for body.Len()%4 != 0 {
			body.WriteByte(0)
		}
	}
	out.Write(hdr)
	out.Write(body.Bytes())
	b := out.Bytes()
	be.PutUint32(b[headAt+8:], 0xB1B0AFBA-checksum(b))
	return b
}
