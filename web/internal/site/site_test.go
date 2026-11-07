package site

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"image/png"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/dunamismax/lorehallow/web"
	"golang.org/x/net/html"
)

// The real site, built once from the repository (drafts left out, as in
// production) and shared by the output tests.
var (
	realOnce sync.Once
	realSite *Output
	realErr  error
)

func built(t *testing.T) *Output {
	t.Helper()
	realOnce.Do(func() {
		realSite, realErr = Build(Config{Root: "../../..", Files: web.Files, Git: true,
			Now: time.Date(2026, 11, 7, 12, 0, 0, 0, time.UTC)})
	})
	if realErr != nil {
		t.Fatal(realErr)
	}
	return realSite
}

func parse(t *testing.T, name string, data []byte) *html.Node {
	t.Helper()
	doc, err := html.Parse(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("%s: %v", name, err)
	}
	return doc
}

func walk(n *html.Node, fn func(*html.Node)) {
	fn(n)
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		walk(c, fn)
	}
}

func attr(n *html.Node, key string) (string, bool) {
	for _, a := range n.Attr {
		if a.Key == key {
			return a.Val, true
		}
	}
	return "", false
}

func TestNoScriptsAnywhere(t *testing.T) {
	o := built(t)
	for _, p := range o.Pages {
		walk(parse(t, p, o.Files[p]), func(n *html.Node) {
			if n.Type != html.ElementNode {
				return
			}
			if n.Data == "script" || n.Data == "iframe" || n.Data == "object" || n.Data == "embed" {
				t.Errorf("%s: <%s> found; the site ships no scripts or embeds", p, n.Data)
			}
			for _, a := range n.Attr {
				if strings.HasPrefix(strings.ToLower(a.Key), "on") {
					t.Errorf("%s: <%s %s> is an inline event handler", p, n.Data, a.Key)
				}
				if a.Key == "style" {
					t.Errorf("%s: <%s style> is refused by the CSP", p, n.Data)
				}
				if v := strings.ToLower(strings.TrimSpace(a.Val)); strings.HasPrefix(v, "javascript:") {
					t.Errorf("%s: <%s %s=%q> is a javascript: URL", p, n.Data, a.Key, a.Val)
				}
			}
		})
	}
	for p, data := range o.Files {
		if strings.HasSuffix(p, ".svg") && bytes.Contains(bytes.ToLower(data), []byte("<script")) {
			t.Errorf("%s contains a script", p)
		}
	}
}

func TestImagesHaveAltAndWholeNumberSizes(t *testing.T) {
	o := built(t)
	for _, p := range o.Pages {
		walk(parse(t, p, o.Files[p]), func(n *html.Node) {
			if n.Type != html.ElementNode || n.Data != "img" {
				return
			}
			src, _ := attr(n, "src")
			if _, ok := attr(n, "alt"); !ok {
				t.Errorf("%s: <img src=%s> has no alt attribute", p, src)
			}
			ws, okW := attr(n, "width")
			hs, okH := attr(n, "height")
			if !okW || !okH {
				t.Errorf("%s: <img src=%s> has no explicit width and height", p, src)
				return
			}
			w, _ := strconv.Atoi(ws)
			h, _ := strconv.Atoi(hs)
			data, ok := o.Files[strings.TrimPrefix(src, "/")]
			if !ok {
				t.Errorf("%s: <img src=%s> doesn't exist", p, src)
				return
			}
			var iw, ih int
			if strings.HasSuffix(src, ".png") {
				cfg, err := png.DecodeConfig(bytes.NewReader(data))
				if err != nil {
					t.Errorf("%s: %s: %v", p, src, err)
					return
				}
				iw, ih = cfg.Width, cfg.Height
			} else if strings.HasSuffix(src, ".svg") {
				m := regexp.MustCompile(`width="(\d+)" height="(\d+)"`).FindSubmatch(data)
				iw, _ = strconv.Atoi(string(m[1]))
				ih, _ = strconv.Atoi(string(m[2]))
			}
			if iw == 0 || w%iw != 0 || h%ih != 0 || w/iw != h/ih {
				t.Errorf("%s: <img src=%s> is %dx%d, not a whole-number multiple of %dx%d", p, src, w, h, iw, ih)
			}
		})
	}
}

