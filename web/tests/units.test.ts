// Unit tests for the site's TypeScript modules, ported from the Go
// generator's tests.
import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import { test } from "node:test";
import { Assets } from "../src/lib/assets.ts";
import { postSchema } from "../src/lib/devlog-schema.ts";
import { wrapPixel } from "../src/lib/og.ts";
import { PixelFont } from "../src/lib/pixelfont.ts";
import { blank, decodePNG, encodePNG, mirrorPair } from "../src/lib/png.ts";
import { parseAchievements } from "../src/lib/progress.ts";
import { contrast, PAIRS, parsePalette, ROLES, Theme } from "../src/lib/theme.ts";
import { fromSFNT, woffTables } from "../src/lib/woff.ts";

const web = new URL("..", import.meta.url).pathname;
const read = (p: string) => readFileSync(new URL(p, new URL("..", import.meta.url)), "utf8");

test("the achievement feed pairs checks, rewords and moves", () => {
	const h = (c: string) => c.repeat(40);
	const log = `commit ${h("a")} 1700000000
@@ -1 +1 @@
-- [ ] Build the thing.
+- [x] Build the thing.
+- [x] \`ADR\` New and already done.
commit ${h("b")} 1700000100
@@ -1 +1 @@
-- [x] Build the thing.
+- [x] Build the whole thing.
-- [x] New and already done.
+- [x] New and already done.
commit ${h("c")} 1700000200
@@ -1 +1 @@
-- [ ] \`HUMAN\` Approve it.
+- [x] \`HUMAN\` Approve it.
`;
	assert.deepEqual(
		parseAchievements(log).map((a) => `${a.commit} ${a.text}`),
		["ccccccc Approve it.", "aaaaaaa New and already done.", "aaaaaaa Build the thing."],
	);
});

test("palette parsing and contrast", () => {
	assert.throws(() => parsePalette("ff00f ; bad\n"));
	assert.throws(() => parsePalette("ffffff\n"));
	const pal = parsePalette("; c\n1a1420 ; ink: outline\nf4ecd8 ; parchment white\n");
	assert.equal(pal.length, 2);
	assert.equal(pal[1]?.name, "parchment-white");
	assert.equal(pal[0]?.hex, "#1a1420");
	const c = contrast("#000000", "#ffffff");
	assert.ok(c > 20.99 && c < 21.01);
});

test("every text and background pair meets WCAG AA", () => {
	const theme = new Theme(parsePalette(read("../art/style/palette.hex")));
	const roles = new Set(ROLES.map((r) => r.name));
	for (const p of PAIRS) {
		assert.ok(roles.has(p.fg) && roles.has(p.bg), `${p.fg} on ${p.bg} names an unknown role`);
		const c = contrast(theme.roleHex(p.fg), theme.roleHex(p.bg));
		assert.ok(c >= (p.large ? 3 : 4.5), `--${p.fg} on --${p.bg} is ${c.toFixed(2)}:1`);
	}
});

test("the stylesheet sets colors only through roles in a contrast pair", () => {
	const roles = new Set(ROLES.map((r) => r.name));
	const fgs = new Set(PAIRS.map((p) => p.fg));
	const problems: string[] = [];
	for (const m of read("static/css/site.css").matchAll(
		/(?:^|[;{\s])(color|background-color)\s*:\s*([^;}]+)/gm,
	)) {
		const v = (m[2] ?? "").trim();
		if (["transparent", "inherit", "currentColor"].includes(v)) continue;
		const r = /^var\(--([a-z-]+)\)$/.exec(v);
		if (!r || !roles.has(r[1] ?? "")) problems.push(`${m[1]}: ${v}`);
		else if (m[1] === "color" && !fgs.has(r[1] ?? "")) problems.push(`color: var(--${r[1]}) is in no pair`);
	}
	assert.deepEqual(problems, []);
});

test("an image without alt text fails; decorative art gets an empty alt", () => {
	const a = new Assets(`${web}assets/art`);
	assert.throws(() => a.img("icons/emblem.png", { alt: "", k: 2 }), /no alt text/);
	const h = a.img("icons/emblem.png", { alt: "decorative", k: 2 });
	assert.match(h, /alt=""/);
	assert.match(h, /width="64"/);
});

