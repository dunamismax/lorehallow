// Package site builds lorehallow.com: every page, feed, image and data file,
// as an in-memory map of paths to bytes that `build` writes to web/dist/ and
// `serve` serves locally. No page has any JavaScript (ADR 0011).
package site

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"html/template"
	"image"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"sync"
	texttemplate "text/template"
	"time"

	"github.com/dunamismax/lorehallow/web/internal/buildmd"
	"github.com/dunamismax/lorehallow/web/internal/pixelfont"
	"github.com/dunamismax/lorehallow/web/internal/woff"
)

// Public addresses.
const (
	BaseURL     = "https://lorehallow.com"
	RepoURL     = "https://github.com/dunamismax/lorehallow"
	CodebergURL = "https://codeberg.org/dunamismax/lorehallow"
)

// Config says where the inputs are and how to build.
type Config struct {
	Root   string    // repository root
	Files  fs.FS     // web/: templates/ and static/ (embedded, or the directory in serve)
	Drafts bool      // include draft devlog posts (serve only)
	Git    bool      // read git history (commit, achievement feed)
	Now    time.Time // build time
	Commit string    // overrides `git rev-parse HEAD` when set
}

// Output is the built site: slash-separated paths (no leading slash) to
// file contents.
type Output struct {
	Files map[string][]byte
	// Pages lists every HTML page path, for the sitemap and the tests.
	Pages []string
}

// Write replaces dir with the site.
func (o *Output) Write(dir string) error {
	if err := os.RemoveAll(dir); err != nil {
		return err
	}
	paths := make([]string, 0, len(o.Files))
	for p := range o.Files {
		paths = append(paths, p)
	}
	slices.Sort(paths)
	for _, p := range paths {
		dst := filepath.Join(dir, filepath.FromSlash(p))
		if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(dst, o.Files[p], 0o644); err != nil {
			return err
		}
	}
	return nil
}

// Site is what every template sees as .Site.
type Site struct {
	BaseURL, RepoURL, CodebergURL, BuildMDURL string
	Commit, CommitShort                       string
	Built                                     time.Time
	Drafts                                    bool
	Progress                                  *Progress
	Posts                                     []*Post
	Tags                                      []string
	Releases                                  []Release
	CSS                                       string
	Fonts                                     map[string]string
	Icons                                     map[string]string
	Nav                                       []NavItem
	Theme                                     *Theme
}

// NavItem is one header link.
type NavItem struct{ Name, URL, Section string }

// Page is what one template renders.
type Page struct {
	Site        *Site
	Path        string // "/progress/"
	Title       string // "" on the home page
	Description string
	Section     string // header link to mark as current
	OG          string // absolute URL of the share image
	OGAlt       string
	Kicker      string // small line above the title on the share card
	Type        string // og:type
	Published   time.Time
	Data        any
}

// FullTitle is the <title>.
func (p *Page) FullTitle() string {
	if p.Title == "" {
		return "Lorehallow: a small pixel-art MMO with a shared world"
	}
	return p.Title + " · Lorehallow"
}

// Canonical is the page's absolute URL.
func (p *Page) Canonical() string { return p.Site.BaseURL + p.Path }

const cssToken = "/assets/__site_css__"

type builder struct {
	cfg    Config
	out    map[string][]byte
	assets *Assets
	md     *markdown
	font   *pixelfont.Font
	site   *Site
	base   *template.Template
	pages  []sitemapURL
	ogBase image.Image
}

var ogCache sync.Map // title|kicker|og art hash -> URL path and bytes

// Build builds the whole site.
func Build(cfg Config) (*Output, error) {
	if cfg.Now.IsZero() {
		cfg.Now = time.Now()
	}
	out := map[string][]byte{}
	b := &builder{cfg: cfg, out: out, md: newMarkdown(),
		assets: newAssets(filepath.Join(cfg.Root, "web", "assets", "art"), out)}
	if err := b.build(); err != nil {
		return nil, err
	}
	o := &Output{Files: out}
	for p := range out {
		if strings.HasSuffix(p, ".html") {
			o.Pages = append(o.Pages, p)
		}
	}
	slices.Sort(o.Pages)
	return o, nil
}

