package site

import (
	"bytes"
	"fmt"
	"html/template"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/yuin/goldmark/v2/extension"
	"github.com/yuin/goldmark/v2/parser"
	"github.com/yuin/goldmark/v2/renderer/html"
)

// Post is one devlog post from web/content/devlog/<slug>.md.
type Post struct {
	Slug    string
	Title   string
	Date    time.Time
	Summary string
	Tags    []string
	Cover   string // art path under web/assets/art, or ""
	Draft   bool
	Body    template.HTML
	URL     string
	Words   int
}

// Minutes is a rough reading time.
func (p *Post) Minutes() int { return max(1, (p.Words+199)/200) }

var (
	slugRE = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)
	tagRE  = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)
)

// markdown renders Markdown the way every page on the site does: GitHub
// Flavored Markdown, smart punctuation, heading anchors, and raw HTML left
// out (so a post can't carry a script).
type markdown struct {
	p parser.Parser
	r html.Renderer
}

func newMarkdown() *markdown {
	return &markdown{
		p: parser.New(parser.WithExtensions(extension.NewGFMParser(), extension.NewTypographerParser()),
			parser.WithAutoHeadingID()),
		r: html.New(html.WithExtensions(extension.NewGFMHTMLRenderer())),
	}
}

func (m *markdown) render(src []byte) (template.HTML, error) {
	var buf bytes.Buffer
	doc := m.p.Parse(src)
	if err := m.r.Render(&buf, src, doc); err != nil {
		return "", err
	}
	return template.HTML(buf.String()), nil //nolint:gosec // goldmark escapes text and drops raw HTML
}

var relLinkRE = regexp.MustCompile(`href="([^"#/:][^":]*)"`)

// repoLinks points relative links (BUILD.md's links to files in the
// repository) at GitHub.
func repoLinks(h template.HTML) template.HTML {
	return template.HTML(relLinkRE.ReplaceAllString(string(h), `href="`+RepoURL+`/blob/main/$1"`)) //nolint:gosec // rewrites an attribute goldmark escaped
}

// inline renders one line of Markdown without the surrounding <p>.
func (m *markdown) inline(s string) template.HTML {
	out, err := m.render([]byte(s))
	if err != nil {
		return template.HTML(template.HTMLEscapeString(s)) //nolint:gosec // escaped
	}
	h := strings.TrimSpace(string(out))
	h = strings.TrimPrefix(h, "<p>")
	h = strings.TrimSuffix(h, "</p>")
	return template.HTML(h) //nolint:gosec // goldmark output
}

func loadPosts(dir string, md *markdown, drafts bool, artDir string) ([]*Post, error) {
	entries, err := os.ReadDir(dir)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var posts []*Post
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") || strings.EqualFold(e.Name(), "README.md") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			return nil, err
		}
		p, err := parsePost(strings.TrimSuffix(e.Name(), ".md"), data, md, artDir)
		if err != nil {
			return nil, fmt.Errorf("devlog %s: %w", e.Name(), err)
		}
		if p.Draft && !drafts {
			continue
		}
		posts = append(posts, p)
	}
	slices.SortStableFunc(posts, func(a, b *Post) int {
		if c := b.Date.Compare(a.Date); c != 0 {
			return c
		}
		return strings.Compare(a.Slug, b.Slug)
	})
	return posts, nil
}