// Every responsive pixel image rule scales by whole numbers.
func TestFitRulesAreWholeNumberScales(t *testing.T) {
	o := built(t)
	css := siteCSS(t, o)
	rule := regexp.MustCompile(`\.(k-[0-9a-f]{8})\{width:(\d+)px;height:(\d+)px\}`)
	base := map[string][2]int{}
	for _, m := range rule.FindAllStringSubmatch(css, -1) {
		w, _ := strconv.Atoi(m[2])
		h, _ := strconv.Atoi(m[3])
		if b, ok := base[m[1]]; !ok {
			base[m[1]] = [2]int{w, h}
		} else if w%b[0] != 0 || h%b[1] != 0 || w/b[0] != h/b[1] {
			t.Errorf("%s: %dx%d is not a whole-number multiple of %dx%d", m[1], w, h, b[0], b[1])
		}
	}
	if len(base) == 0 {
		t.Fatal("no responsive image rules found")
	}
}

func siteCSS(t *testing.T, o *Output) string {
	t.Helper()
	for p, data := range o.Files {
		if strings.HasPrefix(p, "assets/site.") && strings.HasSuffix(p, ".css") {
			return string(data)
		}
	}
	t.Fatal("no stylesheet in the output")
	return ""
}

func TestInternalLinksResolve(t *testing.T) {
	o := built(t)
	exists := func(target string) bool {
		target = strings.SplitN(strings.SplitN(target, "#", 2)[0], "?", 2)[0]
		p := strings.TrimPrefix(target, "/")
		if p == "" || strings.HasSuffix(p, "/") {
			p += "index.html"
		}
		_, ok := o.Files[p]
		return ok
	}
	ids := map[string]map[string]bool{}
	for _, p := range o.Pages {
		ids[p] = map[string]bool{}
		walk(parse(t, p, o.Files[p]), func(n *html.Node) {
			if id, ok := attr(n, "id"); ok {
				ids[p][id] = true
			}
		})
	}
	for _, p := range o.Pages {
		walk(parse(t, p, o.Files[p]), func(n *html.Node) {
			if n.Type != html.ElementNode {
				return
			}
			for _, key := range []string{"href", "src"} {
				v, ok := attr(n, key)
				if !ok {
					continue
				}
				if strings.HasPrefix(v, BaseURL+"/") {
					v = strings.TrimPrefix(v, BaseURL)
				}
				switch {
				case strings.HasPrefix(v, "#"):
					if !ids[p][v[1:]] {
						t.Errorf("%s: %s=%q points at a missing id", p, key, v)
					}
				case strings.HasPrefix(v, "/"):
					if !exists(v) {
						t.Errorf("%s: %s=%q doesn't exist in the site", p, key, v)
						continue
					}
					if base, frag, ok := strings.Cut(v, "#"); ok {
						target := strings.TrimPrefix(base, "/") + "index.html"
						if ids[target] != nil && !ids[target][frag] {
							t.Errorf("%s: %s=%q: no id %q on that page", p, key, v, frag)
						}
					}
				case strings.HasPrefix(v, "http://"), strings.HasPrefix(v, "https://"), strings.HasPrefix(v, "mailto:"):
				default:
					t.Errorf("%s: %s=%q is a relative link; use a root-relative one", p, key, v)
				}
			}
		})
	}
	// CSS references.
	css := siteCSS(t, o)
	for _, m := range regexp.MustCompile(`url\("?(/[^")]+)"?\)`).FindAllStringSubmatch(css, -1) {
		if !exists(m[1]) {
			t.Errorf("site.css: url(%s) doesn't exist", m[1])
		}
	}
}