func (b *builder) build() error {
	root := b.cfg.Root
	paletteSrc, err := os.ReadFile(filepath.Join(root, "art", "style", "palette.hex"))
	if err != nil {
		return err
	}
	palette, err := ParsePalette(string(paletteSrc))
	if err != nil {
		return err
	}
	theme, err := NewTheme(palette)
	if err != nil {
		return err
	}
	fontSrc, err := fs.ReadFile(b.cfg.Files, "static/fonts/lorehallow-pixel.txt")
	if err != nil {
		return err
	}
	if b.font, err = pixelfont.Parse(string(fontSrc)); err != nil {
		return fmt.Errorf("pixel font: %w", err)
	}

	buildSrc, err := os.ReadFile(filepath.Join(root, "BUILD.md"))
	if err != nil {
		return err
	}
	plan := buildmd.Parse(string(buildSrc))
	if len(plan.Problems) > 0 {
		return fmt.Errorf("BUILD.md: %d format problems; run make buildmd-lint", len(plan.Problems))
	}
	landmarks, err := b.assets.Frames("web/progress_landmarks.png")
	if err != nil {
		return err
	}
	inline := func(s string) any { return repoLinks(b.md.inline(s)) }
	prog, err := buildProgress(root, plan, landmarks, inline, b.cfg.Git)
	if err != nil {
		return err
	}
	posts, err := loadPosts(filepath.Join(root, "web", "content", "devlog"), b.md, b.cfg.Drafts, b.assets.artDir)
	if err != nil {
		return err
	}
	releases, err := loadReleases(filepath.Join(root, "web", "data", "releases.json"))
	if err != nil {
		return err
	}
	commit := b.cfg.Commit
	if commit == "" && b.cfg.Git {
		if c, err := exec.Command("git", "-C", root, "rev-parse", "HEAD").Output(); err == nil {
			commit = strings.TrimSpace(string(c))
		}
	}
	if commit == "" {
		commit = "unknown"
	}
	tags, _ := tagIndex(posts)
	s := &Site{
		BaseURL: BaseURL, RepoURL: RepoURL, CodebergURL: CodebergURL,
		BuildMDURL: RepoURL + "/blob/main/BUILD.md",
		Commit:     commit, CommitShort: commit[:min(7, len(commit))], Built: b.cfg.Now,
		Drafts: b.cfg.Drafts, Progress: prog, Posts: posts, Tags: tags, Releases: releases,
		CSS: cssToken, Fonts: map[string]string{}, Icons: map[string]string{}, Theme: theme,
		Nav: []NavItem{
			{"Home", "/", "home"}, {"World", "/world/", "world"}, {"Progress", "/progress/", "progress"},
			{"Devlog", "/devlog/", "devlog"}, {"Downloads", "/downloads/", "downloads"}, {"About", "/about/", "about"},
		},
	}
	b.site = s
	if err := b.fonts(); err != nil {
		return err
	}
	if err := b.icons(); err != nil {
		return err
	}
	og, err := b.assets.Image("web/og_card.png")
	if err != nil {
		return err
	}
	b.ogBase = og.Img

	if b.base, err = template.New("").Funcs(b.funcs()).ParseFS(b.cfg.Files, "templates/*.html"); err != nil {
		return err
	}
	if err := b.pagesAll(); err != nil {
		return err
	}

	// The stylesheet comes last: it includes the pixel image rules the pages used.
	css, err := b.css(theme)
	if err != nil {
		return err
	}
	cssURL := b.assets.Add("site.css", css)
	for p, data := range b.out {
		if strings.HasSuffix(p, ".html") {
			b.out[p] = bytes.ReplaceAll(data, []byte(cssToken), []byte(cssURL))
		}
	}

	// Data, feeds and crawler files.
	if b.out["progress.json"], err = prog.JSON(BaseURL, b.cfg.Now, commit); err != nil {
		return err
	}
	b.out["progress.svg"] = prog.Badge(b.font, theme)
	if b.out["devlog/rss.xml"], err = rssFeed(BaseURL, posts); err != nil {
		return err
	}
	if b.out["devlog/atom.xml"], err = atomFeedXML(BaseURL, posts, b.cfg.Now); err != nil {
		return err
	}
	if b.out["sitemap.xml"], err = sitemap(BaseURL, b.pages); err != nil {
		return err
	}
	b.out["robots.txt"] = []byte("User-agent: *\nAllow: /\n\nSitemap: " + BaseURL + "/sitemap.xml\n")
	return nil
}

