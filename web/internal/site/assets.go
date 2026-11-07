package site

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"html/template"
	"image"
	"image/draw"
	"image/png"
	"os"
	"path"
	"path/filepath"
	"slices"
	"sort"
	"strings"
	"sync"
)

// Assets collects every fingerprinted file under /assets/ and the pixel
// image rules the pages ask for.
type Assets struct {
	artDir string // web/assets/art
	out    map[string][]byte

	mu     sync.Mutex
	images map[string]*Art // by art path ("web/hero_scene.png", "icons/section_icons.png#quest")
	urls   map[string]string
	fits   map[string]fitRule // class -> rule
}

// Art is one image: its pixels and its published URL.
type Art struct {
	Key  string
	W, H int
	URL  string
	Img  image.Image
}

type fitRule struct {
	class  string
	w, h   int
	max    int // widest it may be drawn, in CSS pixels
	minK   int
	gutter int
}

func newAssets(artDir string, out map[string][]byte) *Assets {
	return &Assets{artDir: artDir, out: out, images: map[string]*Art{}, urls: map[string]string{},
		fits: map[string]fitRule{}}
}

// Add publishes data under /assets/ with a content hash in its name and
// returns the URL. name is the readable part, e.g. "site.css".
func (a *Assets) Add(name string, data []byte) string {
	sum := sha256.Sum256(data)
	ext := path.Ext(name)
	base := strings.TrimSuffix(path.Base(name), ext)
	p := fmt.Sprintf("assets/%s.%s%s", base, hex.EncodeToString(sum[:])[:10], ext)
	a.mu.Lock()
	a.out[p] = data
	a.mu.Unlock()
	return "/" + p
}

// Image loads an art file (relative to web/assets/art), publishes it and
// returns it. "sheet.png#frame" crops one named frame from a sheet using
// its sidecar .json; "file.png#mirror" is the image followed by its mirror
// image, which tiles seamlessly in a row.
func (a *Assets) Image(key string) (*Art, error) {
	a.mu.Lock()
	if im, ok := a.images[key]; ok {
		a.mu.Unlock()
		return im, nil
	}
	a.mu.Unlock()
	file, frag, _ := strings.Cut(key, "#")
	img, err := a.decode(file)
	if err != nil {
		return nil, err
	}
	name := strings.TrimSuffix(path.Base(file), ".png")
	switch frag {
	case "":
	case "mirror":
		img = mirrorPair(img)
		name += "-pair"
	default:
		r, err := a.frame(file, frag)
		if err != nil {
			return nil, err
		}
		dst := image.NewNRGBA(image.Rect(0, 0, r.Dx(), r.Dy()))
		draw.Draw(dst, dst.Bounds(), img, r.Min, draw.Src)
		img = dst
		name += "-" + strings.ReplaceAll(frag, "_", "-")
	}
	var data []byte
	if frag == "" {
		data, err = os.ReadFile(filepath.Join(a.artDir, filepath.FromSlash(file)))
	} else {
		data, err = encodePNG(img)
	}
	if err != nil {
		return nil, err
	}
	b := img.Bounds()
	im := &Art{Key: key, W: b.Dx(), H: b.Dy(), URL: a.Add(name+".png", data), Img: img}
	a.mu.Lock()
	a.images[key] = im
	a.mu.Unlock()
	return im, nil
}

func (a *Assets) decode(file string) (image.Image, error) {
	f, err := os.Open(filepath.Join(a.artDir, filepath.FromSlash(file)))
	if err != nil {
		return nil, fmt.Errorf("art %s: %w", file, err)
	}
	defer func() { _ = f.Close() }()
	img, err := png.Decode(f)
	if err != nil {
		return nil, fmt.Errorf("art %s: %w", file, err)
	}
	return img, nil
}

// frame finds a named frame in a sheet's sidecar JSON.
func (a *Assets) frame(file, name string) (image.Rectangle, error) {
	data, err := os.ReadFile(filepath.Join(a.artDir, filepath.FromSlash(strings.TrimSuffix(file, ".png")+".json")))
	if err != nil {
		return image.Rectangle{}, fmt.Errorf("sheet %s: %w", file, err)
	}
	var side struct {
		Frames []struct {
			Name       string
			X, Y, W, H int
		}
	}
	if err := json.Unmarshal(data, &side); err != nil {
		return image.Rectangle{}, fmt.Errorf("sheet %s: %w", file, err)
	}
	for _, f := range side.Frames {
		if f.Name == name {
			return image.Rect(f.X, f.Y, f.X+f.W, f.Y+f.H), nil
		}
	}
	return image.Rectangle{}, fmt.Errorf("sheet %s has no frame %q", file, name)
}

