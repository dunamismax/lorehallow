// BUILD.md parser: implements the format contract at the top of BUILD.md.
//
// The website renders the progress tracker from the result, and
// `make buildmd-lint` fails on any problem it reports. If the format
// changes, change this file, its tests and the contract together.
//
// Written for Node's built-in TypeScript type stripping (no enums, no
// parameter properties, explicit .ts imports), so `node` runs it directly.

export type Tag = "HUMAN" | "ADR";

export const TAGS: readonly Tag[] = ["HUMAN", "ADR"];

export const META_KEYS = ["Goal", "Ships", "Lesson", "Depends on"] as const;
export type MetaKey = (typeof META_KEYS)[number];

export interface Item {
  /** Item text without the checkbox and tag. */
  text: string;
  checked: boolean;
  tag: Tag | null;
  /** 1-based line number in BUILD.md. */
  line: number;
}

export interface Section {
  /** "N.M" for numbered sections, "acceptance" for acceptance criteria. */
  id: string;
  title: string;
  items: Item[];
  line: number;
}

export interface Phase {
  number: number;
  title: string;
  meta: Partial<Record<MetaKey, string>>;
  sections: Section[];
  line: number;
}

export interface Problem {
  line: number;
  message: string;
}

export interface BuildPlan {
  phases: Phase[];
  problems: Problem[];
}

export interface Progress {
  done: number;
  total: number;
}

const PHASE_RE = /^## Phase (\d+): (\S.*)$/;
const H2_RE = /^## /;
const META_RE = /^> \*\*([^*]+):\*\* (\S.*)$/;
const SECTION_RE = /^### (\d+)\.(\d+) (\S.*)$/;
const ACCEPTANCE = "### Acceptance criteria";
const ITEM_RE = /^- \[( |x)\] (.*)$/;
// Anything that looks like a checkbox, anywhere on a line.
const CHECKBOXISH_RE = /^\s*[-*+] \[[^\]]?\]/;
const TAG_RE = /^`([A-Z]+)` (\S.*)$/;
const OVERVIEW_ROW_RE = /^\| (\d+) \| ([^|]+?) \|/;