func (b *builder) fonts() error {
	ttf, err := b.font.TTF()
	if err != nil {
		return err
	}
	w, err := woff.FromSFNT(ttf)
	if err != nil {
		return err
	}
	b.site.Fonts["pixel"] = b.assets.Add("lorehallow-pixel.woff", w)
	for key, file := range map[string]string{"regular": "Go-Regular", "italic": "Go-Italic", "bold": "Go-Bold", "mono": "Go-Mono"} {
		data, err := fs.ReadFile(b.cfg.Files, "static/fonts/"+file+".ttf")
		if err != nil {
			return err
		}
		w, err := woff.FromSFNT(data)
		if err != nil {
			return fmt.Errorf("%s: %w", file, err)
		}
		b.site.Fonts[key] = b.assets.Add(strings.ToLower(file)+".woff", w)
	}
	return nil
}

func (b *builder) icons() error {
	for _, n := range []int{16, 32, 180, 192, 512} {
		im, err := b.assets.Image(fmt.Sprintf("icons/favicon_%d.png", n))
		if err != nil {
			return err
		}
		b.site.Icons[strconv.Itoa(n)] = im.URL
	}
	// /favicon.ico (16 and 32 px, stored as PNG) and /apple-touch-icon.png
	// for clients that ask for them without reading the page.
	var pngs [][]byte
	for _, n := range []int{16, 32} {
		data, err := os.ReadFile(filepath.Join(b.assets.artDir, "icons", fmt.Sprintf("favicon_%d.png", n)))
		if err != nil {
			return err
		}
		pngs = append(pngs, data)
	}
	b.out["favicon.ico"] = ico([]int{16, 32}, pngs)
	touch, err := os.ReadFile(filepath.Join(b.assets.artDir, "icons", "favicon_180.png"))
	if err != nil {
		return err
	}
	b.out["apple-touch-icon.png"] = touch
	return nil
}

// ico packs PNG images into an .ico file.
func ico(sizes []int, pngs [][]byte) []byte {
	var buf bytes.Buffer
	le := binary.LittleEndian
	_ = binary.Write(&buf, le, []uint16{0, 1, uint16(len(pngs))})
	offset := 6 + 16*len(pngs)
	for i, data := range pngs {
		s := byte(sizes[i])
		buf.Write([]byte{s, s, 0, 0})
		_ = binary.Write(&buf, le, []uint16{1, 32})
		_ = binary.Write(&buf, le, []uint32{uint32(len(data)), uint32(offset)})
		offset += len(data)
	}
	for _, data := range pngs {
		buf.Write(data)
	}
	return buf.Bytes()
}

func (b *builder) funcs() template.FuncMap {
	return template.FuncMap{
		// img "web/hero_scene.png" "alt text" ["k=3"] ["max=960"] ["min=2"] ["eager"] ["class=x"]
		"img": func(key, alt string, opts ...string) (template.HTML, error) {
			o := ImgOpts{Alt: alt, Max: 1200}
			for _, opt := range opts {
				k, v, _ := strings.Cut(opt, "=")
				n, _ := strconv.Atoi(v)
				switch k {
				case "k":
					o.Scale = n
				case "max":
					o.Max = n
				case "min":
					o.MinK = n
				case "eager":
					o.Eager = true
				case "class":
					o.Class = v
				default:
					return "", fmt.Errorf("img %s: unknown option %q", key, opt)
				}
			}
			return b.assets.Img(key, o)
		},
		"art": func(key string) (string, error) {
			im, err := b.assets.Image(key)
			if err != nil {
				return "", err
			}
			return im.URL, nil
		},
		"md":      func(s string) template.HTML { return repoLinks(b.md.inline(s)) },
		"date":    func(t time.Time) string { return t.Format("2 January 2006") },
		"isodate": func(t time.Time) string { return t.Format("2006-01-02") },
		"add":     func(a, c int) int { return a + c },
		"sub":     func(a, c int) int { return a - c },
		"seq": func(n int) []int {
			s := make([]int, n)
			for i := range s {
				s[i] = i
			}
			return s
		},
		"first":    func(n int, posts []*Post) []*Post { return posts[:min(n, len(posts))] },
		"firstAch": func(n int, a []Achievement) []Achievement { return a[:min(n, len(a))] },
		"human": func(n int64) string {
			switch {
			case n >= 1<<20:
				return fmt.Sprintf("%.1f MB", float64(n)/(1<<20))
			case n >= 1<<10:
				return fmt.Sprintf("%d KB", n>>10)
			}
			return fmt.Sprintf("%d bytes", n)
		},
		"pad2":  func(n int) string { return fmt.Sprintf("%02d", n) },
		"lower": strings.ToLower,
		"dict": func(kv ...any) map[string]any {
			m := map[string]any{}
			for i := 0; i+1 < len(kv); i += 2 {
				m[kv[i].(string)] = kv[i+1]
			}
			return m
		},
	}
}

