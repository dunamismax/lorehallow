package site

import (
	"bufio"
	"fmt"
	"math"
	"strconv"
	"strings"
)

// Color is one palette entry from art/style/palette.hex.
type Color struct {
	Name string // CSS token name, e.g. "lantern-gold"
	Hex  string // "#f2c45a"
}

// ParsePalette reads palette.hex: one hex color per line, optionally
// followed by "; name: description" or "; name". Every color gets a CSS
// custom property named after it (--c-<name>).
func ParsePalette(src string) ([]Color, error) {
	var out []Color
	seen := map[string]bool{}
	sc := bufio.NewScanner(strings.NewReader(src))
	n := 0
	for sc.Scan() {
		n++
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, ";") {
			continue
		}
		hex, comment, _ := strings.Cut(line, ";")
		hex = strings.ToLower(strings.TrimSpace(hex))
		if len(hex) != 6 {
			return nil, fmt.Errorf("palette line %d: %q is not a 6-digit hex color", n, hex)
		}
		if _, err := strconv.ParseUint(hex, 16, 32); err != nil {
			return nil, fmt.Errorf("palette line %d: %q is not hex", n, hex)
		}
		name, _, _ := strings.Cut(comment, ":")
		name = strings.Join(strings.Fields(strings.ToLower(name)), "-")
		if name == "" {
			return nil, fmt.Errorf("palette line %d: color %s has no name", n, hex)
		}
		if seen[name] {
			return nil, fmt.Errorf("palette line %d: duplicate name %q", n, name)
		}
		seen[name] = true
		out = append(out, Color{Name: name, Hex: "#" + hex})
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("palette is empty")
	}
	return out, nil
}

// Role is a semantic color the stylesheet uses for text or backgrounds.
// The stylesheet may only set `color` and `background-color` through roles,
// so the contrast table below covers every pair it can draw.
type Role struct {
	Name  string // --<name>
	Token string // palette name
}

// Roles of the one (dark, lantern-lit) theme.
var Roles = []Role{
	{"bg", "ink"},
	{"bg-raised", "night-plum"},
	{"bg-dusk", "deep-dusk"},
	{"bg-sky", "dusk-blue"},
	{"bg-page", "parchment-white"},
	{"bg-button", "lantern-gold"},
	{"bg-button-hover", "glow"},
	{"bg-done", "sunlit-leaf"},
	{"bg-human", "banner-red"},
	{"bg-adr", "arcane-violet"},
	{"bg-code", "sand"},
	{"text", "parchment-white"},
	{"text-muted", "pale-stone"},
	{"text-heading", "lantern-gold"},
	{"text-link", "glow"},
	{"text-accent", "amber"},
	{"text-done", "sunlit-leaf"},
	{"text-on-light", "ink"},
	{"text-heading-on-light", "bark"},
	{"text-link-on-light", "banner-red"},
	{"text-muted-on-light", "dusk-violet-grey"},
}

// Pair is a text role drawn on a background role.
type Pair struct {
	FG, BG string
	Large  bool // large text (24px+, or 19px+ bold): WCAG AA needs 3:1, not 4.5:1
}

// Pairs lists every text-on-background combination the stylesheet uses.
// The output tests check each one meets WCAG AA.
var Pairs = []Pair{
	{"text", "bg", false}, {"text", "bg-raised", false}, {"text", "bg-dusk", false}, {"text", "bg-sky", false},
	{"text-muted", "bg", false}, {"text-muted", "bg-raised", false}, {"text-muted", "bg-dusk", false},
	{"text-heading", "bg", false}, {"text-heading", "bg-raised", false}, {"text-heading", "bg-dusk", false},
	{"text-heading", "bg-sky", false},
	{"text-link", "bg", false}, {"text-link", "bg-raised", false}, {"text-link", "bg-dusk", false},
	{"text-link", "bg-sky", false},
	{"text-accent", "bg", false}, {"text-accent", "bg-raised", false}, {"text-accent", "bg-dusk", false},
	{"text-done", "bg", false}, {"text-done", "bg-raised", false},
	{"text-on-light", "bg-page", false}, {"text-heading-on-light", "bg-page", false},
	{"text-link-on-light", "bg-page", false}, {"text-muted-on-light", "bg-page", false},
	{"text-on-light", "bg-button", false}, {"text-on-light", "bg-button-hover", false},
	{"text-on-light", "bg-done", false}, {"text-on-light", "bg-code", false},
	{"text", "bg-human", false}, {"text", "bg-adr", false},
}

// Contrast is the WCAG 2 contrast ratio of two "#rrggbb" colors.
func Contrast(a, b string) float64 {
	la, lb := luminance(a), luminance(b)
	if la < lb {
		la, lb = lb, la
	}
	return (la + 0.05) / (lb + 0.05)
}

func luminance(hex string) float64 {
	v, _ := strconv.ParseUint(strings.TrimPrefix(hex, "#"), 16, 32)
	ch := func(c uint64) float64 {
		s := float64(c) / 255
		if s <= 0.04045 {
			return s / 12.92
		}
		return math.Pow((s+0.055)/1.055, 2.4)
	}
	return 0.2126*ch(v>>16&0xff) + 0.7152*ch(v>>8&0xff) + 0.0722*ch(v&0xff)
}

// Theme resolves roles to palette colors.
type Theme struct {
	Palette []Color
	byName  map[string]string
}

// NewTheme checks that every role names a palette color.
func NewTheme(palette []Color) (*Theme, error) {
	t := &Theme{Palette: palette, byName: map[string]string{}}
	for _, c := range palette {
		t.byName[c.Name] = c.Hex
	}
	for _, r := range Roles {
		if _, ok := t.byName[r.Token]; !ok {
			return nil, fmt.Errorf("role --%s uses %q, which isn't in the palette", r.Name, r.Token)
		}
	}
	return t, nil
}

// Hex is a palette color by name ("" if unknown).
func (t *Theme) Hex(name string) string { return t.byName[name] }

// RoleHex is a role's color.
func (t *Theme) RoleHex(role string) string {
	for _, r := range Roles {
		if r.Name == role {
			return t.byName[r.Token]
		}
	}
	return ""
}

// CSS writes the palette tokens and roles as custom properties.
func (t *Theme) CSS() string {
	var b strings.Builder
	b.WriteString(":root {\n\t/* Palette: generated from art/style/palette.hex. */\n")
	for _, c := range t.Palette {
		fmt.Fprintf(&b, "\t--c-%s: %s;\n", c.Name, c.Hex)
	}
	b.WriteString("\t/* Roles: the only colors text and backgrounds use (internal/site/theme.go). */\n")
	for _, r := range Roles {
		fmt.Fprintf(&b, "\t--%s: var(--c-%s);\n", r.Name, r.Token)
	}
	b.WriteString("}\n")
	return b.String()
}
