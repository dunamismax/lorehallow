// Package buildmd parses BUILD.md. It implements the format contract at the
// top of BUILD.md: phases, metadata, sections, items, tags and checked state,
// plus every format problem with its line number.
//
// The website renders the progress tracker from the result, and
// `make buildmd-lint` fails on any problem it reports. If the format changes,
// change this file, its tests and the contract together.
package buildmd

import (
	"fmt"
	"regexp"
	"slices"
	"sort"
	"strconv"
	"strings"
)

// Tags an item may carry, in backticks right after its checkbox.
var Tags = []string{"HUMAN", "ADR"}

// MetaKeys are the metadata lines every phase has, in order.
var MetaKeys = []string{"Goal", "Ships", "Lesson", "Depends on"}

// Acceptance is the ID of a phase's acceptance-criteria section.
const Acceptance = "acceptance"

// Item is one checkbox line.
type Item struct {
	Text    string `json:"text"` // without the checkbox and tag
	Checked bool   `json:"checked"`
	Tag     string `json:"tag,omitempty"` // "HUMAN", "ADR" or ""
	Line    int    `json:"line"`          // 1-based line number in BUILD.md
}

// Section is a "### N.M Title" section, or the acceptance criteria.
type Section struct {
	ID    string  `json:"id"` // "N.M", or Acceptance
	Title string  `json:"title"`
	Items []*Item `json:"items"`
	Line  int     `json:"line"`
}

// Phase is a "## Phase N: Title" block.
type Phase struct {
	Number   int               `json:"number"`
	Title    string            `json:"title"`
	Meta     map[string]string `json:"meta"`
	Sections []*Section        `json:"sections"`
	Line     int               `json:"line"`
}

// Problem is one format-contract violation.
type Problem struct {
	Line    int    `json:"line"`
	Message string `json:"message"`
}

func (p Problem) String() string { return fmt.Sprintf("%d: %s", p.Line, p.Message) }

// Plan is a parsed BUILD.md.
type Plan struct {
	Phases   []*Phase  `json:"phases"`
	Problems []Problem `json:"problems"`
}

// Progress counts checked boxes.
type Progress struct {
	Done  int `json:"done"`
	Total int `json:"total"`
}

var (
	phaseRE       = regexp.MustCompile(`^## Phase (\d+): (\S.*)$`)
	badPhaseRE    = regexp.MustCompile(`^## Phase\s*\d`)
	metaRE        = regexp.MustCompile(`^> \*\*([^*]+):\*\* (\S.*)$`)
	sectionRE     = regexp.MustCompile(`^### (\d+)\.(\d+) (\S.*)$`)
	itemRE        = regexp.MustCompile(`^- \[( |x)\] (.*)$`)
	checkboxishRE = regexp.MustCompile(`^\s*[-*+] \[[^\]]?\]`) // anything that looks like a checkbox
	tagRE         = regexp.MustCompile("^`([A-Z]+)` (\\S.*)$")
	overviewRowRE = regexp.MustCompile(`^\| (\d+) \| ([^|]+?) \|`)
)

const acceptanceHeading = "### Acceptance criteria"

type overviewRow struct {
	title string
	line  int
}

