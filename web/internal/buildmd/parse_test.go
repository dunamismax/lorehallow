package buildmd

import (
	"fmt"
	"os"
	"regexp"
	"strings"
	"testing"
)

const meta = "> **Goal:** A goal.\n> **Ships:** v0.0.0\n> **Lesson:** A lesson.\n> **Depends on:** Nothing"

func phase(n int, title string, body ...string) string {
	return fmt.Sprintf("## Phase %d: %s\n\n%s\n\n%s\n", n, title, meta, strings.Join(body, "\n"))
}

var good = strings.Join([]string{
	"# Plan",
	"",
	"## Phase overview",
	"",
	"| Phase | Title | Ships | Core lesson |",
	"|---|---|---|---|",
	"| 0 | Start | x | y |",
	"| 1 | Next | x | y |",
	"",
	phase(0, "Start",
		"### 0.1 First",
		"",
		"Some prose is fine here.",
		"",
		"- [x] Done thing.",
		"- [ ] `HUMAN` Owner thing.",
		"  - a note under an item is ignored",
		"- [ ] `server/cmd/x`: code in backticks is not a tag.",
		"",
		"### Acceptance criteria",
		"",
		"- [x] It works.",
	),
	phase(1, "Next", "### 1.1 Only", "", "- [ ] `ADR` Decide.", "", "### Acceptance criteria", "", "- [ ] Ships."),
	"## Deferred",
	"",
	"- not counted",
}, "\n")

// replace swaps the first occurrence, like JavaScript's String.replace.
func replace(s, old, new string) string { return strings.Replace(s, old, new, 1) }

func problemsOf(text string) []string {
	var out []string
	for _, p := range Parse(text).Problems {
		out = append(out, p.String())
	}
	return out
}

func TestParsesPhasesMetadataSectionsItemsAndTags(t *testing.T) {
	plan := Parse(good)
	if len(plan.Problems) != 0 {
		t.Fatalf("problems: %v", plan.Problems)
	}
	if len(plan.Phases) != 2 {
		t.Fatalf("phases = %d, want 2", len(plan.Phases))
	}
	p0, p1 := plan.Phases[0], plan.Phases[1]
	if p0.Title != "Start" || p0.Meta["Goal"] != "A goal." || p0.Meta["Depends on"] != "Nothing" {
		t.Errorf("phase 0 = %q %v", p0.Title, p0.Meta)
	}
	var ids []string
	for _, s := range p0.Sections {
		ids = append(ids, s.ID)
	}
	if strings.Join(ids, ",") != "0.1,acceptance" {
		t.Errorf("section ids = %v", ids)
	}
	items := p0.Sections[0].Items
	want := []struct {
		checked bool
		tag     string
	}{{true, ""}, {false, "HUMAN"}, {false, ""}}
	if len(items) != len(want) {
		t.Fatalf("items = %d, want %d", len(items), len(want))
	}
	for i, w := range want {
		if items[i].Checked != w.checked || items[i].Tag != w.tag {
			t.Errorf("item %d = %v %q, want %v %q", i, items[i].Checked, items[i].Tag, w.checked, w.tag)
		}
	}
	if items[1].Text != "Owner thing." {
		t.Errorf("item 1 text = %q", items[1].Text)
	}
	if items[2].Text != "`server/cmd/x`: code in backticks is not a tag." {
		t.Errorf("item 2 text = %q", items[2].Text)
	}
	if p1.Sections[0].Items[0].Tag != "ADR" {
		t.Errorf("phase 1 tag = %q", p1.Sections[0].Items[0].Tag)
	}
}

func TestProgressAndCurrentPhase(t *testing.T) {
	plan := Parse(good)
	if got := plan.Phases[0].Progress(); got != (Progress{2, 4}) {
		t.Errorf("phase 0 progress = %v", got)
	}
	if got := plan.Progress(); got != (Progress{2, 6}) {
		t.Errorf("overall progress = %v", got)
	}
	if c := plan.Current(); c == nil || c.Number != 0 {
		t.Errorf("current = %v, want Phase 0", c)
	}
	if c := Parse(strings.ReplaceAll(good, "- [ ]", "- [x]")).Current(); c != nil {
		t.Errorf("current = Phase %d, want none", c.Number)
	}
}

func TestItemsOutsidePhasesAreNotCounted(t *testing.T) {
	text := good + "\n- [ ] stray\n"
	if got := Parse(text).Progress().Total; got != 6 {
		t.Errorf("total = %d, want 6", got)
	}
	if !strings.Contains(strings.Join(problemsOf(text), "\n"), "checkbox outside a phase") {
		t.Errorf("no stray-checkbox problem: %v", problemsOf(text))
	}
}

