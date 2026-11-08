// The BUILD.md parser. It implements the format contract at the top of
// BUILD.md: phases, metadata, sections, items, tags and checked state, plus
// every format problem with its line number.
//
// The website renders the progress tracker from the result, and
// `make buildmd-lint` fails on any problem it reports. If the format changes,
// change this file, its tests and the contract together.
//
// It was ported from the Go parser of the retired Go site generator and held to
// it on real, fixture and randomly broken input until that was removed. The
// regular expressions keep Go's (RE2's) ASCII meaning of \s and \S, and
// trimming keeps Go's strings.TrimSpace, so the format contract didn't change.

/** Tags an item may carry, in backticks right after its checkbox. */
export const TAGS = ["HUMAN", "ADR"] as const;

/** The metadata lines every phase has, in order. */
export const META_KEYS = ["Goal", "Ships", "Lesson", "Depends on"] as const;

/** The ID of a phase's acceptance-criteria section. */
export const ACCEPTANCE = "acceptance";

export type Tag = (typeof TAGS)[number];

/** One checkbox line. */
export interface Item {
	/** Without the checkbox and tag. */
	text: string;
	checked: boolean;
	/** "HUMAN", "ADR" or "" (left out of JSON when empty, as in Go). */
	tag: Tag | "";
	/** 1-based line number in BUILD.md. */
	line: number;
}

/** A "### N.M Title" section, or the acceptance criteria. */
export interface Section {
	/** "N.M", or ACCEPTANCE. */
	id: string;
	title: string;
	items: Item[];
	line: number;
}

/** A "## Phase N: Title" block. */
export interface Phase {
	number: number;
	title: string;
	meta: Record<string, string>;
	sections: Section[];
	line: number;
}

/** One format-contract violation. */
export interface Problem {
	line: number;
	message: string;
}

/** A parsed BUILD.md. */
export interface Plan {
	phases: Phase[];
	problems: Problem[];
}

/** Checked boxes out of all boxes. */
export interface Progress {
	done: number;
	total: number;
}

const phaseRE = /^## Phase (\d+): ([^\t\n\f\r ][^\n]*)$/u;
const badPhaseRE = /^## Phase[\t\n\f\r ]*\d/u;
const metaRE = /^> \*\*([^*]+):\*\* ([^\t\n\f\r ][^\n]*)$/u;
const sectionRE = /^### (\d+)\.(\d+) ([^\t\n\f\r ][^\n]*)$/u;
const itemRE = /^- \[( |x)\] ([^\n]*)$/u;
/** Anything that looks like a checkbox. */
const checkboxishRE = /^[\t\n\f\r ]*[-*+] \[[^\]]?\]/u;
const tagRE = /^`([A-Z]+)` ([^\t\n\f\r ][^\n]*)$/u;
const overviewRowRE = /^\| (\d+) \| ([^|]+?) \|/u;

const acceptanceHeading = "### Acceptance criteria";

// Go's unicode.IsSpace: the characters strings.TrimSpace removes. JavaScript's
// trim() also removes U+FEFF and treats a few others differently.
const goSpace = new Set([
	0x09, 0x0a, 0x0b, 0x0c, 0x0d, 0x20, 0x85, 0xa0, 0x1680, 0x2000, 0x2001, 0x2002, 0x2003, 0x2004, 0x2005,
	0x2006, 0x2007, 0x2008, 0x2009, 0x200a, 0x2028, 0x2029, 0x202f, 0x205f, 0x3000,
]);

/** strings.TrimSpace. */
export function trimSpace(s: string): string {
	let start = 0;
	let end = s.length;
	while (start < end && goSpace.has(s.charCodeAt(start))) start++;
	while (end > start && goSpace.has(s.charCodeAt(end - 1))) end--;
	return s.slice(start, end);
}

/** strconv.Atoi for a string of ASCII digits. */
function atoi(digits: string): number {
	return Number.parseInt(digits, 10);
}