// Parse reads BUILD.md text into phases, with every format problem found.
func Parse(text string) *Plan {
	plan := &Plan{Phases: []*Phase{}, Problems: []Problem{}}
	problem := func(line int, format string, args ...any) {
		plan.Problems = append(plan.Problems, Problem{line, fmt.Sprintf(format, args...)})
	}
	overview := map[int]overviewRow{}
	var overviewOrder []int

	var phase *Phase
	var section *Section
	inMeta := false // still inside the metadata blockquote under a phase heading
	inOverview := false

	for index, raw := range strings.Split(text, "\n") {
		n := index + 1
		line := strings.TrimSuffix(raw, "\r")

		if strings.HasPrefix(line, "## ") {
			inOverview = line == "## Phase overview"
			if m := phaseRE.FindStringSubmatch(line); m != nil {
				number, _ := strconv.Atoi(m[1])
				expected := 0
				if len(plan.Phases) > 0 {
					expected = plan.Phases[len(plan.Phases)-1].Number + 1
				}
				if number != expected {
					problem(n, "phase %d is out of order; expected Phase %d", number, expected)
				}
				phase = &Phase{Number: number, Title: strings.TrimSpace(m[2]), Meta: map[string]string{},
					Sections: []*Section{}, Line: n}
				plan.Phases = append(plan.Phases, phase)
				section = nil
				inMeta = true
				continue
			}
			if badPhaseRE.MatchString(line) {
				problem(n, `phase heading must be "## Phase <N>: <Title>"`)
			}
			phase, section, inMeta = nil, nil, false
			continue
		}

		if inOverview {
			if row := overviewRowRE.FindStringSubmatch(line); row != nil {
				number, _ := strconv.Atoi(row[1])
				if _, seen := overview[number]; !seen {
					overviewOrder = append(overviewOrder, number)
				}
				overview[number] = overviewRow{strings.TrimSpace(row[2]), n}
			}
		}

		if phase == nil {
			if checkboxishRE.MatchString(line) {
				problem(n, "checkbox outside a phase")
			}
			continue
		}

		if inMeta {
			if strings.TrimSpace(line) == "" {
				if len(phase.Meta) > 0 {
					inMeta = false
				}
				continue
			}
			if strings.HasPrefix(line, ">") {
				m := metaRE.FindStringSubmatch(line)
				switch {
				case m == nil:
					problem(n, `metadata line must be "> **<Key>:** <text>"`)
				case !slices.Contains(MetaKeys, m[1]):
					problem(n, `unknown metadata key "%s"; use %s`, m[1], strings.Join(MetaKeys, ", "))
				default:
					if _, dup := phase.Meta[m[1]]; dup {
						problem(n, `duplicate metadata key "%s"`, m[1])
					} else {
						phase.Meta[m[1]] = strings.TrimSpace(m[2])
					}
				}
				continue
			}
			inMeta = false
		} else if strings.HasPrefix(line, "> **") {
			problem(n, "metadata must sit directly under the phase heading")
			continue
		}

		if strings.HasPrefix(line, "### ") {
			if line == acceptanceHeading {
				if phase.has(Acceptance) {
					problem(n, "duplicate Acceptance criteria section")
				}
				section = &Section{ID: Acceptance, Title: "Acceptance criteria", Items: []*Item{}, Line: n}
				phase.Sections = append(phase.Sections, section)
				continue
			}
			m := sectionRE.FindStringSubmatch(line)
			if m == nil {
				problem(n, `section heading must be "### <N>.<M> <Title>" or "### Acceptance criteria"`)
				section = nil
				continue
			}
			major, _ := strconv.Atoi(m[1])
			minor, _ := strconv.Atoi(m[2])
			if major != phase.Number {
				problem(n, "section %d.%d is in Phase %d", major, minor, phase.Number)
			}
			numbered := 0
			for _, s := range phase.Sections {
				if s.ID != Acceptance {
					numbered++
				}
			}
			if minor != numbered+1 {
				problem(n, "section %d.%d is out of order; expected %d.%d", major, minor, phase.Number, numbered+1)
			}
			if phase.has(Acceptance) {
				problem(n, "numbered sections must come before Acceptance criteria")
			}
			section = &Section{ID: fmt.Sprintf("%d.%d", major, minor), Title: strings.TrimSpace(m[3]),
				Items: []*Item{}, Line: n}
			phase.Sections = append(phase.Sections, section)
			continue
		}

		if strings.HasPrefix(line, "#") {
			problem(n, "only ## and ### headings are allowed inside a phase")
			continue
		}

		if m := itemRE.FindStringSubmatch(line); m != nil {
			if section == nil {
				problem(n, "item outside a section")
				continue
			}
			text, tag := m[2], ""
			if t := tagRE.FindStringSubmatch(text); t != nil {
				if slices.Contains(Tags, t[1]) {
					tag, text = t[1], t[2]
				} else {
					problem(n, "unknown tag `%s`; use `%s`", t[1], strings.Join(Tags, "` or `"))
				}
			}
			if strings.TrimSpace(text) == "" {
				problem(n, "item has no text")
				continue
			}
			section.Items = append(section.Items, &Item{Text: strings.TrimSpace(text), Checked: m[1] == "x",
				Tag: tag, Line: n})
			continue
		}
		if checkboxishRE.MatchString(line) {
			problem(n, `checkboxes must start at column 0 as "- [ ] " or "- [x] " (no nesting)`)
		}
	}

	for _, p := range plan.Phases {
		for _, key := range MetaKeys {
			if _, ok := p.Meta[key]; !ok {
				problem(p.Line, `Phase %d is missing "> **%s:**"`, p.Number, key)
			}
		}
		if !p.has(Acceptance) {
			problem(p.Line, "Phase %d has no Acceptance criteria section", p.Number)
		}
		for _, s := range p.Sections {
			if len(s.Items) == 0 {
				problem(s.Line, "section %s has no items", s.ID)
			}
		}
	}

	if len(overview) > 0 {
		for _, p := range plan.Phases {
			row, ok := overview[p.Number]
			switch {
			case !ok:
				problem(p.Line, "Phase %d is missing from the Phase overview table", p.Number)
			case row.title != p.Title:
				problem(row.line, `overview title "%s" doesn't match "## Phase %d: %s"`, row.title, p.Number, p.Title)
			}
		}
		for _, number := range overviewOrder {
			if !slices.ContainsFunc(plan.Phases, func(p *Phase) bool { return p.Number == number }) {
				problem(overview[number].line, "overview lists Phase %d, which has no heading", number)
			}
		}
	}

	sort.SliceStable(plan.Problems, func(i, j int) bool { return plan.Problems[i].Line < plan.Problems[j].Line })
	return plan
}

func (p *Phase) has(id string) bool {
	return slices.ContainsFunc(p.Sections, func(s *Section) bool { return s.ID == id })
}

// Items lists every item in the phase, acceptance criteria included.
func (p *Phase) Items() []*Item {
	var items []*Item
	for _, s := range p.Sections {
		items = append(items, s.Items...)
	}
	return items
}

// Progress counts the phase's checked boxes.
func (p *Phase) Progress() Progress {
	items := p.Items()
	done := 0
	for _, it := range items {
		if it.Checked {
			done++
		}
	}
	return Progress{done, len(items)}
}

// Progress counts every checked box in the plan.
func (pl *Plan) Progress() Progress {
	var total Progress
	for _, p := range pl.Phases {
		pp := p.Progress()
		total.Done += pp.Done
		total.Total += pp.Total
	}
	return total
}

// Current is the lowest-numbered phase with an unchecked box, or nil when
// every phase is done.
func (pl *Plan) Current() *Phase {
	for _, p := range pl.Phases {
		if slices.ContainsFunc(p.Items(), func(i *Item) bool { return !i.Checked }) {
			return p
		}
	}
	return nil
}