// Frames lists a sheet's frame names in order.
func (a *Assets) Frames(file string) ([]string, error) {
	data, err := os.ReadFile(filepath.Join(a.artDir, filepath.FromSlash(strings.TrimSuffix(file, ".png")+".json")))
	if err != nil {
		return nil, err
	}
	var side struct{ Frames []struct{ Name string } }
	if err := json.Unmarshal(data, &side); err != nil {
		return nil, err
	}
	var out []string
	for _, f := range side.Frames {
		out = append(out, f.Name)
	}
	return out, nil
}

func mirrorPair(img image.Image) image.Image {
	b := img.Bounds()
	w, h := b.Dx(), b.Dy()
	src := image.NewNRGBA(image.Rect(0, 0, w, h))
	draw.Draw(src, src.Rect, img, b.Min, draw.Src)
	dst := image.NewNRGBA(image.Rect(0, 0, 2*w, h))
	for y := range h {
		for x := range w {
			px := src.Pix[y*src.Stride+x*4 : y*src.Stride+x*4+4]
			copy(dst.Pix[y*dst.Stride+x*4:], px)
			copy(dst.Pix[y*dst.Stride+(2*w-1-x)*4:], px)
		}
	}
	return dst
}

func encodePNG(img image.Image) ([]byte, error) {
	var buf bytes.Buffer
	enc := png.Encoder{CompressionLevel: png.DefaultCompression}
	if err := enc.Encode(&buf, img); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// ImgOpts describes how a pixel image is drawn.
type ImgOpts struct {
	Alt   string
	Scale int    // fixed whole-number scale; 0 means responsive (Max)
	Max   int    // responsive: the widest it may be drawn, in CSS pixels
	MinK  int    // responsive: the smallest scale (default 1)
	Eager bool   // above the fold: no lazy loading
	Class string // extra classes
}

// Img renders an <img> for pixel art: explicit width and height, always a
// whole-number multiple of the art, and pixelated scaling. Responsive
// images get a generated class that steps the scale up with the viewport.
func (a *Assets) Img(key string, o ImgOpts) (template.HTML, error) {
	im, err := a.Image(key)
	if err != nil {
		return "", err
	}
	if o.Alt == "" {
		return "", fmt.Errorf("image %s has no alt text (use \"decorative\" for decorative art)", key)
	}
	alt := o.Alt
	if alt == "decorative" {
		alt = ""
	}
	classes := []string{"px"}
	var w, h int
	if o.Scale > 0 {
		w, h = im.W*o.Scale, im.H*o.Scale
	} else {
		k := max(o.MinK, 1)
		w, h = im.W*k, im.H*k
		classes = append(classes, a.fit(im, o.Max, k))
	}
	if o.Class != "" {
		classes = append(classes, o.Class)
	}
	loading := ` loading="lazy"`
	if o.Eager {
		loading = ""
	}
	return template.HTML(fmt.Sprintf(`<img src="%s" width="%d" height="%d" alt="%s" class="%s"%s decoding="async">`,
		im.URL, w, h, template.HTMLEscapeString(alt), strings.Join(classes, " "), loading)), nil
}

func (a *Assets) fit(im *Art, maxW, minK int) string {
	sum := sha256.Sum256([]byte(fmt.Sprintf("%s|%d|%d", im.Key, maxW, minK)))
	class := "k-" + hex.EncodeToString(sum[:])[:8]
	a.mu.Lock()
	a.fits[class] = fitRule{class: class, w: im.W, h: im.H, max: maxW, minK: minK, gutter: 48}
	a.mu.Unlock()
	return class
}

// FitCSS writes the rules for every responsive pixel image the pages used:
// the largest whole-number scale that fits the viewport (less the page
// gutters) and the image's maximum width.
func (a *Assets) FitCSS() string {
	a.mu.Lock()
	defer a.mu.Unlock()
	classes := make([]string, 0, len(a.fits))
	for c := range a.fits {
		classes = append(classes, c)
	}
	sort.Strings(classes)
	type step struct {
		minVW int
		rule  string
	}
	var steps []step
	var b strings.Builder
	b.WriteString("/* Responsive pixel art: whole-number scales only (generated). */\n")
	for _, c := range classes {
		f := a.fits[c]
		fmt.Fprintf(&b, ".%s{width:%dpx;height:%dpx}\n", c, f.w*f.minK, f.h*f.minK)
		for k := f.minK + 1; f.w*k <= f.max; k++ {
			steps = append(steps, step{f.w*k + f.gutter, fmt.Sprintf(".%s{width:%dpx;height:%dpx}", c, f.w*k, f.h*k)})
		}
	}
	slices.SortStableFunc(steps, func(x, y step) int { return x.minVW - y.minVW })
	for i := 0; i < len(steps); {
		j := i
		fmt.Fprintf(&b, "@media (min-width:%dpx){", steps[i].minVW)
		for ; j < len(steps) && steps[j].minVW == steps[i].minVW; j++ {
			b.WriteString(steps[j].rule)
		}
		b.WriteString("}\n")
		i = j
	}
	return b.String()
}