test("PNGs round-trip with the same pixels, opaque or not", () => {
	const img = blank(3, 2);
	img.data.set([
		255, 0, 0, 255, 0, 255, 0, 128, 0, 0, 255, 0, 9, 9, 9, 255, 1, 2, 3, 255, 250, 251, 252, 255,
	]);
	assert.deepEqual(decodePNG(encodePNG(img)).data, img.data);
	const opaque = blank(2, 1);
	opaque.data.set([1, 2, 3, 255, 4, 5, 6, 255]);
	assert.deepEqual(decodePNG(encodePNG(opaque)).data, opaque.data);
	const pair = mirrorPair(img);
	assert.equal(pair.width, 6);
	assert.deepEqual(pair.data.subarray(0, 4), pair.data.subarray(20, 24));
	const art = decodePNG(readFileSync(`${web}assets/art/web/og_card.png`));
	assert.deepEqual(decodePNG(encodePNG(art)).data, art.data);
});

test("the pixel font covers every heading, compiles reproducibly and measures right", () => {
	const f = PixelFont.parse(read("static/fonts/lorehallow-pixel.txt"));
	assert.ok(f.has("Lorehallow: Phase 20 · You are here"));
	const a = f.ttf();
	assert.deepEqual(f.ttf(), a);
	assert.deepEqual([...woffTables(fromSFNT(a)).keys()].sort(), [
		"OS/2",
		"cmap",
		"glyf",
		"head",
		"hhea",
		"hmtx",
		"loca",
		"maxp",
		"name",
		"post",
	]);
	assert.equal(f.measure(""), 0);
	assert.equal(f.measure("ii"), 2 * ((f.glyphs.get(0x69)?.width ?? 0) + 1) - 1);
	assert.ok(wrapPixel(f, "Phase 13: Classes and the v1 World", 132).length <= 2);
	assert.throws(() => PixelFont.parse("name: x\n\nglyph A\n#.\n##\n"), /rows/);
	assert.throws(() => PixelFont.parse("glyph U+0020\n.\n.\n.\n.\n.\n.\n.\n"), /name/);
});

test("devlog front matter: all six fields, nothing else, each checked", () => {
	const schema = postSchema(`${web}assets/art`);
	const good = {
		title: "A: title",
		date: new Date("2026-11-07"),
		summary: "Plain words",
		tags: ["a", "b-c"],
		cover: null,
		draft: false,
	};
	assert.ok(schema.safeParse(good).success);
	const bad: Record<string, unknown> = {
		"unknown field": { ...good, author: "me" },
		"bad date": { ...good, date: "7 Nov 2026" },
		"bad draft": { ...good, draft: "maybe" },
		"bad tag": { ...good, tags: ["Big Tag"] },
		"missing cover": { ...good, cover: undefined },
		"missing art": { ...good, cover: "web/nope.png" },
		"empty title": { ...good, title: "" },
	};
	for (const [name, data] of Object.entries(bad))
		assert.ok(!schema.safeParse(data).success, `${name} accepted`);
	assert.ok(schema.safeParse({ ...good, cover: "web/devlog_header.png" }).success);
});

test("release data: valid entries pass, every bad field fails", async () => {
	const { parseReleases } = await import("../src/lib/releases.ts");
	const good = {
		releases: [
			{ version: "v0.3.0", date: "2027-01-10", channel: "alpha", notes: "/devlog/", protocol: 4 },
			{ version: "v0.2.0", date: "2026-12-01", channel: "alpha", notes: "", protocol: 3 },
		],
	};
	assert.equal(parseReleases(JSON.stringify(good)).length, 2);
	assert.deepEqual(parseReleases('{"releases": []}'), []);
	const first = good.releases[0];
	const bad: Record<string, unknown> = {
		version: { ...first, version: "0.3" },
		date: { ...first, date: "10 Jan 2027" },
		channel: { ...first, channel: "nightly" },
		notes: { ...first, notes: "http://x" },
		protocol: { ...first, protocol: 0 },
		"unknown field": { ...first, files: [] },
	};
	for (const [name, r] of Object.entries(bad)) {
		assert.throws(
			() => parseReleases(JSON.stringify({ releases: [r] })),
			/releases.json/,
			`${name} accepted`,
		);
	}
	assert.throws(
		() => parseReleases(JSON.stringify({ releases: [...good.releases].reverse() })),
		/newest first/,
	);
	assert.throws(() => parseReleases("{}"));
	assert.equal(parseReleases(readFileSync(`${web}data/releases.json`, "utf8")).length >= 0, true);
});