// render writes one page. tmpl is a file in templates/pages/.
func (b *builder) render(tmpl string, p *Page) error {
	p.Site = b.site
	if p.Type == "" {
		p.Type = "website"
	}
	if p.Kicker == "" {
		p.Kicker = "Lorehallow"
	}
	title := p.Title
	if title == "" {
		title = "A small pixel-art MMO with a shared world"
	}
	og, err := b.og(title, p.Kicker)
	if err != nil {
		return err
	}
	p.OG = BaseURL + og
	p.OGAlt = "Lorehallow: " + title + ". A lantern-lit path through pixel-art hills to a village."
	t, err := b.base.Clone()
	if err != nil {
		return err
	}
	if _, err := t.ParseFS(b.cfg.Files, "templates/pages/"+tmpl+".html"); err != nil {
		return err
	}
	var buf bytes.Buffer
	if err := t.ExecuteTemplate(&buf, "layout", p); err != nil {
		return fmt.Errorf("%s: %w", p.Path, err)
	}
	file := strings.TrimPrefix(p.Path, "/")
	switch {
	case p.Path == "/404.html":
	case strings.HasSuffix(file, "/") || file == "":
		file += "index.html"
	}
	b.out[file] = tidy(buf.Bytes())
	if p.Path != "/404.html" {
		lastmod := ""
		if !p.Published.IsZero() {
			lastmod = p.Published.Format("2006-01-02")
		}
		b.pages = append(b.pages, sitemapURL{Loc: p.Path, LastMod: lastmod})
	}
	return nil
}

var blankLinesRE = regexp.MustCompile(`\n[ \t]*\n(?:[ \t]*\n)+`)

// tidy collapses the runs of blank lines templates leave behind.
func tidy(h []byte) []byte { return blankLinesRE.ReplaceAll(h, []byte("\n\n")) }

func (b *builder) og(title, kicker string) (string, error) {
	key := title + "|" + kicker
	if v, ok := ogCache.Load(key); ok {
		e := v.([2]any)
		url := e[0].(string)
		b.out[strings.TrimPrefix(url, "/")] = e[1].([]byte)
		return url, nil
	}
	img := ogImage(b.ogBase, b.font, b.site.Theme, title, kicker)
	data, err := encodePNG(img)
	if err != nil {
		return "", err
	}
	slug := strings.Trim(regexp.MustCompile(`[^a-z0-9]+`).ReplaceAllString(strings.ToLower(title), "-"), "-")
	url := b.assets.Add("og-"+slug[:min(40, len(slug))]+".png", data)
	ogCache.Store(key, [2]any{url, data})
	return url, nil
}

