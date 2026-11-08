// Tests for the BUILD.md parser, ported from web/internal/buildmd/parse_test.go.
import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import { test } from "node:test";
import { currentPhase, formatProblem, parse, phaseProgress, planProgress } from "../src/lib/buildmd.ts";
import { fuzzSeeds, good, problemCases, replace } from "./buildmd-fixtures.ts";

const problemsOf = (text: string) => parse(text).problems.map(formatProblem);

test("parses phases, metadata, sections, items and tags", () => {
	const plan = parse(good);
	assert.deepEqual(plan.problems, []);
	assert.equal(plan.phases.length, 2);
	const [p0, p1] = plan.phases;
	assert.ok(p0 && p1);
	assert.equal(p0.title, "Start");
	assert.equal(p0.meta.Goal, "A goal.");
	assert.equal(p0.meta["Depends on"], "Nothing");
	assert.deepEqual(
		p0.sections.map((s) => s.id),
		["0.1", "acceptance"],
	);
	const items = p0.sections[0]?.items ?? [];
	assert.deepEqual(
		items.map((i) => [i.checked, i.tag]),
		[
			[true, ""],
			[false, "HUMAN"],
			[false, ""],
		],
	);
	assert.equal(items[1]?.text, "Owner thing.");
	assert.equal(items[2]?.text, "`server/cmd/x`: code in backticks is not a tag.");
	assert.equal(p1.sections[0]?.items[0]?.tag, "ADR");
});

test("progress and the current phase", () => {
	const plan = parse(good);
	assert.ok(plan.phases[0]);
	assert.deepEqual(phaseProgress(plan.phases[0]), { done: 2, total: 4 });
	assert.deepEqual(planProgress(plan), { done: 2, total: 6 });
	assert.equal(currentPhase(plan)?.number, 0);
	assert.equal(currentPhase(parse(good.replaceAll("- [ ]", "- [x]"))), null);
});

test("items outside phases are not counted", () => {
	const text = `${good}\n- [ ] stray\n`;
	assert.equal(planProgress(parse(text)).total, 6);
	assert.ok(problemsOf(text).some((p) => p.includes("checkbox outside a phase")));
});

for (const c of problemCases) {
	test(`reports format problems: ${c.name}`, () => {
		const got = problemsOf(c.text);
		assert.ok(
			got.some((p) => c.want.test(p)),
			`want ${c.want}, got:\n${got.join("\n")}`,
		);
	});
}

test("problems carry line numbers", () => {
	const text = replace(good, "- [x] Done thing.", "- [X] Done thing.");
	const line = text.split("\n").findIndex((l) => l.startsWith("- [X]")) + 1;
	assert.ok(parse(text).problems.some((p) => p.line === line));
});

test("CRLF line endings", () => {
	const plan = parse(good.replaceAll("\n", "\r\n"));
	assert.deepEqual(plan.problems, []);
	assert.deepEqual(planProgress(plan), { done: 2, total: 6 });
});

test("the real BUILD.md follows the contract", () => {
	const plan = parse(readFileSync(new URL("../../BUILD.md", import.meta.url), "utf8"));
	assert.deepEqual(plan.problems, []);
	assert.equal(plan.phases[0]?.number, 0);
	assert.ok(planProgress(plan).done > 0);
});

test("random input never breaks the invariants", () => {
	let seed = 1;
	const rand = () => {
		seed = (seed * 1103515245 + 12345) % 2 ** 31;
		return seed / 2 ** 31;
	};
	const pieces = [
		"## Phase 0: X",
		"> **Goal:** y",
		"### 0.1 Z",
		"- [ ] `HUMAN` a",
		"- [x] b",
		"\n",
		" ",
		"#",
	];
	for (let i = 0; i < 500; i++) {
		let text = fuzzSeeds[i % fuzzSeeds.length] ?? "";
		for (let j = 0; j < 10; j++) {
			const at = Math.floor(rand() * text.length);
			text = text.slice(0, at) + (pieces[Math.floor(rand() * pieces.length)] ?? "") + text.slice(at + 1);
		}
		const plan = parse(text);
		const p = planProgress(plan);
		assert.ok(p.done >= 0 && p.done <= p.total);
		for (let k = 1; k < plan.problems.length; k++) {
			assert.ok((plan.problems[k]?.line ?? 0) >= (plan.problems[k - 1]?.line ?? 0));
		}
	}
});
