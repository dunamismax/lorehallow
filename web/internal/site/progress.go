package site

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/dunamismax/lorehallow/web/internal/buildmd"
	"github.com/dunamismax/lorehallow/web/internal/pixelfont"
)

// Progress is BUILD.md as the site shows it.
type Progress struct {
	Plan    *buildmd.Plan
	Phases  []*PhaseView
	Done    int
	Total   int
	Percent int
	Current *PhaseView // nil when everything is done
	// Achievements are checked boxes, newest first, from git history.
	Achievements []Achievement
	// Waiting lists unchecked HUMAN items in the current phase.
	Waiting []*ItemView
}

// PhaseView is one phase with everything its pages need.
type PhaseView struct {
	*buildmd.Phase
	Done, Total, Percent int
	State                string // "done", "current" or "future"
	URL                  string
	Landmark             string // frame name in progress_landmarks
	Walkthrough          string // GitHub URL, or ""
	Sections             []*SectionView
	Version              string // from the overview table's Ships column via Meta, e.g. "v0.2.0"
	Prev, Next           *PhaseView
}

// SectionView is one section of a phase.
type SectionView struct {
	*buildmd.Section
	Items      []*ItemView
	Done       int
	Acceptance bool
}

// ItemView is one checkbox, with its text rendered as inline Markdown.
type ItemView struct {
	*buildmd.Item
	Phase *PhaseView
	HTML  any // template.HTML
}

// Achievement is a box that was checked in a commit.
type Achievement struct {
	Text   string
	HTML   any
	Phase  *PhaseView // nil if the item has since been reworded
	Date   time.Time
	Commit string
	URL    string
}

var walkthroughRE = regexp.MustCompile(`^phase-(\d+)-[a-z0-9-]+\.md$`)

func buildProgress(root string, plan *buildmd.Plan, landmarks []string, inline func(string) any, git bool) (*Progress, error) {
	pr := &Progress{Plan: plan}
	walks := map[int]string{}
	if entries, err := os.ReadDir(filepath.Join(root, "docs", "walkthroughs")); err == nil {
		for _, e := range entries {
			if m := walkthroughRE.FindStringSubmatch(e.Name()); m != nil {
				n, _ := strconv.Atoi(m[1])
				walks[n] = RepoURL + "/blob/main/docs/walkthroughs/" + e.Name()
			}
		}
	}
	current := plan.Current()
	state := "done"
	for _, p := range plan.Phases {
		pp := p.Progress()
		v := &PhaseView{Phase: p, Done: pp.Done, Total: pp.Total, Percent: percent(pp.Done, pp.Total),
			URL: fmt.Sprintf("/progress/phase-%d/", p.Number), Walkthrough: walks[p.Number]}
		if current != nil && p.Number == current.Number {
			state = "current"
		} else if state == "current" {
			state = "future"
		}
		v.State = state
		if p.Number < len(landmarks) {
			v.Landmark = landmarks[p.Number]
		}
		if m := regexp.MustCompile(`v\d+\.\d+\.\d+`).FindString(p.Meta["Ships"]); m != "" {
			v.Version = m
		}
		for _, s := range p.Sections {
			sv := &SectionView{Section: s, Acceptance: s.ID == buildmd.Acceptance}
			for _, it := range s.Items {
				iv := &ItemView{Item: it, Phase: v, HTML: inline(it.Text)}
				sv.Items = append(sv.Items, iv)
				if it.Checked {
					sv.Done++
				}
				if v.State == "current" && !it.Checked && it.Tag == "HUMAN" {
					pr.Waiting = append(pr.Waiting, iv)
				}
			}
			v.Sections = append(v.Sections, sv)
		}
		if n := len(pr.Phases); n > 0 {
			v.Prev, pr.Phases[n-1].Next = pr.Phases[n-1], v
		}
		pr.Phases = append(pr.Phases, v)
		if v.State == "current" {
			pr.Current = v
		}
	}
	total := plan.Progress()
	pr.Done, pr.Total, pr.Percent = total.Done, total.Total, percent(total.Done, total.Total)
	if git {
		ach, err := achievements(root)
		if err != nil {
			return nil, err
		}
		byText := map[string]*PhaseView{}
		for _, p := range pr.Phases {
			for _, it := range p.Items() {
				byText[it.Text] = p
			}
		}
		for i := range ach {
			ach[i].Phase = byText[ach[i].Text]
			ach[i].HTML = inline(ach[i].Text)
		}
		pr.Achievements = ach
	}
	return pr, nil
}