func (b *builder) pagesAll() error {
	s := b.site
	pr := s.Progress
	posts := s.Posts
	type tagPage struct {
		Tag   string
		Posts []*Post
	}
	if err := b.render("home", &Page{Path: "/", Section: "home",
		Description: "Lorehallow is a small pixel-art fantasy MMO built around a shared world, with a Go server built in the open. Follow the build."}); err != nil {
		return err
	}
	if err := b.render("world", &Page{Path: "/world/", Title: "The world", Section: "world",
		Description: "Mossy forests, lantern-lit towns and old stone ruins: the setting and mood of Lorehallow, early and subject to change."}); err != nil {
		return err
	}
	desc := fmt.Sprintf("Lorehallow is %d%% built: %d of %d steps in the build plan are done.", pr.Percent, pr.Done, pr.Total)
	if pr.Current != nil {
		desc += fmt.Sprintf(" Now on Phase %d: %s.", pr.Current.Number, pr.Current.Title)
	}
	if err := b.render("progress", &Page{Path: "/progress/", Title: "Progress", Section: "progress",
		Description: desc, Kicker: fmt.Sprintf("%d%% built", pr.Percent), Data: pr.BadgeWidth(b.font)}); err != nil {
		return err
	}
	for _, p := range pr.Phases {
		if err := b.render("phase", &Page{Path: p.URL, Title: fmt.Sprintf("Phase %d: %s", p.Number, p.Title),
			Section: "progress", Kicker: fmt.Sprintf("Progress · %d of %d done", p.Done, p.Total),
			Description: p.Meta["Goal"], Data: p}); err != nil {
			return err
		}
	}
	if err := b.render("devlog", &Page{Path: "/devlog/", Title: "Devlog", Section: "devlog",
		Description: "Notes from building Lorehallow in the open: the server, the pixel art and the world.", Data: posts}); err != nil {
		return err
	}
	tags, byTag := tagIndex(posts)
	for _, t := range tags {
		if err := b.render("tag", &Page{Path: "/devlog/tags/" + t + "/", Title: "Devlog: " + t, Section: "devlog",
			Description: "Lorehallow devlog posts tagged " + t + ".", Data: tagPage{t, byTag[t]}}); err != nil {
			return err
		}
	}
	for i, p := range posts {
		var newer, older *Post
		if i > 0 {
			newer = posts[i-1]
		}
		if i+1 < len(posts) {
			older = posts[i+1]
		}
		if err := b.render("post", &Page{Path: p.URL, Title: p.Title, Section: "devlog", Kicker: "Devlog · " + p.Date.Format("2 Jan 2006"),
			Description: p.Summary, Type: "article", Published: p.Date,
			Data: map[string]*Post{"Post": p, "Newer": newer, "Older": older}}); err != nil {
			return err
		}
	}
	if err := b.render("downloads", &Page{Path: "/downloads/", Title: "Downloads", Section: "downloads",
		Description: "Lorehallow builds for Windows, macOS and Linux. Nothing is available yet; follow the progress."}); err != nil {
		return err
	}
	if err := b.render("about", &Page{Path: "/about/", Title: "About", Section: "about",
		Description: "Why Lorehallow exists, how it's made, and the tech behind it."}); err != nil {
		return err
	}
	if err := b.render("faq", &Page{Path: "/faq/", Title: "FAQ", Section: "faq",
		Description: "Is Lorehallow free? When can I play? How many players? Answers to common questions."}); err != nil {
		return err
	}
	return b.render("404", &Page{Path: "/404.html", Title: "Lost in the woods",
		Description: "This page doesn't exist."})
}

// css renders static/css/site.css (a text/template) with the palette, the
// fonts, art URLs and generated rules.
func (b *builder) css(theme *Theme) ([]byte, error) {
	src, err := fs.ReadFile(b.cfg.Files, "static/css/site.css")
	if err != nil {
		return nil, err
	}
	t, err := texttemplate.New("site.css").Funcs(texttemplate.FuncMap{
		"art": func(key string) (string, error) {
			im, err := b.assets.Image(key)
			if err != nil {
				return "", err
			}
			return im.URL, nil
		},
		"font": func(k string) string { return b.site.Fonts[k] },
	}).Parse(string(src))
	if err != nil {
		return nil, err
	}
	var buf bytes.Buffer
	buf.WriteString("/* lorehallow.com. Built from web/static/css/site.css by cmd/lorehallow-web. */\n")
	buf.WriteString(theme.CSS())
	percents := make([]int, 101)
	for i := range percents {
		percents[i] = i
	}
	data := map[string]any{"QuestMap": questMapCSS(len(b.site.Progress.Phases)), "Fireflies": fireflies, "Percents": percents}
	if err := t.Execute(&buf, data); err != nil {
		return nil, err
	}
	buf.WriteString(b.assets.FitCSS())
	return buf.Bytes(), nil
}

// fireflies are placed by hand in art pixels around the hero's center:
// x, height above the bottom, drift and blink seconds, and their delays.
var fireflies = [][6]int{
	{-210, 34, 9, 3, 1, 0}, {-172, 58, 11, 4, 4, 1}, {-140, 22, 8, 3, 2, 2}, {-104, 70, 12, 5, 7, 3},
	{-76, 40, 10, 3, 3, 1}, {-38, 64, 9, 4, 6, 2}, {-12, 30, 13, 3, 9, 0}, {22, 52, 10, 5, 2, 4},
	{58, 26, 11, 3, 5, 1}, {92, 74, 9, 4, 8, 2}, {126, 44, 12, 3, 1, 0}, {164, 30, 10, 4, 6, 3},
	{196, 62, 8, 3, 3, 1}, {232, 36, 11, 5, 4, 2},
}

