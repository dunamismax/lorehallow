// The TypeScript BUILD.md parser must return exactly what the Go parser in
// web/internal/buildmd returns: on the real BUILD.md, on every fixture in the
// Go tests and the fuzz seed corpus, and on randomly broken copies of
// BUILD.md (BUILD.md 1.4). Runs in `make check` until the Go parser is retired.
import assert from "node:assert/strict";
import { execFileSync } from "node:child_process";
import { mkdtempSync, readFileSync, rmSync, writeFileSync } from "node:fs";
import { tmpdir } from "node:os";
import { join } from "node:path";
import { test } from "node:test";
import { type Plan, parse } from "../src/lib/buildmd.ts";
import { fuzzSeeds, good, problemCases } from "./buildmd-fixtures.ts";

const webDir = new URL("..", import.meta.url).pathname;
const real = readFileSync(new URL("../../BUILD.md", import.meta.url), "utf8");

/** A small seeded generator, so a failure can be reproduced. */
function rng(seed: number): () => number {
	let s = seed >>> 0;
	return () => {
		s = (s + 0x6d2b79f5) >>> 0;
		let t = s;
		t = Math.imul(t ^ (t >>> 15), t | 1);
		t ^= t + Math.imul(t ^ (t >>> 7), t | 61);
		return ((t ^ (t >>> 14)) >>> 0) / 2 ** 32;
	};
}

/** Breaks BUILD.md the ways a careless edit would. */
function breakCopy(text: string, seed: number): string {
	const rand = rng(seed);
	const pick = <T>(xs: T[]): T => xs[Math.floor(rand() * xs.length)] as T;
	const lines = text.split("\n");
	const edits = 1 + Math.floor(rand() * 4);
	for (let e = 0; e < edits; e++) {
		const i = Math.floor(rand() * lines.length);
		const line = lines[i] ?? "";
		switch (Math.floor(rand() * 10)) {
			case 0:
				lines.splice(i, 1);
				break;
			case 1:
				lines.splice(i, 0, line);
				break;
			case 2:
				lines[i] = line.replace("- [x]", "- [X]").replace("- [ ]", "  - [ ]");
				break;
			case 3:
				lines[i] = line.replace(/`(HUMAN|ADR)`/, "`OWNER`");
				break;
			case 4:
				lines[i] = line.replace(/### (\d+)\.(\d+)/, (_, a, b) => `### ${a}.${Number(b) + 1}`);
				break;
			case 5:
				lines[i] = line.replace(/## Phase (\d+):/, (_, a) => `## Phase ${Number(a) + 2}:`);
				break;
			case 6:
				lines[i] = line.replace("> **Goal:**", pick(["> **Gaol:**", "> Goal:", "> **Ships:**"]));
				break;
			case 7:
				lines[i] = `${line}\r`;
				break;
			case 8:
				lines.splice(
					i,
					0,
					pick(["#### Note", "- [ ] stray", "> **Lesson:** late", "### Acceptance criteria"]),
				);
				break;
			default:
				lines[i] = line.slice(0, Math.floor(rand() * line.length));
		}
	}
	return lines.join("\n");
}

/** Drops Go's omitempty fields so the two JSON shapes compare directly. */
function normalize(plan: Plan): unknown {
	return JSON.parse(
		JSON.stringify(plan, (key, value) => (key === "tag" && value === "" ? undefined : value)),
	);
}

test("the TypeScript parser matches the Go parser", () => {
	const inputs: { name: string; text: string }[] = [
		{ name: "BUILD.md", text: real },
		{ name: "good", text: good },
		{ name: "good, CRLF", text: good.replaceAll("\n", "\r\n") },
		...problemCases.map((c) => ({ name: c.name, text: c.text })),
		...fuzzSeeds.map((text, i) => ({ name: `fuzz seed ${i}`, text })),
	];
	for (let seed = 1; seed <= 300; seed++) {
		inputs.push({ name: `broken BUILD.md, seed ${seed}`, text: breakCopy(real, seed) });
	}
	const dir = mkdtempSync(join(tmpdir(), "buildmd-parity-"));
	try {
		const files = inputs.map((input, i) => {
			const file = join(dir, `${i}.md`);
			writeFileSync(file, input.text);
			return file;
		});
		const out = execFileSync("go", ["run", "./cmd/buildmd-lint", "-json", ...files], {
			cwd: webDir,
			maxBuffer: 1 << 30,
			encoding: "utf8",
		});
		const goPlans = JSON.parse(out) as unknown[];
		assert.equal(goPlans.length, inputs.length);
		let broken = 0;
		inputs.forEach((input, i) => {
			const plan = parse(input.text);
			assert.deepEqual(normalize(plan), goPlans[i], `parsers differ on ${input.name}`);
			if (input.name.startsWith("broken") && plan.problems.length > 0) broken++;
		});
		// The broken copies only test something if most of them are broken.
		assert.ok(broken >= 100, `only ${broken} of 300 broken copies have problems`);
	} finally {
		rmSync(dir, { recursive: true, force: true });
	}
});
