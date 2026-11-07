package pixelfont

import (
	"bytes"
	"os"
	"testing"
)

func load(t *testing.T) *Font {
	t.Helper()
	src, err := os.ReadFile("../../static/fonts/lorehallow-pixel.txt")
	if err != nil {
		t.Fatal(err)
	}
	f, err := Parse(string(src))
	if err != nil {
		t.Fatal(err)
	}
	return f
}

func TestRealFontCoversHeadings(t *testing.T) {
	f := load(t)
	all := "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789 !?.,:;'\"-()/&+=%#*<>_@$’‘“”–—…·×→←[]"
	for _, r := range all {
		if f.Glyphs[r] == nil {
			t.Errorf("no glyph for %q", r)
		}
	}
}

func TestTTFIsReproducible(t *testing.T) {
	f := load(t)
	a, err := f.TTF()
	if err != nil {
		t.Fatal(err)
	}
	b, _ := f.TTF()
	if !bytes.Equal(a, b) {
		t.Fatal("two builds differ")
	}
	if checksum(a) != 0xB1B0AFBA {
		t.Fatalf("whole-font checksum %#x; want 0xB1B0AFBA", checksum(a))
	}
}

func TestContours(t *testing.T) {
	tests := []struct {
		name   string
		rows   []string
		loops  int
		points int
	}{
		{"square", []string{"##", "##"}, 1, 4},
		{"ring", []string{"###", "#.#", "###"}, 2, 8},
		{"diagonal", []string{"#.", ".#"}, 2, 8},
		{"L", []string{"#.", "##"}, 1, 6},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := &Glyph{Width: len(tt.rows[0])}
			for i := range Rows {
				g.Bits[i] = make([]bool, g.Width)
				if i < len(tt.rows) {
					for j, c := range tt.rows[i] {
						g.Bits[i][j] = c == '#'
					}
				}
			}
			loops := g.contours()
			pts := 0
			for _, l := range loops {
				pts += len(l)
			}
			if len(loops) != tt.loops || pts != tt.points {
				t.Fatalf("got %d loops, %d points; want %d, %d", len(loops), pts, tt.loops, tt.points)
			}
		})
	}
}

func TestParseErrors(t *testing.T) {
	tests := map[string]string{
		"no name":    "glyph U+0020\n.\n.\n.\n.\n.\n.\n.\n",
		"short":      "name: X\nglyph U+0020\n.\n.\n",
		"ragged":     "name: X\nglyph U+0020\n..\n.\n.\n.\n.\n.\n.\n",
		"bad char":   "name: X\nglyph U+0020\n.\n.\n.\n.\n.\n.\nx\n",
		"duplicate":  "name: X\nglyph a\n.\n.\n.\n.\n.\n.\n.\n\nglyph a\n.\n.\n.\n.\n.\n.\n.\n",
		"long name":  "name: X\nglyph ab\n",
		"no space":   "name: X\nglyph a\n.\n.\n.\n.\n.\n.\n.\n",
		"stray text": "name: X\nhello\n",
	}
	for name, src := range tests {
		if _, err := Parse(src); err == nil {
			t.Errorf("%s: no error", name)
		}
	}
}

func TestMeasureAndRects(t *testing.T) {
	f := load(t)
	if w := f.Measure("Hi"); w != 5+1+1 {
		t.Fatalf("Measure(Hi) = %d", w)
	}
	if n := len(f.Rects("I", 0, 0, 2)); n != 11 {
		t.Fatalf("I has %d pixels", n)
	}
}