// No page may load anything from another origin.
func TestNoThirdPartyRequests(t *testing.T) {
	o := built(t)
	for _, p := range o.Pages {
		walk(parse(t, p, o.Files[p]), func(n *html.Node) {
			if n.Type != html.ElementNode {
				return
			}
			check := func(key string) {
				if v, ok := attr(n, key); ok && (strings.Contains(v, "://") || strings.HasPrefix(v, "//")) {
					t.Errorf("%s: <%s %s=%q> loads from another origin", p, n.Data, key, v)
				}
			}
			switch n.Data {
			case "img", "source", "video", "audio":
				check("src")
				check("srcset")
			case "link":
				if rel, _ := attr(n, "rel"); rel != "canonical" && rel != "alternate" {
					check("href")
				}
			}
		})
	}
	if regexp.MustCompile(`url\("?(https?:)?//`).MatchString(siteCSS(t, o)) {
		t.Error("site.css loads from another origin")
	}
}

func TestLandingPageUnderOneMegabyte(t *testing.T) {
	o := built(t)
	page := o.Files["index.html"]
	total := len(page)
	seen := map[string]bool{}
	add := func(u string) {
		p := strings.TrimPrefix(u, "/")
		if seen[p] || !strings.HasPrefix(u, "/") {
			return
		}
		seen[p] = true
		total += len(o.Files[p])
	}
	walk(parse(t, "index.html", page), func(n *html.Node) {
		if n.Type != html.ElementNode {
			return
		}
		switch n.Data {
		case "img":
			v, _ := attr(n, "src")
			add(v)
		case "link":
			rel, _ := attr(n, "rel")
			if rel == "stylesheet" || rel == "preload" || rel == "icon" {
				v, _ := attr(n, "href")
				add(v)
			}
		}
	})
	for _, m := range regexp.MustCompile(`url\("?(/[^")]+)"?\)`).FindAllStringSubmatch(siteCSS(t, o), -1) {
		add(m[1]) // every font and background the stylesheet can load
	}
	t.Logf("landing page: %d KB with everything it can load", total>>10)
	if total > 1<<20 {
		t.Errorf("landing page is %d KB; the budget is 1024 KB", total>>10)
	}
}

func TestContrastMeetsWCAGAA(t *testing.T) {
	src, err := os.ReadFile("../../../art/style/palette.hex")
	if err != nil {
		t.Fatal(err)
	}
	pal, err := ParsePalette(string(src))
	if err != nil {
		t.Fatal(err)
	}
	theme, err := NewTheme(pal)
	if err != nil {
		t.Fatal(err)
	}
	roles := map[string]bool{}
	for _, r := range Roles {
		roles[r.Name] = true
	}
	for _, p := range Pairs {
		if !roles[p.FG] || !roles[p.BG] {
			t.Errorf("pair %s on %s names an unknown role", p.FG, p.BG)
			continue
		}
		need := 4.5
		if p.Large {
			need = 3
		}
		if c := Contrast(theme.RoleHex(p.FG), theme.RoleHex(p.BG)); c < need {
			t.Errorf("--%s on --%s is %.2f:1; WCAG AA needs %.1f:1", p.FG, p.BG, c, need)
		}
	}
}

// The stylesheet sets text and background colors only through roles, so
// the contrast table above covers everything it draws.
func TestStylesheetUsesColorRoles(t *testing.T) {
	src, err := os.ReadFile("../../static/css/site.css")
	if err != nil {
		t.Fatal(err)
	}
	roles := map[string]bool{}
	fgs := map[string]bool{}
	for _, r := range Roles {
		roles[r.Name] = true
	}
	for _, p := range Pairs {
		fgs[p.FG] = true
	}
	decl := regexp.MustCompile(`(?m)(?:^|[;{\s])(color|background-color)\s*:\s*([^;}]+)`)
	for _, m := range decl.FindAllStringSubmatch(string(src), -1) {
		v := strings.TrimSpace(m[2])
		if v == "transparent" || v == "inherit" || v == "currentColor" {
			continue
		}
		r := regexp.MustCompile(`^var\(--([a-z-]+)\)$`).FindStringSubmatch(v)
		if r == nil || !roles[r[1]] {
			t.Errorf("site.css: %s: %s; use a role variable from theme.go", m[1], v)
			continue
		}
		if m[1] == "color" && !fgs[r[1]] {
			t.Errorf("site.css: color: var(--%s) is in no contrast pair", r[1])
		}
	}
}