// percent rounds down, so 100% means every box is checked.
func percent(done, total int) int {
	if total == 0 {
		return 0
	}
	return done * 100 / total
}

var (
	diffCheckedRE = regexp.MustCompile("^\\+- \\[x\\] (?:`(?:HUMAN|ADR)` )?(.*)$")
	diffRemovedRE = regexp.MustCompile("^-- \\[( |x)\\] (?:`(?:HUMAN|ADR)` )?(.*)$")
	commitLineRE  = regexp.MustCompile(`^commit ([0-9a-f]{40}) (\d+)$`)
)

// achievements walks BUILD.md's history and returns every box that was
// checked, with the commit that checked it, newest first.
//
// In each commit, an added "- [x]" line is a checked box when the same
// commit removed that line unchecked, or added it new. An added checked line
// that pairs with a removed checked line is a reword or a move, not news.
func achievements(root string) ([]Achievement, error) {
	cmd := exec.Command("git", "-C", root, "log", "--reverse", "--no-merges", "-p", "-U0",
		"--format=commit %H %ct", "--", "BUILD.md")
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("git log BUILD.md: %w", err)
	}
	return parseAchievements(out), nil
}

func parseAchievements(log []byte) []Achievement {
	var all []Achievement
	var hash string
	var when time.Time
	var added []string
	unchecked := map[string]bool{}
	removedChecked := map[string]int{}
	flush := func() {
		var maybe []string
		for _, t := range added {
			switch {
			case unchecked[t]:
				all = append(all, Achievement{Text: t, Date: when, Commit: hash[:7], URL: RepoURL + "/commit/" + hash})
			case removedChecked[t] > 0:
				removedChecked[t]--
			default:
				maybe = append(maybe, t)
			}
		}
		// Whatever checked lines were removed and not re-added verbatim pair
		// up with these as rewords; the rest are new, already-checked items.
		left := 0
		for _, n := range removedChecked {
			left += n
		}
		for _, t := range maybe {
			if left > 0 {
				left--
				continue
			}
			all = append(all, Achievement{Text: t, Date: when, Commit: hash[:7], URL: RepoURL + "/commit/" + hash})
		}
		added = nil
		unchecked = map[string]bool{}
		removedChecked = map[string]int{}
	}
	sc := bufio.NewScanner(bytes.NewReader(log))
	sc.Buffer(make([]byte, 1<<20), 1<<24)
	for sc.Scan() {
		line := sc.Text()
		if m := commitLineRE.FindStringSubmatch(line); m != nil {
			if hash != "" {
				flush()
			}
			hash = m[1]
			sec, _ := strconv.ParseInt(m[2], 10, 64)
			when = time.Unix(sec, 0).UTC()
			continue
		}
		if m := diffCheckedRE.FindStringSubmatch(line); m != nil {
			added = append(added, strings.TrimSpace(m[1]))
			continue
		}
		if m := diffRemovedRE.FindStringSubmatch(line); m != nil {
			t := strings.TrimSpace(m[2])
			if m[1] == " " {
				unchecked[t] = true
			} else {
				removedChecked[t]++
			}
		}
	}
	if hash != "" {
		flush()
	}
	slices.Reverse(all)
	return all
}

const badgePad = 6

func (pr *Progress) badgeText(f *pixelfont.Font) (label, value string, lw, vw int) {
	label = "lorehallow"
	value = fmt.Sprintf("%d%% built", pr.Percent)
	if pr.Current != nil {
		value = fmt.Sprintf("phase %d · %d%%", pr.Current.Number, pr.Percent)
	}
	return label, value, f.Measure(label) + 2*badgePad, f.Measure(value) + 2*badgePad
}