func TestReportsFormatProblems(t *testing.T) {
	cases := []struct {
		name, text, want string
	}{
		{"nested checkbox", replace(good, "  - a note", "  - [ ] a nested box"), `column 0`},
		{"uppercase X", replace(good, "- [x] Done thing.", "- [X] Done thing."), `column 0`},
		{"star bullet", replace(good, "- [x] Done thing.", "* [x] Done thing."), `column 0`},
		{"unknown tag", replace(good, "`HUMAN` Owner", "`OWNER` Owner"), "unknown tag `OWNER`"},
		{"empty item", replace(good, "- [x] Done thing.", "- [x] "), `no text`},
		{"missing metadata", replace(good, "> **Lesson:** A lesson.\n", ""), `missing "> \*\*Lesson:\*\*"`},
		{"unknown metadata", replace(good, "> **Ships:**", "> **Shipz:**"), `unknown metadata key "Shipz"`},
		{"duplicate metadata", replace(good, "> **Ships:** v0.0.0", "> **Goal:** again"), `duplicate metadata key "Goal"`},
		{"malformed metadata", replace(good, "> **Ships:** v0.0.0", "> Ships: v0.0.0"), `metadata line must be`},
		{"phase out of order", replace(good, "## Phase 1: Next", "## Phase 2: Next"), `phase 2 is out of order; expected Phase 1`},
		{"bad phase heading", replace(good, "## Phase 1: Next", "## Phase 1 - Next"), `phase heading must be`},
		{"section in wrong phase", replace(good, "### 1.1 Only", "### 0.2 Only"), `section 0.2 is in Phase 1`},
		{"section out of order", replace(good, "### 1.1 Only", "### 1.2 Only"), `expected 1.1`},
		{"bad section heading", replace(good, "### 1.1 Only", "### Only"), `section heading must be`},
		{"no acceptance", replace(good, "### Acceptance criteria\n\n- [ ] Ships.", ""), `Phase 1 has no Acceptance criteria`},
		{"section after acceptance", replace(good, "- [ ] Ships.", "- [ ] Ships.\n\n### 1.2 Late\n\n- [ ] x"), `before Acceptance`},
		{"item before any section", replace(good, "### 1.1 Only\n", ""), `item outside a section`},
		{"empty section", replace(good, "- [ ] `ADR` Decide.", ""), `section 1.1 has no items`},
		{"h4 inside a phase", replace(good, "Some prose is fine here.", "#### Sub"), `only ## and ###`},
		{"overview title mismatch", replace(good, "| 1 | Next |", "| 1 | Later |"), `overview title "Later"`},
		{"overview missing phase", replace(good, "| 1 | Next | x | y |\n", ""), `missing from the Phase overview`},
		{"overview extra phase", replace(good, "| 1 | Next | x | y |", "| 1 | Next | x | y |\n| 2 | Ghost | x | y |"), `lists Phase 2`},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			re := regexp.MustCompile(c.want)
			got := problemsOf(c.text)
			for _, p := range got {
				if re.MatchString(p) {
					return
				}
			}
			t.Errorf("want %s, got:\n%s", c.want, strings.Join(got, "\n"))
		})
	}
}

func TestProblemsCarryLineNumbers(t *testing.T) {
	text := replace(good, "- [x] Done thing.", "- [X] Done thing.")
	line := 0
	for i, l := range strings.Split(text, "\n") {
		if strings.HasPrefix(l, "- [X]") {
			line = i + 1
			break
		}
	}
	for _, p := range Parse(text).Problems {
		if p.Line == line {
			return
		}
	}
	t.Errorf("no problem on line %d: %v", line, Parse(text).Problems)
}

func TestCRLFLineEndings(t *testing.T) {
	plan := Parse(strings.ReplaceAll(good, "\n", "\r\n"))
	if len(plan.Problems) != 0 || plan.Progress() != (Progress{2, 6}) {
		t.Errorf("CRLF: problems %v, progress %v", plan.Problems, plan.Progress())
	}
}

func TestRealBuildMDFollowsTheContract(t *testing.T) {
	data, err := os.ReadFile("../../../BUILD.md")
	if err != nil {
		t.Fatal(err)
	}
	plan := Parse(string(data))
	if len(plan.Problems) != 0 {
		t.Fatalf("BUILD.md problems: %v", plan.Problems)
	}
	if plan.Phases[0].Number != 0 || plan.Progress().Done == 0 {
		t.Errorf("phase 0 = %d, done = %d", plan.Phases[0].Number, plan.Progress().Done)
	}
}

func FuzzParse(f *testing.F) {
	f.Add(good)
	f.Add("## Phase 0: X\n> **Goal:** y\n### 0.1 Z\n- [ ] `HUMAN` a\n")
	f.Fuzz(func(t *testing.T, text string) {
		plan := Parse(text) // must never panic
		p := plan.Progress()
		if p.Done < 0 || p.Done > p.Total {
			t.Errorf("progress %v", p)
		}
		for i := 1; i < len(plan.Problems); i++ {
			if plan.Problems[i].Line < plan.Problems[i-1].Line {
				t.Errorf("problems out of line order")
			}
		}
	})
}