func TestEveryPageHasMetadata(t *testing.T) {
	o := built(t)
	for _, p := range o.Pages {
		doc := parse(t, p, o.Files[p])
		found := map[string]string{}
		walk(doc, func(n *html.Node) {
			if n.Type != html.ElementNode {
				return
			}
			switch n.Data {
			case "title":
				if n.FirstChild != nil {
					found["title"] = n.FirstChild.Data
				}
			case "link":
				if rel, _ := attr(n, "rel"); rel == "canonical" {
					found["canonical"], _ = attr(n, "href")
				}
			case "meta":
				k, _ := attr(n, "name")
				if k == "" {
					k, _ = attr(n, "property")
				}
				found[k], _ = attr(n, "content")
			case "html":
				found["lang"], _ = attr(n, "lang")
			case "a":
				if h, _ := attr(n, "href"); h == RepoURL {
					found["github"] = h
				} else if h == CodebergURL {
					found["codeberg"] = h
				}
			}
		})
		for _, k := range []string{"title", "canonical", "description", "og:title", "og:image", "og:url", "lang", "github", "codeberg"} {
			if found[k] == "" {
				t.Errorf("%s: missing %s", p, k)
			}
		}
		if c := found["canonical"]; c != "" && !strings.HasPrefix(c, BaseURL+"/") {
			t.Errorf("%s: canonical %q isn't absolute", p, c)
		}
		if img := found["og:image"]; img != "" {
			data := o.Files[strings.TrimPrefix(img, BaseURL+"/")]
			cfg, err := png.DecodeConfig(bytes.NewReader(data))
			if err != nil || cfg.Width != 1200 || cfg.Height != 630 {
				t.Errorf("%s: og:image %s isn't a 1200x630 PNG", p, img)
			}
		}
	}
}

func TestFeedsSitemapAndProgressData(t *testing.T) {
	o := built(t)
	for _, f := range []string{"devlog/rss.xml", "devlog/atom.xml", "sitemap.xml"} {
		var v any
		if err := xml.Unmarshal(o.Files[f], &v); err != nil {
			t.Errorf("%s: %v", f, err)
		}
	}
	sm := string(o.Files["sitemap.xml"])
	for _, p := range o.Pages {
		if p == "404.html" {
			if strings.Contains(sm, "404") {
				t.Error("sitemap lists the 404 page")
			}
			continue
		}
		loc := BaseURL + "/" + strings.TrimSuffix(p, "index.html")
		if !strings.Contains(sm, "<loc>"+loc+"</loc>") {
			t.Errorf("sitemap is missing %s", loc)
		}
	}
	var pj struct {
		Done, Total, Percent int
		Phases               []struct{ Number int }
	}
	if err := json.Unmarshal(o.Files["progress.json"], &pj); err != nil {
		t.Fatal(err)
	}
	if pj.Total == 0 || len(pj.Phases) != 21 {
		t.Errorf("progress.json: %d boxes, %d phases", pj.Total, len(pj.Phases))
	}
	if !bytes.HasPrefix(o.Files["progress.svg"], []byte("<svg")) {
		t.Error("progress.svg isn't an SVG")
	}
	if !strings.Contains(string(o.Files["robots.txt"]), "Sitemap: "+BaseURL+"/sitemap.xml") {
		t.Error("robots.txt doesn't point at the sitemap")
	}
	for _, f := range []string{"favicon.ico", "apple-touch-icon.png", "404.html"} {
		if len(o.Files[f]) == 0 {
			t.Errorf("%s missing", f)
		}
	}
}

