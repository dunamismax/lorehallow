// buildmd-lint fails on any BUILD.md format-contract violation.
//
// Usage: node web/scripts/buildmd-lint.ts [path/to/BUILD.md]   (default: the repository's BUILD.md)
import { readFileSync } from "node:fs";
import { resolve } from "node:path";
import { currentPhase, parse, planProgress } from "../src/lib/buildmd.ts";

function run(args: string[]): number {
	const path = resolve(args[0] ?? new URL("../../BUILD.md", import.meta.url).pathname);
	let text: string;
	try {
		text = readFileSync(path, "utf8");
	} catch (err) {
		console.error(`buildmd-lint: ${(err as Error).message}`);
		return 1;
	}
	const plan = parse(text);
	for (const p of plan.problems) console.error(`${path}:${p.line}: ${p.message}`);
	if (plan.phases.length === 0) {
		console.error(`${path}: no phases found`);
		return 1;
	}
	if (plan.problems.length > 0) {
		console.error(`buildmd-lint: ${plan.problems.length} problem(s)`);
		return 1;
	}
	const pr = planProgress(plan);
	let msg = `buildmd-lint: ok, ${plan.phases.length} phases, ${pr.done}/${pr.total} items checked`;
	const c = currentPhase(plan);
	msg += c ? `, current phase ${c.number} (${c.title})` : ", all phases done";
	console.log(msg);
	return 0;
}

process.exitCode = run(process.argv.slice(2));