func parsePost(slug string, data []byte, md *markdown, artDir string) (*Post, error) {
	if !slugRE.MatchString(slug) {
		return nil, fmt.Errorf("file name %q must be lowercase words joined by hyphens", slug)
	}
	fm, body, err := splitFrontMatter(string(data))
	if err != nil {
		return nil, err
	}
	p := &Post{Slug: slug, URL: "/devlog/" + slug + "/"}
	required := map[string]bool{"title": false, "date": false, "summary": false, "tags": false, "cover": false, "draft": false}
	for _, f := range fm {
		seen, known := required[f.key]
		if !known {
			return nil, fmt.Errorf("line %d: unknown front matter field %q (allowed: title, date, summary, tags, cover, draft)", f.line, f.key)
		}
		if seen {
			return nil, fmt.Errorf("line %d: %q given twice", f.line, f.key)
		}
		required[f.key] = true
		switch f.key {
		case "title", "summary":
			s, err := fmString(f)
			if err != nil {
				return nil, err
			}
			if s == "" {
				return nil, fmt.Errorf("line %d: %s is empty", f.line, f.key)
			}
			if f.key == "title" {
				p.Title = s
			} else {
				p.Summary = s
			}
		case "date":
			s, err := fmString(f)
			if err != nil {
				return nil, err
			}
			d, err := time.Parse("2006-01-02", s)
			if err != nil {
				return nil, fmt.Errorf("line %d: date must be YYYY-MM-DD", f.line)
			}
			p.Date = d
		case "tags":
			tags, err := fmList(f)
			if err != nil {
				return nil, err
			}
			for _, t := range tags {
				if !tagRE.MatchString(t) {
					return nil, fmt.Errorf("line %d: tag %q must be lowercase words joined by hyphens", f.line, t)
				}
			}
			p.Tags = tags
		case "cover":
			if f.value != "null" {
				s, err := fmString(f)
				if err != nil {
					return nil, err
				}
				if _, err := os.Stat(filepath.Join(artDir, filepath.FromSlash(s))); err != nil {
					return nil, fmt.Errorf("line %d: cover %q isn't in web/assets/art", f.line, s)
				}
				p.Cover = s
			}
		case "draft":
			switch f.value {
			case "true":
				p.Draft = true
			case "false":
			default:
				return nil, fmt.Errorf("line %d: draft must be true or false", f.line)
			}
		}
	}
	for k, seen := range required {
		if !seen {
			return nil, fmt.Errorf("front matter is missing %q", k)
		}
	}
	p.Body, err = md.render([]byte(body))
	if err != nil {
		return nil, err
	}
	p.Words = len(strings.Fields(body))
	return p, nil
}

type fmField struct {
	key, value string
	line       int
}

// splitFrontMatter reads the "---" block at the top of a post. It accepts
// a small, strict subset of YAML: `key: value` lines, where a value is a
// quoted or bare string, true, false, null, or a [flow, list].
func splitFrontMatter(src string) ([]fmField, string, error) {
	lines := strings.Split(strings.ReplaceAll(src, "\r\n", "\n"), "\n")
	if len(lines) == 0 || lines[0] != "---" {
		return nil, "", fmt.Errorf("must start with a --- front matter block")
	}
	var fields []fmField
	for i := 1; i < len(lines); i++ {
		line := lines[i]
		if line == "---" {
			return fields, strings.Join(lines[i+1:], "\n"), nil
		}
		if strings.TrimSpace(line) == "" || strings.HasPrefix(strings.TrimSpace(line), "#") {
			continue
		}
		k, v, ok := strings.Cut(line, ":")
		if !ok || strings.TrimSpace(k) != k || k == "" {
			return nil, "", fmt.Errorf("line %d: expected \"key: value\"", i+1)
		}
		fields = append(fields, fmField{k, strings.TrimSpace(v), i + 1})
	}
	return nil, "", fmt.Errorf("front matter has no closing ---")
}

func fmString(f fmField) (string, error) {
	v := f.value
	if strings.HasPrefix(v, `"`) {
		s, err := strconv.Unquote(v)
		if err != nil {
			return "", fmt.Errorf("line %d: bad quoted string", f.line)
		}
		return s, nil
	}
	if strings.HasPrefix(v, "'") {
		if len(v) < 2 || !strings.HasSuffix(v, "'") {
			return "", fmt.Errorf("line %d: bad quoted string", f.line)
		}
		return strings.ReplaceAll(v[1:len(v)-1], "''", "'"), nil
	}
	if strings.ContainsAny(v, "[]{}#&*!|>%@`") || strings.Contains(v, ": ") {
		return "", fmt.Errorf("line %d: quote this value", f.line)
	}
	return v, nil
}

func fmList(f fmField) ([]string, error) {
	v := f.value
	if !strings.HasPrefix(v, "[") || !strings.HasSuffix(v, "]") {
		return nil, fmt.Errorf("line %d: %s must be a [list, like, this]", f.line, f.key)
	}
	var out []string
	for _, part := range strings.Split(v[1:len(v)-1], ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		s, err := fmString(fmField{f.key, part, f.line})
		if err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, nil
}

// Tags lists every tag with its posts, alphabetically.
func tagIndex(posts []*Post) ([]string, map[string][]*Post) {
	by := map[string][]*Post{}
	for _, p := range posts {
		for _, t := range p.Tags {
			by[t] = append(by[t], p)
		}
	}
	tags := make([]string, 0, len(by))
	for t := range by {
		tags = append(tags, t)
	}
	slices.Sort(tags)
	return tags, by
}