func TestDraftsOnlyInServe(t *testing.T) {
	root := t.TempDir()
	copyRepoInputs(t, root)
	post := "---\ntitle: \"Secret\"\ndate: 2026-11-07\nsummary: \"Not yet\"\ntags: [news]\ncover: null\ndraft: true\n---\n\nHello.\n"
	if err := os.WriteFile(filepath.Join(root, "web/content/devlog/secret.md"), []byte(post), 0o644); err != nil {
		t.Fatal(err)
	}
	prod, err := Build(Config{Root: root, Files: web.Files})
	if err != nil {
		t.Fatal(err)
	}
	for p, data := range prod.Files {
		if strings.Contains(p, "secret") || bytes.Contains(data, []byte("Secret")) {
			t.Errorf("build output %s contains the draft", p)
		}
	}
	dev, err := Build(Config{Root: root, Files: web.Files, Drafts: true})
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := dev.Files["devlog/secret/index.html"]; !ok {
		t.Error("serve output is missing the draft")
	}
}

func TestDownloadsWithARelease(t *testing.T) {
	root := t.TempDir()
	copyRepoInputs(t, root)
	rel := `{"releases":[{"version":"v0.2.0","date":"2026-12-01","channel":"alpha","notes":"/devlog/","files":[{"platform":"linux","url":"https://example.com/l.tar.gz","size":1048576,"sha256":"` + strings.Repeat("ab", 32) + `"}]}]}`
	if err := os.WriteFile(filepath.Join(root, "web/data/releases.json"), []byte(rel), 0o644); err != nil {
		t.Fatal(err)
	}
	o, err := Build(Config{Root: root, Files: web.Files})
	if err != nil {
		t.Fatal(err)
	}
	page := string(o.Files["downloads/index.html"])
	for _, want := range []string{"v0.2.0", "Download for Linux", "System requirements", "Verify your download", "Not available yet"} {
		if !strings.Contains(page, want) {
			t.Errorf("downloads page is missing %q", want)
		}
	}
	for _, bad := range []string{`"version":"0.2"`, `"platform":"amiga"`, `"url":"http://x"`} {
		broken := strings.Replace(strings.Replace(strings.Replace(rel, `"version":"v0.2.0"`, bad, 1), `"platform":"linux"`, bad, 1), `"url":"https://example.com/l.tar.gz"`, bad, 1)
		_ = os.WriteFile(filepath.Join(root, "web/data/releases.json"), []byte(broken), 0o644)
		if _, err := Build(Config{Root: root, Files: web.Files}); err == nil {
			t.Errorf("releases.json with %s was accepted", bad)
		}
	}
}

// copyRepoInputs copies what Build reads into root, without git.
func copyRepoInputs(t *testing.T, root string) {
	t.Helper()
	for _, p := range []string{"BUILD.md", "art/style/palette.hex", "web/assets/art", "web/content/devlog", "web/data"} {
		src := filepath.Join("../../..", p)
		err := filepath.Walk(src, func(f string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			rel, _ := filepath.Rel("../../..", f)
			dst := filepath.Join(root, rel)
			if info.IsDir() {
				return os.MkdirAll(dst, 0o755)
			}
			data, err := os.ReadFile(f)
			if err != nil {
				return err
			}
			if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
				return err
			}
			return os.WriteFile(dst, data, 0o644)
		})
		if err != nil {
			t.Fatal(err)
		}
	}
}

