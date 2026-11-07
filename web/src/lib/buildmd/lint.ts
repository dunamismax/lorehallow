// buildmd-lint: fail on any BUILD.md format-contract violation.
// Usage: node src/lib/buildmd/lint.ts [path/to/BUILD.md]
import { readFileSync } from "node:fs";
import { resolve } from "node:path";
import { currentPhase, overallProgress, parse } from "./parse.ts";

const path = resolve(process.argv[2] ?? "../BUILD.md");
const plan = parse(readFileSync(path, "utf8"));

for (const p of plan.problems) {
  console.error(`${path}:${p.line}: ${p.message}`);
}
if (plan.phases.length === 0) {
  console.error(`${path}: no phases found`);
  process.exit(1);
}
if (plan.problems.length > 0) {
  console.error(`buildmd-lint: ${plan.problems.length} problem(s)`);
  process.exit(1);
}
const { done, total } = overallProgress(plan);
const current = currentPhase(plan);
console.log(
  `buildmd-lint: ok, ${plan.phases.length} phases, ${done}/${total} items checked` +
    (current ? `, current phase ${current.number} (${current.title})` : ", all phases done"),
);
