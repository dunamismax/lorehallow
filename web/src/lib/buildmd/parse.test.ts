import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import { test } from "node:test";
import { currentPhase, overallProgress, parse, phaseProgress } from "./parse.ts";

const META = [
  "> **Goal:** A goal.",
  "> **Ships:** v0.0.0",
  "> **Lesson:** A lesson.",
  "> **Depends on:** Nothing",
].join("\n");

function phase(n: number, body: string, title = `Title ${n}`): string {
  return `## Phase ${n}: ${title}\n\n${META}\n\n${body}\n`;
}

const GOOD = [
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
    [
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
    ].join("\n"),
    "Start",
  ),
  phase(1, ["### 1.1 Only", "", "- [ ] `ADR` Decide.", "", "### Acceptance criteria", "", "- [ ] Ships."].join("\n"), "Next"),
  "## Deferred",
  "",
  "- not counted",
].join("\n");

const problemsOf = (text: string): string[] => parse(text).problems.map((p) => `${p.line}: ${p.message}`);

test("parses phases, metadata, sections, items and tags", () => {
  const plan = parse(GOOD);
  assert.deepEqual(plan.problems, []);
  assert.equal(plan.phases.length, 2);
  const [p0, p1] = plan.phases;
  assert.equal(p0.title, "Start");
  assert.equal(p0.meta.Goal, "A goal.");
  assert.equal(p0.meta["Depends on"], "Nothing");
  assert.deepEqual(
    p0.sections.map((s) => s.id),
    ["0.1", "acceptance"],
  );
  const items = p0.sections[0].items;
  assert.equal(items.length, 3);
  assert.deepEqual(
    items.map((i) => [i.checked, i.tag]),
    [
      [true, null],
      [false, "HUMAN"],
      [false, null],
    ],
  );
  assert.equal(items[1].text, "Owner thing.");
  assert.equal(items[2].text, "`server/cmd/x`: code in backticks is not a tag.");
  assert.equal(p1.sections[0].items[0].tag, "ADR");
});

test("computes progress and the current phase", () => {
  const plan = parse(GOOD);
  assert.deepEqual(phaseProgress(plan.phases[0]), { done: 2, total: 4 });
  assert.deepEqual(overallProgress(plan), { done: 2, total: 6 });
  assert.equal(currentPhase(plan)?.number, 0);
  const allDone = parse(GOOD.replaceAll("- [ ]", "- [x]"));
  assert.equal(currentPhase(allDone), null);
});

test("items outside phases are not counted", () => {
  const plan = parse(GOOD + "\n- [ ] stray\n");
  assert.equal(overallProgress(plan).total, 6);
  assert.match(problemsOf(GOOD + "\n- [ ] stray\n").join("\n"), /checkbox outside a phase/);
});

const cases: [string, string, RegExp][] = [
  ["nested checkbox", GOOD.replace("  - a note", "  - [ ] a nested box"), /column 0/],
  ["uppercase X", GOOD.replace("- [x] Done thing.", "- [X] Done thing."), /column 0/],
  ["star bullet", GOOD.replace("- [x] Done thing.", "* [x] Done thing."), /column 0/],
  ["unknown tag", GOOD.replace("`HUMAN` Owner", "`OWNER` Owner"), /unknown tag `OWNER`/],
  ["empty item", GOOD.replace("- [x] Done thing.", "- [x] "), /no text/],
  ["missing metadata", GOOD.replace("> **Lesson:** A lesson.\n", ""), /missing "> \*\*Lesson:\*\*"/],
  ["unknown metadata", GOOD.replace("> **Ships:**", "> **Shipz:**"), /unknown metadata key "Shipz"/],
  ["duplicate metadata", GOOD.replace("> **Ships:** v0.0.0", "> **Goal:** again"), /duplicate metadata key "Goal"/],
  ["malformed metadata", GOOD.replace("> **Ships:** v0.0.0", "> Ships: v0.0.0"), /metadata line must be/],
  ["phase out of order", GOOD.replace("## Phase 1: Next", "## Phase 2: Next"), /phase 2 is out of order; expected Phase 1/],
  ["bad phase heading", GOOD.replace("## Phase 1: Next", "## Phase 1 - Next"), /phase heading must be/],
  ["section in wrong phase", GOOD.replace("### 1.1 Only", "### 0.2 Only"), /section 0.2 is in Phase 1/],
  ["section out of order", GOOD.replace("### 1.1 Only", "### 1.2 Only"), /expected 1.1/],
  ["bad section heading", GOOD.replace("### 1.1 Only", "### Only"), /section heading must be/],
  ["no acceptance", GOOD.replace("### Acceptance criteria\n\n- [ ] Ships.", ""), /Phase 1 has no Acceptance criteria/],
  ["section after acceptance", GOOD.replace("- [ ] Ships.", "- [ ] Ships.\n\n### 1.2 Late\n\n- [ ] x"), /before Acceptance/],
  ["item before any section", GOOD.replace("### 1.1 Only\n", ""), /item outside a section/],
  ["empty section", GOOD.replace("- [ ] `ADR` Decide.", ""), /section 1.1 has no items/],
  ["h4 inside a phase", GOOD.replace("Some prose is fine here.", "#### Sub"), /only ## and ###/],
  ["overview title mismatch", GOOD.replace("| 1 | Next |", "| 1 | Later |"), /overview title "Later"/],
  ["overview missing phase", GOOD.replace("| 1 | Next | x | y |\n", ""), /missing from the Phase overview/],
  ["overview extra phase", GOOD.replace("| 1 | Next | x | y |", "| 1 | Next | x | y |\n| 2 | Ghost | x | y |"), /lists Phase 2/],
];

for (const [name, text, want] of cases) {
  test(`reports ${name}`, () => {
    const got = problemsOf(text);
    assert.ok(
      got.some((p) => want.test(p)),
      `want ${want}, got:\n${got.join("\n") || "(no problems)"}`,
    );
  });
}

test("problems carry line numbers", () => {
  const text = GOOD.replace("- [x] Done thing.", "- [X] Done thing.");
  const line = text.split("\n").findIndex((l) => l.startsWith("- [X]")) + 1;
  assert.ok(parse(text).problems.some((p) => p.line === line));
});

test("the real BUILD.md follows the contract", () => {
  const text = readFileSync(new URL("../../../../BUILD.md", import.meta.url), "utf8");
  const plan = parse(text);
  assert.deepEqual(plan.problems, []);
  assert.equal(plan.phases[0].number, 0);
  assert.ok(overallProgress(plan).done > 0);
});
