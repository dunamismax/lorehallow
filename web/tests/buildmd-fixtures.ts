// The Go parser's test fixtures (web/internal/buildmd/parse_test.go), shared
// by the TypeScript parser's tests and the parity test.

const meta = "> **Goal:** A goal.\n> **Ships:** v0.0.0\n> **Lesson:** A lesson.\n> **Depends on:** Nothing";

function phase(n: number, title: string, ...body: string[]): string {
	return `## Phase ${n}: ${title}\n\n${meta}\n\n${body.join("\n")}\n`;
}

export const good = [
	"# Plan",
	"",
	"## Phase overview",
	"",
	"| Phase | Title | Ships | Core lesson |",
	"|---|---|---|---|",
	"| 0 | Start | x | y |",
	"| 1 | Next | x | y |",
	"",
	phase(
		0,
		"Start",
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
	phase(
		1,
		"Next",
		"### 1.1 Only",
		"",
		"- [ ] `ADR` Decide.",
		"",
		"### Acceptance criteria",
		"",
		"- [ ] Ships.",
	),
	"## Deferred",
	"",
	"- not counted",
].join("\n");

/** Swaps the first occurrence, like the Go tests' replace. */
export function replace(s: string, old: string, repl: string): string {
	return s.replace(old, () => repl);
}

/** Each broken copy of `good` and a pattern its problems must match. */
export const problemCases: { name: string; text: string; want: RegExp }[] = [
	{ name: "nested checkbox", text: replace(good, "  - a note", "  - [ ] a nested box"), want: /column 0/ },
	{ name: "uppercase X", text: replace(good, "- [x] Done thing.", "- [X] Done thing."), want: /column 0/ },
	{ name: "star bullet", text: replace(good, "- [x] Done thing.", "* [x] Done thing."), want: /column 0/ },
	{ name: "unknown tag", text: replace(good, "`HUMAN` Owner", "`OWNER` Owner"), want: /unknown tag `OWNER`/ },
	{ name: "empty item", text: replace(good, "- [x] Done thing.", "- [x] "), want: /no text/ },
	{
		name: "missing metadata",
		text: replace(good, "> **Lesson:** A lesson.\n", ""),
		want: /missing "> \*\*Lesson:\*\*"/,
	},
	{
		name: "unknown metadata",
		text: replace(good, "> **Ships:**", "> **Shipz:**"),
		want: /unknown metadata key "Shipz"/,
	},
	{
		name: "duplicate metadata",
		text: replace(good, "> **Ships:** v0.0.0", "> **Goal:** again"),
		want: /duplicate metadata key "Goal"/,
	},
	{
		name: "malformed metadata",
		text: replace(good, "> **Ships:** v0.0.0", "> Ships: v0.0.0"),
		want: /metadata line must be/,
	},
	{
		name: "phase out of order",
		text: replace(good, "## Phase 1: Next", "## Phase 2: Next"),
		want: /phase 2 is out of order; expected Phase 1/,
	},
	{
		name: "bad phase heading",
		text: replace(good, "## Phase 1: Next", "## Phase 1 - Next"),
		want: /phase heading must be/,
	},
	{
		name: "section in wrong phase",
		text: replace(good, "### 1.1 Only", "### 0.2 Only"),
		want: /section 0.2 is in Phase 1/,
	},
	{ name: "section out of order", text: replace(good, "### 1.1 Only", "### 1.2 Only"), want: /expected 1.1/ },
	{
		name: "bad section heading",
		text: replace(good, "### 1.1 Only", "### Only"),
		want: /section heading must be/,
	},
	{
		name: "no acceptance",
		text: replace(good, "### Acceptance criteria\n\n- [ ] Ships.", ""),
		want: /Phase 1 has no Acceptance criteria/,
	},
	{
		name: "section after acceptance",
		text: replace(good, "- [ ] Ships.", "- [ ] Ships.\n\n### 1.2 Late\n\n- [ ] x"),
		want: /before Acceptance/,
	},
	{
		name: "item before any section",
		text: replace(good, "### 1.1 Only\n", ""),
		want: /item outside a section/,
	},
	{ name: "empty section", text: replace(good, "- [ ] `ADR` Decide.", ""), want: /section 1.1 has no items/ },
	{
		name: "h4 inside a phase",
		text: replace(good, "Some prose is fine here.", "#### Sub"),
		want: /only ## and ###/,
	},
	{
		name: "overview title mismatch",
		text: replace(good, "| 1 | Next |", "| 1 | Later |"),
		want: /overview title "Later"/,
	},
	{
		name: "overview missing phase",
		text: replace(good, "| 1 | Next | x | y |\n", ""),
		want: /missing from the Phase overview/,
	},
	{
		name: "overview extra phase",
		text: replace(good, "| 1 | Next | x | y |", "| 1 | Next | x | y |\n| 2 | Ghost | x | y |"),
		want: /lists Phase 2/,
	},
];

/** The Go fuzz target's seed corpus (FuzzParse). */
export const fuzzSeeds = [good, "## Phase 0: X\n> **Goal:** y\n### 0.1 Z\n- [ ] `HUMAN` a\n"];