/** Reads BUILD.md text into phases, with every format problem found. */
export function parse(text: string): Plan {
	const plan: Plan = { phases: [], problems: [] };
	const problem = (line: number, message: string) => {
		plan.problems.push({ line, message });
	};
	const overview = new Map<number, { title: string; line: number }>();
	const overviewOrder: number[] = [];

	let phase: Phase | null = null;
	let section: Section | null = null;
	let inMeta = false; // still inside the metadata blockquote under a phase heading
	let inOverview = false;

	const lines = text.split("\n");
	for (let index = 0; index < lines.length; index++) {
		const n = index + 1;
		const raw = lines[index] ?? "";
		const line = raw.endsWith("\r") ? raw.slice(0, -1) : raw;

		if (line.startsWith("## ")) {
			inOverview = line === "## Phase overview";
			const m = phaseRE.exec(line);
			if (m) {
				const number = atoi(m[1] ?? "");
				const last = plan.phases.at(-1);
				const expected = last ? last.number + 1 : 0;
				if (number !== expected) {
					problem(n, `phase ${number} is out of order; expected Phase ${expected}`);
				}
				phase = { number, title: trimSpace(m[2] ?? ""), meta: {}, sections: [], line: n };
				plan.phases.push(phase);
				section = null;
				inMeta = true;
				continue;
			}
			if (badPhaseRE.test(line)) {
				problem(n, `phase heading must be "## Phase <N>: <Title>"`);
			}
			phase = null;
			section = null;
			inMeta = false;
			continue;
		}

		if (inOverview) {
			const row = overviewRowRE.exec(line);
			if (row) {
				const number = atoi(row[1] ?? "");
				if (!overview.has(number)) overviewOrder.push(number);
				overview.set(number, { title: trimSpace(row[2] ?? ""), line: n });
			}
		}

		if (phase === null) {
			if (checkboxishRE.test(line)) problem(n, "checkbox outside a phase");
			continue;
		}

		if (inMeta) {
			if (trimSpace(line) === "") {
				if (Object.keys(phase.meta).length > 0) inMeta = false;
				continue;
			}
			if (line.startsWith(">")) {
				const m = metaRE.exec(line);
				const key = m?.[1] ?? "";
				if (!m) {
					problem(n, `metadata line must be "> **<Key>:** <text>"`);
				} else if (!(META_KEYS as readonly string[]).includes(key)) {
					problem(n, `unknown metadata key "${key}"; use ${META_KEYS.join(", ")}`);
				} else if (Object.hasOwn(phase.meta, key)) {
					problem(n, `duplicate metadata key "${key}"`);
				} else {
					phase.meta[key] = trimSpace(m[2] ?? "");
				}
				continue;
			}
			inMeta = false;
		} else if (line.startsWith("> **")) {
			problem(n, "metadata must sit directly under the phase heading");
			continue;
		}

		if (line.startsWith("### ")) {
			if (line === acceptanceHeading) {
				if (has(phase, ACCEPTANCE)) problem(n, "duplicate Acceptance criteria section");
				section = { id: ACCEPTANCE, title: "Acceptance criteria", items: [], line: n };
				phase.sections.push(section);
				continue;
			}
			const m = sectionRE.exec(line);
			if (!m) {
				problem(n, `section heading must be "### <N>.<M> <Title>" or "### Acceptance criteria"`);
				section = null;
				continue;
			}
			const major = atoi(m[1] ?? "");
			const minor = atoi(m[2] ?? "");
			if (major !== phase.number) {
				problem(n, `section ${major}.${minor} is in Phase ${phase.number}`);
			}
			const numbered = phase.sections.filter((s) => s.id !== ACCEPTANCE).length;
			if (minor !== numbered + 1) {
				problem(n, `section ${major}.${minor} is out of order; expected ${phase.number}.${numbered + 1}`);
			}
			if (has(phase, ACCEPTANCE)) {
				problem(n, "numbered sections must come before Acceptance criteria");
			}
			section = { id: `${major}.${minor}`, title: trimSpace(m[3] ?? ""), items: [], line: n };
			phase.sections.push(section);
			continue;
		}

		if (line.startsWith("#")) {
			problem(n, "only ## and ### headings are allowed inside a phase");
			continue;
		}

		const m = itemRE.exec(line);
		if (m) {
			if (section === null) {
				problem(n, "item outside a section");
				continue;
			}
			let text = m[2] ?? "";
			let tag: Tag | "" = "";
			const t = tagRE.exec(text);
			if (t) {
				const name = t[1] ?? "";
				if ((TAGS as readonly string[]).includes(name)) {
					tag = name as Tag;
					text = t[2] ?? "";
				} else {
					problem(n, `unknown tag \`${name}\`; use \`${TAGS.join("` or `")}\``);
				}
			}
			if (trimSpace(text) === "") {
				problem(n, "item has no text");
				continue;
			}
			section.items.push({ text: trimSpace(text), checked: m[1] === "x", tag, line: n });
			continue;
		}
		if (checkboxishRE.test(line)) {
			problem(n, `checkboxes must start at column 0 as "- [ ] " or "- [x] " (no nesting)`);
		}
	}

	for (const p of plan.phases) {
		for (const key of META_KEYS) {
			if (!Object.hasOwn(p.meta, key)) problem(p.line, `Phase ${p.number} is missing "> **${key}:**"`);
		}
		if (!has(p, ACCEPTANCE)) problem(p.line, `Phase ${p.number} has no Acceptance criteria section`);
		for (const s of p.sections) {
			if (s.items.length === 0) problem(s.line, `section ${s.id} has no items`);
		}
	}

	if (overview.size > 0) {
		for (const p of plan.phases) {
			const row = overview.get(p.number);
			if (!row) {
				problem(p.line, `Phase ${p.number} is missing from the Phase overview table`);
			} else if (row.title !== p.title) {
				problem(row.line, `overview title "${row.title}" doesn't match "## Phase ${p.number}: ${p.title}"`);
			}
		}
		for (const number of overviewOrder) {
			if (!plan.phases.some((p) => p.number === number)) {
				problem(overview.get(number)?.line ?? 0, `overview lists Phase ${number}, which has no heading`);
			}
		}
	}

	plan.problems.sort((a, b) => a.line - b.line); // stable, like sort.SliceStable
	return plan;
}

function has(p: Phase, id: string): boolean {
	return p.sections.some((s) => s.id === id);
}

/** Every item in the phase, acceptance criteria included. */
export function phaseItems(p: Phase): Item[] {
	return p.sections.flatMap((s) => s.items);
}

/** The phase's checked boxes. */
export function phaseProgress(p: Phase): Progress {
	const items = phaseItems(p);
	return { done: items.filter((i) => i.checked).length, total: items.length };
}

/** Every checked box in the plan. */
export function planProgress(plan: Plan): Progress {
	const total = { done: 0, total: 0 };
	for (const p of plan.phases) {
		const pp = phaseProgress(p);
		total.done += pp.done;
		total.total += pp.total;
	}
	return total;
}

/** The lowest-numbered phase with an unchecked box, or null when every phase is done. */
export function currentPhase(plan: Plan): Phase | null {
	return plan.phases.find((p) => phaseItems(p).some((i) => !i.checked)) ?? null;
}

/** One problem as the linter prints it. */
export function formatProblem(p: Problem): string {
	return `${p.line}: ${p.message}`;
}