/** Parse BUILD.md text into phases, with every format problem found. */
export function parse(text: string): BuildPlan {
  const lines = text.split("\n");
  const phases: Phase[] = [];
  const problems: Problem[] = [];
  const overview = new Map<number, { title: string; line: number }>();

  let phase: Phase | null = null;
  let section: Section | null = null;
  // True while still inside the metadata blockquote directly under a phase heading.
  let inMeta = false;
  let inOverview = false;

  const problem = (line: number, message: string): void => {
    problems.push({ line, message });
  };

  lines.forEach((raw, index) => {
    const n = index + 1;
    const line = raw.replace(/\r$/, "");

    if (H2_RE.test(line)) {
      inOverview = line === "## Phase overview";
      const m = PHASE_RE.exec(line);
      if (m) {
        const number = Number(m[1]);
        const expected = phases.length === 0 ? 0 : phases[phases.length - 1].number + 1;
        if (number !== expected) {
          problem(n, `phase ${number} is out of order; expected Phase ${expected}`);
        }
        phase = { number, title: m[2].trim(), meta: {}, sections: [], line: n };
        phases.push(phase);
        section = null;
        inMeta = true;
        return;
      }
      if (/^## Phase\s*\d/.test(line)) {
        problem(n, 'phase heading must be "## Phase <N>: <Title>"');
      }
      phase = null;
      section = null;
      inMeta = false;
      return;
    }

    if (inOverview) {
      const row = OVERVIEW_ROW_RE.exec(line);
      if (row) overview.set(Number(row[1]), { title: row[2].trim(), line: n });
    }

    if (phase === null) {
      if (CHECKBOXISH_RE.test(line)) problem(n, "checkbox outside a phase");
      return;
    }

    if (inMeta) {
      if (line.trim() === "") {
        if (Object.keys(phase.meta).length > 0) inMeta = false;
        return;
      }
      if (line.startsWith(">")) {
        const m = META_RE.exec(line);
        if (!m) {
          problem(n, 'metadata line must be "> **<Key>:** <text>"');
          return;
        }
        const key = m[1] as MetaKey;
        if (!META_KEYS.includes(key)) {
          problem(n, `unknown metadata key "${m[1]}"; use ${META_KEYS.join(", ")}`);
        } else if (phase.meta[key] !== undefined) {
          problem(n, `duplicate metadata key "${key}"`);
        } else {
          phase.meta[key] = m[2].trim();
        }
        return;
      }
      inMeta = false;
    } else if (line.startsWith("> **")) {
      problem(n, "metadata must sit directly under the phase heading");
      return;
    }

    if (line.startsWith("### ")) {
      if (line === ACCEPTANCE) {
        if (phase.sections.some((s) => s.id === "acceptance")) {
          problem(n, "duplicate Acceptance criteria section");
        }
        section = { id: "acceptance", title: "Acceptance criteria", items: [], line: n };
        phase.sections.push(section);
        return;
      }
      const m = SECTION_RE.exec(line);
      if (!m) {
        problem(n, 'section heading must be "### <N>.<M> <Title>" or "### Acceptance criteria"');
        section = null;
        return;
      }
      const [major, minor] = [Number(m[1]), Number(m[2])];
      if (major !== phase.number) {
        problem(n, `section ${major}.${minor} is in Phase ${phase.number}`);
      }
      const numbered = phase.sections.filter((s) => s.id !== "acceptance");
      if (minor !== numbered.length + 1) {
        problem(n, `section ${major}.${minor} is out of order; expected ${phase.number}.${numbered.length + 1}`);
      }
      if (phase.sections.some((s) => s.id === "acceptance")) {
        problem(n, "numbered sections must come before Acceptance criteria");
      }
      section = { id: `${major}.${minor}`, title: m[3].trim(), items: [], line: n };
      phase.sections.push(section);
      return;
    }

    if (line.startsWith("#")) {
      problem(n, "only ## and ### headings are allowed inside a phase");
      return;
    }

    const item = ITEM_RE.exec(line);
    if (item) {
      if (section === null) {
        problem(n, "item outside a section");
        return;
      }
      let text = item[2];
      let tag: Tag | null = null;
      const t = TAG_RE.exec(text);
      if (t && /^[A-Z]+$/.test(t[1])) {
        if ((TAGS as readonly string[]).includes(t[1])) {
          tag = t[1] as Tag;
          text = t[2];
        } else {
          problem(n, `unknown tag \`${t[1]}\`; use ${TAGS.map((x) => "`" + x + "`").join(" or ")}`);
        }
      }
      if (text.trim() === "") {
        problem(n, "item has no text");
        return;
      }
      section.items.push({ text: text.trim(), checked: item[1] === "x", tag, line: n });
      return;
    }
    if (CHECKBOXISH_RE.test(line)) {
      problem(n, 'checkboxes must start at column 0 as "- [ ] " or "- [x] " (no nesting)');
    }
  });

  for (const p of phases) {
    for (const key of META_KEYS) {
      if (p.meta[key] === undefined) problem(p.line, `Phase ${p.number} is missing "> **${key}:**"`);
    }
    if (!p.sections.some((s) => s.id === "acceptance")) {
      problem(p.line, `Phase ${p.number} has no Acceptance criteria section`);
    }
    for (const s of p.sections) {
      if (s.items.length === 0) problem(s.line, `section ${s.id} has no items`);
    }
  }

  if (overview.size > 0) {
    for (const p of phases) {
      const row = overview.get(p.number);
      if (!row) {
        problem(p.line, `Phase ${p.number} is missing from the Phase overview table`);
      } else if (row.title !== p.title) {
        problem(row.line, `overview title "${row.title}" doesn't match "## Phase ${p.number}: ${p.title}"`);
      }
    }
    for (const [number, row] of overview) {
      if (!phases.some((p) => p.number === number)) {
        problem(row.line, `overview lists Phase ${number}, which has no heading`);
      }
    }
  }

  problems.sort((a, b) => a.line - b.line);
  return { phases, problems };
}

/** Every item in a phase, acceptance criteria included. */
export function phaseItems(phase: Phase): Item[] {
  return phase.sections.flatMap((s) => s.items);
}

export function phaseProgress(phase: Phase): Progress {
  const items = phaseItems(phase);
  return { done: items.filter((i) => i.checked).length, total: items.length };
}

export function overallProgress(plan: BuildPlan): Progress {
  return plan.phases.map(phaseProgress).reduce((a, b) => ({ done: a.done + b.done, total: a.total + b.total }), {
    done: 0,
    total: 0,
  });
}

/** The lowest-numbered phase with an unchecked box, or null when all are done. */
export function currentPhase(plan: BuildPlan): Phase | null {
  return plan.phases.find((p) => phaseItems(p).some((i) => !i.checked)) ?? null;
}
