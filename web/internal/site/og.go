package site

import (
	"image"
	"image/color"
	"image/draw"

	"github.com/dunamismax/lorehallow/web/internal/pixelfont"
)

// OG card: the 240×126 og_card art at exactly 5x (1200×630), with the page
// title drawn on a plaque in the hand-drawn pixel font. Everything is placed
// on the art's own pixel grid, so the card stays true pixel art.
const ogScale = 5

func ogImage(base image.Image, f *pixelfont.Font, t *Theme, title, kicker string) image.Image {
	b := base.Bounds()
	art := image.NewNRGBA(image.Rect(0, 0, b.Dx(), b.Dy()))
	draw.Draw(art, art.Bounds(), base, b.Min, draw.Src)

	ink, plum := hexColor(t.Hex("ink")), hexColor(t.Hex("night-plum"))
	gold, paper := hexColor(t.Hex("lantern-gold")), hexColor(t.Hex("parchment-white"))
	edge := hexColor(t.Hex("dusk-violet-grey"))

	// Wrap the title to fit the plaque (at most two lines).
	const maxW = 132
	lines := wrapPixel(f, title, maxW)
	if len(lines) > 2 {
		lines = lines[:2]
	}
	w := f.Measure(kicker)
	for _, l := range lines {
		w = max(w, f.Measure(l))
	}
	const x0, y0, pad = 8, 8, 5
	h := pad + pixelfont.CapRows + 4 + len(lines)*(pixelfont.Rows+2) + pad - 2
	plaque := image.Rect(x0, y0, x0+w+2*pad, y0+h)
	fill := func(r image.Rectangle, c color.Color) {
		draw.Draw(art, r, image.NewUniform(c), image.Point{}, draw.Src)
	}
	fill(plaque.Inset(-1), ink)
	fill(plaque, plum)
	fill(image.Rect(plaque.Min.X, plaque.Max.Y-1, plaque.Max.X, plaque.Max.Y), edge)
	fill(image.Rect(plaque.Min.X+1, plaque.Max.Y, plaque.Max.X+1, plaque.Max.Y+1), ink) // drop shadow
	fill(image.Rect(plaque.Max.X+1, plaque.Min.Y+1, plaque.Max.X+2, plaque.Max.Y+1), ink)

	y := y0 + pad
	f.Draw(art, kicker, x0+pad, y, 1, gold)
	y += pixelfont.CapRows + 4
	for _, l := range lines {
		f.Draw(art, l, x0+pad, y, 1, paper)
		y += pixelfont.Rows + 2
	}

	return upscale(art, ogScale)
}

// upscale enlarges an image by a whole number, nearest neighbor.
func upscale(src *image.NRGBA, k int) *image.NRGBA {
	w, h := src.Rect.Dx(), src.Rect.Dy()
	out := image.NewNRGBA(image.Rect(0, 0, w*k, h*k))
	for y := range h * k {
		srow := src.Pix[(y/k)*src.Stride:]
		drow := out.Pix[y*out.Stride:]
		for x := range w * k {
			copy(drow[x*4:x*4+4], srow[(x/k)*4:(x/k)*4+4])
		}
	}
	return out
}

// wrapPixel breaks text into lines no wider than maxW pixels, dropping any
// characters the font doesn't have.
func wrapPixel(f *pixelfont.Font, s string, maxW int) []string {
	var clean []rune
	for _, r := range s {
		if f.Glyphs[r] != nil {
			clean = append(clean, r)
		}
	}
	var lines []string
	line := ""
	word := ""
	flushWord := func() {
		if word == "" {
			return
		}
		try := word
		if line != "" {
			try = line + " " + word
		}
		if f.Measure(try) > maxW && line != "" {
			lines = append(lines, line)
			line = word
		} else {
			line = try
		}
		word = ""
	}
	for _, r := range clean {
		if r == ' ' {
			flushWord()
			continue
		}
		word += string(r)
	}
	flushWord()
	if line != "" {
		lines = append(lines, line)
	}
	return lines
}

func hexColor(h string) color.NRGBA {
	var c color.NRGBA
	c.A = 255
	if len(h) == 7 {
		var v uint32
		for _, ch := range h[1:] {
			v <<= 4
			switch {
			case ch >= '0' && ch <= '9':
				v |= uint32(ch - '0')
			case ch >= 'a' && ch <= 'f':
				v |= uint32(ch-'a') + 10
			}
		}
		c.R, c.G, c.B = uint8(v>>16), uint8(v>>8), uint8(v)
	}
	return c
}
