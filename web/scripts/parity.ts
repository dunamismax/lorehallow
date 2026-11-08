// The parity check (BUILD.md 1.8): builds lorehallow.com with the Go generator
// and with Astro from the same commit, and compares them file by file. Every
// difference is either fixed or listed as accepted in
// docs/design/website-parity.md and in ACCEPTED below.
//
// Usage: node web/scripts/parity.ts [--go DIR --astro DIR]
//   With no arguments it builds both sites into temporary folders first.
import { execFileSync } from "node:child_process";
import { mkdtempSync, readdirSync, readFileSync, rmSync } from "node:fs";
import { tmpdir } from "node:os";
import { join } from "node:path";
import { decodePNG } from "../src/lib/png.ts";
import { woffTables } from "../src/lib/woff.ts";

const web = new URL("..", import.meta.url).pathname;

/**
 * Differences the parity check accepts, by URL path, each with its reason.
 * They change what is compared; they never skip a file.
 */
const ACCEPTED: { path: RegExp; why: string; normalize: (s: string) => string }[] = [
	{
		path: /^progress\.json$/,
		why: "generated is the build time",
		normalize: (s) => s.replace(/"generated": "[^"]*"/, '"generated": ""'),
	},
];

function files(dir: string): string[] {
	return readdirSync(dir, { recursive: true, withFileTypes: true })
		.filter((e) => e.isFile())
		.map((e) => join(e.parentPath, e.name).slice(dir.length + 1))
		.filter((f) => !f.startsWith("_astro/"))
		.sort();
}

/** "assets/site.0123456789.css" -> "assets/site.css": assets pair up by readable name. */
const unhash = (p: string) => p.replace(/\.[0-9a-f]{10}(\.[a-z0-9]+)$/, "$1");

const entities: Record<string, string> = { amp: "&", lt: "<", gt: ">", quot: '"', apos: "'", nbsp: " " };

function decodeEntities(s: string): string {
	return s.replace(/&(#x[0-9a-f]+|#[0-9]+|[a-z]+);/gi, (m, e: string) => {
		if (e[0] === "#") {
			const n = e[1] === "x" || e[1] === "X" ? Number.parseInt(e.slice(2), 16) : Number(e.slice(1));
			return String.fromCodePoint(n);
		}
		return entities[e.toLowerCase()] ?? m;
	});
}

/** HTML with whitespace between tags collapsed, entities decoded and asset hashes hidden. */
export function normalizeHTML(s: string): string {
	return decodeEntities(
		s
			.replace(/<!doctype html>/i, "<!doctype html>")
			.replace(/<!--[\s\S]*?-->/g, "")
			.replace(/\/assets\/([^"'\s)]+?)\.[0-9a-f]{10}\.([a-z0-9]+)/g, "/assets/$1.HASH.$2")
			.replace(/\s+/g, " ")
			.replace(/> </g, "><")
			.replace(/ >/g, ">"),
	).trim();
}

/** Where two strings first differ, with some context. */
function firstDifference(a: string, b: string): string {
	let i = 0;
	while (i < a.length && i < b.length && a[i] === b[i]) i++;
	const from = Math.max(0, i - 120);
	return `\n    go:    …${a.slice(from, i + 160)}…\n    astro: …${b.slice(from, i + 160)}…`;
}

function same(a: Uint8Array, b: Uint8Array): boolean {
	return Buffer.compare(Buffer.from(a), Buffer.from(b)) === 0;
}

export function compare(goDir: string, astroDir: string): string[] {
	const problems: string[] = [];
	const goFiles = new Map(files(goDir).map((f) => [unhash(f), f]));
	const astroFiles = new Map(files(astroDir).map((f) => [unhash(f), f]));
	for (const k of goFiles.keys()) if (!astroFiles.has(k)) problems.push(`missing from the Astro build: ${k}`);
	for (const k of astroFiles.keys()) if (!goFiles.has(k)) problems.push(`only in the Astro build: ${k}`);
	for (const [k, goName] of goFiles) {
		const astroName = astroFiles.get(k);
		if (!astroName) continue;
		const a = readFileSync(join(goDir, goName));
		const b = readFileSync(join(astroDir, astroName));
		const accepted = ACCEPTED.filter((x) => x.path.test(k));
		const text = (buf: Uint8Array) => accepted.reduce((s, x) => x.normalize(s), buf.toString());
		if (k.endsWith(".png")) {
			const pa = decodePNG(a, `go ${k}`);
			const pb = decodePNG(b, `astro ${k}`);
			if (pa.width !== pb.width || pa.height !== pb.height || !same(pa.data, pb.data)) {
				problems.push(`${k}: pixels differ (${pa.width}×${pa.height} vs ${pb.width}×${pb.height})`);
			}
		} else if (k.endsWith(".woff")) {
			const ta = woffTables(a);
			const tb = woffTables(b);
			for (const [tag, data] of ta) {
				const other = tb.get(tag);
				if (!other || !same(data, other)) problems.push(`${k}: font table ${tag} differs`);
			}
			if (tb.size !== ta.size) problems.push(`${k}: ${ta.size} font tables vs ${tb.size}`);
		} else if (k.endsWith(".html") || k.endsWith(".css") || k.endsWith(".xml") || k.endsWith(".svg")) {
			const na = normalizeHTML(text(a));
			const nb = normalizeHTML(text(b));
			if (na !== nb) problems.push(`${k}: differs${firstDifference(na, nb)}`);
		} else if (k.endsWith(".json") || k.endsWith(".txt")) {
			if (text(a) !== text(b)) problems.push(`${k}: differs${firstDifference(text(a), text(b))}`);
		} else if (!same(a, b)) {
			problems.push(`${k}: bytes differ`);
		}
	}
	return problems;
}

function main(args: string[]): number {
	let goDir = args[args.indexOf("--go") + 1];
	let astroDir = args[args.indexOf("--astro") + 1];
	let tmp = "";
	if (!args.includes("--go") || !args.includes("--astro")) {
		tmp = mkdtempSync(join(tmpdir(), "lorehallow-parity-"));
		goDir = join(tmp, "go");
		astroDir = join(tmp, "astro");
		execFileSync("go", ["run", "./cmd/lorehallow-web", "build", "-out", goDir], {
			cwd: web,
			stdio: "inherit",
		});
		execFileSync("npx", ["--no-install", "astro", "build", "--silent"], {
			cwd: web,
			stdio: "inherit",
			env: { ...process.env, LOREHALLOW_ASTRO_OUT: astroDir },
		});
	}
	try {
		const problems = compare(goDir as string, astroDir as string);
		for (const p of problems) console.error(`parity: ${p}`);
		if (problems.length > 0) {
			console.error(`parity: ${problems.length} difference(s) between the Go and Astro builds`);
			return 1;
		}
		console.log(`parity: the Go and Astro builds match (${files(goDir as string).length} files)`);
		return 0;
	} finally {
		if (tmp) rmSync(tmp, { recursive: true, force: true });
	}
}

if (import.meta.main) process.exitCode = main(process.argv.slice(2));