func TestParseAchievements(t *testing.T) {
	h := func(c byte) string { return strings.Repeat(string(c), 40) }
	log := fmt.Sprintf(`commit %s 1700000000
@@ -1 +1 @@
-- [ ] Build the thing.
+- [x] Build the thing.
+- [x] `+"`ADR`"+` New and already done.
commit %s 1700000100
@@ -1 +1 @@
-- [x] Build the thing.
+- [x] Build the whole thing.
-- [x] New and already done.
+- [x] New and already done.
commit %s 1700000200
@@ -1 +1 @@
-- [ ] `+"`HUMAN`"+` Approve it.
+- [x] `+"`HUMAN`"+` Approve it.
`, h('a'), h('b'), h('c'))
	got := parseAchievements([]byte(log))
	var texts []string
	for _, a := range got {
		texts = append(texts, a.Commit+" "+a.Text)
	}
	want := []string{"ccccccc Approve it.", "aaaaaaa New and already done.", "aaaaaaa Build the thing."}
	if strings.Join(texts, "|") != strings.Join(want, "|") {
		t.Fatalf("got %q\nwant %q", texts, want)
	}
}

func TestFrontMatter(t *testing.T) {
	md := newMarkdown()
	good := "---\ntitle: \"A: title\"\ndate: 2026-11-07\nsummary: Plain words\ntags: [a, b-c]\ncover: null\ndraft: false\n---\n\n# Hi\n\n<script>alert(1)</script>\n"
	p, err := parsePost("a-post", []byte(good), md, "../../assets/art")
	if err != nil {
		t.Fatal(err)
	}
	if p.Title != "A: title" || p.Summary != "Plain words" || len(p.Tags) != 2 || p.Draft {
		t.Fatalf("parsed %+v", p)
	}
	if strings.Contains(string(p.Body), "<script") {
		t.Fatal("raw HTML survived into the post body")
	}
	bad := map[string]string{
		"no front matter": "# Hi\n",
		"unclosed":        "---\ntitle: x\n",
		"unknown field":   strings.Replace(good, "draft: false", "draft: false\nauthor: me", 1),
		"bad date":        strings.Replace(good, "2026-11-07", "7 Nov 2026", 1),
		"bad draft":       strings.Replace(good, "draft: false", "draft: maybe", 1),
		"bad tag":         strings.Replace(good, "[a, b-c]", "[Big Tag]", 1),
		"missing cover":   strings.Replace(good, "cover: null\n", "", 1),
		"missing art":     strings.Replace(good, "cover: null", "cover: web/nope.png", 1),
		"duplicate":       strings.Replace(good, "draft: false", "draft: false\ndraft: true", 1),
		"unquoted colon":  strings.Replace(good, "summary: Plain words", "summary: a: b", 1),
	}
	for name, src := range bad {
		if _, err := parsePost("a-post", []byte(src), md, "../../assets/art"); err == nil {
			t.Errorf("%s: accepted", name)
		}
	}
	if _, err := parsePost("Bad Name", []byte(good), md, "../../assets/art"); err == nil {
		t.Error("bad slug accepted")
	}
}

func TestPaletteParsing(t *testing.T) {
	if _, err := ParsePalette("ff00f ; bad\n"); err == nil {
		t.Error("short hex accepted")
	}
	if _, err := ParsePalette("ffffff\n"); err == nil {
		t.Error("unnamed color accepted")
	}
	pal, err := ParsePalette("; c\n1a1420 ; ink: outline\nf4ecd8 ; parchment white\n")
	if err != nil || len(pal) != 2 || pal[1].Name != "parchment-white" || pal[0].Hex != "#1a1420" {
		t.Fatalf("got %v, %v", pal, err)
	}
	if c := Contrast("#000000", "#ffffff"); c < 20.99 || c > 21.01 {
		t.Errorf("black on white is %.2f:1", c)
	}
}

// Rendering a page that uses an image without alt text fails the build.
func TestImgWithoutAltFails(t *testing.T) {
	a := newAssets("../../assets/art", map[string][]byte{})
	if _, err := a.Img("icons/emblem.png", ImgOpts{Scale: 2}); err == nil {
		t.Error("an image with no alt text was allowed")
	}
	h, err := a.Img("icons/emblem.png", ImgOpts{Alt: "decorative", Scale: 2})
	if err != nil || !strings.Contains(string(h), `alt=""`) || !strings.Contains(string(h), `width="64"`) {
		t.Errorf("got %s, %v", h, err)
	}
}