// questMapCSS lays the landmarks out along a winding road: one column on
// phones, then rows that turn back on themselves (4 and then 7 across).
func questMapCSS(n int) string {
	var s strings.Builder
	for _, l := range []struct{ minW, cols int }{{640, 4}, {1024, 7}} {
		fmt.Fprintf(&s, "@media (min-width:%dpx){\n.quest-map ol{grid-template-columns:repeat(%d,1fr)}\n", l.minW, l.cols)
		for i := range n {
			row, pos := i/l.cols, i%l.cols
			col := pos
			if row%2 == 1 {
				col = l.cols - 1 - pos
			}
			fmt.Fprintf(&s, ".quest-map .qm-%d{grid-row:%d;grid-column:%d}", i, row+1, col+1)
			if i == n-1 {
				fmt.Fprintf(&s, ".quest-map .qm-%d::before{display:none}", i)
			} else if (i+1)/l.cols != row {
				fmt.Fprintf(&s, ".quest-map .qm-%d::before{left:calc(50%% - var(--road)/2);top:var(--road-y);width:var(--road);height:calc(100%% + var(--gap))}", i)
			} else if row%2 == 0 {
				fmt.Fprintf(&s, ".quest-map .qm-%d::before{left:50%%;top:calc(var(--road-y) - var(--road)/2);width:calc(100%% + var(--gap));height:var(--road)}", i)
			} else {
				fmt.Fprintf(&s, ".quest-map .qm-%d::before{right:50%%;left:auto;top:calc(var(--road-y) - var(--road)/2);width:calc(100%% + var(--gap));height:var(--road)}", i)
			}
			s.WriteByte('\n')
		}
		s.WriteString("}\n")
	}
	return s.String()
}

// Release is one entry in web/data/releases.json (schema in web/data/README.md).
type Release struct {
	Version string        `json:"version"`
	Date    string        `json:"date"`
	Channel string        `json:"channel"`
	Notes   string        `json:"notes"`
	Files   []ReleaseFile `json:"files"`
}

// ReleaseFile is one download.
type ReleaseFile struct {
	Platform string `json:"platform"`
	URL      string `json:"url"`
	Size     int64  `json:"size"`
	SHA256   string `json:"sha256"`
}

var (
	versionRE = regexp.MustCompile(`^v\d+\.\d+\.\d+$`)
	sha256RE  = regexp.MustCompile(`^[0-9a-f]{64}$`)
)

// Platforms the downloads page shows, in order.
var Platforms = []string{"windows", "macos", "linux"}

func loadReleases(file string) ([]Release, error) {
	data, err := os.ReadFile(file)
	if err != nil {
		return nil, err
	}
	var doc struct {
		Releases []Release `json:"releases"`
	}
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&doc); err != nil {
		return nil, fmt.Errorf("releases.json: %w", err)
	}
	for i, r := range doc.Releases {
		where := fmt.Sprintf("releases.json release %d", i)
		if !versionRE.MatchString(r.Version) {
			return nil, fmt.Errorf("%s: version %q must look like v0.2.0", where, r.Version)
		}
		if _, err := time.Parse("2006-01-02", r.Date); err != nil {
			return nil, fmt.Errorf("%s: date must be YYYY-MM-DD", where)
		}
		if !slices.Contains([]string{"alpha", "beta", "stable"}, r.Channel) {
			return nil, fmt.Errorf("%s: channel must be alpha, beta or stable", where)
		}
		if len(r.Files) == 0 {
			return nil, fmt.Errorf("%s: no files", where)
		}
		for _, f := range r.Files {
			if !slices.Contains(Platforms, f.Platform) {
				return nil, fmt.Errorf("%s: platform %q must be one of %v", where, f.Platform, Platforms)
			}
			if !strings.HasPrefix(f.URL, "https://") {
				return nil, fmt.Errorf("%s: url must be https", where)
			}
			if f.Size <= 0 || !sha256RE.MatchString(f.SHA256) {
				return nil, fmt.Errorf("%s: each file needs a size and a lowercase sha256", where)
			}
		}
	}
	return doc.Releases, nil
}

// Latest returns the newest release's file for a platform, if any.
func (s *Site) Latest(platform string) map[string]any {
	for _, r := range s.Releases {
		for _, f := range r.Files {
			if f.Platform == platform {
				return map[string]any{"Release": r, "File": f}
			}
		}
	}
	return nil
}

// PlatformList is Platforms for templates.
func (s *Site) PlatformList() []string { return Platforms }