// BadgeWidth is the badge's width in pixels.
func (pr *Progress) BadgeWidth(f *pixelfont.Font) int {
	_, _, lw, vw := pr.badgeText(f)
	return lw + vw
}

// ProgressJSON is /progress.json.
func (pr *Progress) JSON(base string, built time.Time, commit string) ([]byte, error) {
	type phase struct {
		Number  int    `json:"number"`
		Title   string `json:"title"`
		Done    int    `json:"done"`
		Total   int    `json:"total"`
		Percent int    `json:"percent"`
		State   string `json:"state"`
		URL     string `json:"url"`
	}
	type cur struct {
		Number int    `json:"number"`
		Title  string `json:"title"`
		URL    string `json:"url"`
	}
	type waiting struct {
		Text  string `json:"text"`
		Phase int    `json:"phase"`
	}
	type ach struct {
		Text   string `json:"text"`
		Phase  *int   `json:"phase"`
		Date   string `json:"date"`
		Commit string `json:"commit"`
	}
	doc := struct {
		Generated string    `json:"generated"`
		Commit    string    `json:"commit"`
		Done      int       `json:"done"`
		Total     int       `json:"total"`
		Percent   int       `json:"percent"`
		Current   *cur      `json:"current"`
		Phases    []phase   `json:"phases"`
		Waiting   []waiting `json:"waiting_on_owner"`
		Recent    []ach     `json:"recent_achievements"`
	}{Generated: built.UTC().Format(time.RFC3339), Commit: commit, Done: pr.Done, Total: pr.Total,
		Percent: pr.Percent, Phases: []phase{}, Waiting: []waiting{}, Recent: []ach{}}
	if pr.Current != nil {
		doc.Current = &cur{pr.Current.Number, pr.Current.Title, base + pr.Current.URL}
	}
	for _, p := range pr.Phases {
		doc.Phases = append(doc.Phases, phase{p.Number, p.Title, p.Done, p.Total, p.Percent, p.State, base + p.URL})
	}
	for _, w := range pr.Waiting {
		doc.Waiting = append(doc.Waiting, waiting{w.Text, w.Phase.Number})
	}
	for i, a := range pr.Achievements {
		if i == 20 {
			break
		}
		var n *int
		if a.Phase != nil {
			n = &a.Phase.Number
		}
		doc.Recent = append(doc.Recent, ach{a.Text, n, a.Date.Format("2006-01-02"), a.Commit})
	}
	out, err := json.MarshalIndent(doc, "", "  ")
	return append(out, '\n'), err
}

// Badge draws /progress.svg with the hand-drawn pixel font, so it looks the
// same everywhere and needs no font at all.
func (pr *Progress) Badge(f *pixelfont.Font, t *Theme) []byte {
	label, value, lw, vw := pr.badgeText(f)
	const pad, h, scale = 6, 20, 1
	top := (h - pixelfont.CapRows*scale) / 2
	var b strings.Builder
	fmt.Fprintf(&b, `<svg xmlns="http://www.w3.org/2000/svg" width="%d" height="%d" viewBox="0 0 %d %d" shape-rendering="crispEdges" role="img" aria-label="%s: %s">`,
		lw+vw, h, lw+vw, h, label, value)
	fmt.Fprintf(&b, `<title>%s: %s</title>`, label, value)
	fmt.Fprintf(&b, `<rect width="%d" height="%d" fill="%s"/>`, lw, h, t.Hex("night-plum"))
	fmt.Fprintf(&b, `<rect x="%d" width="%d" height="%d" fill="%s"/>`, lw, vw, h, t.Hex("lantern-gold"))
	path := func(s string, x int, color string) {
		fmt.Fprintf(&b, `<path fill="%s" d="`, color)
		for _, r := range f.Rects(s, x, top, scale) {
			fmt.Fprintf(&b, "M%d %dh%dv%dh-%dz", r.Min.X, r.Min.Y, r.Dx(), r.Dy(), r.Dx())
		}
		b.WriteString(`"/>`)
	}
	path(label, pad, t.Hex("parchment-white"))
	path(value, lw+pad, t.Hex("ink"))
	b.WriteString("</svg>\n")
	return []byte(b.String())
}
