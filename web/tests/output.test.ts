// Output tests on the real Astro build (BUILD.md 1.8), ported from the Go
// generator's (web/internal/site/site_test.go). The site is built once, from
// the repository, with an extra draft post that must not appear anywhere.
import assert from "node:assert/strict";
import { execFileSync } from "node:child_process";
import { cpSync, mkdtempSync, readdirSync, readFileSync, rmSync, writeFileSync } from "node:fs";
import { tmpdir } from "node:os";
import { join } from "node:path";
import { after, before, test } from "node:test";
import { decodePNG } from "../src/lib/png.ts";
import { BaseURL, CodebergURL, RepoURL } from "../src/lib/state.ts";
import { type Tag, tags, title } from "./html.ts";

const web = new URL("..", import.meta.url).pathname;
let tmp = "";
let out = "";
const files = new Map<string, Buffer>();
let pages: string[] = [];

const draft = `---
title: "Secret plans"
date: 2026-11-07
summary: "Not yet"
tags: [unreleased-news]
cover: null
draft: true
---

The hidden word is zanzibar.
`;

before(() => {
	tmp = mkdtempSync(join(tmpdir(), "lorehallow-output-"));
	const devlog = join(tmp, "devlog");
	cpSync(join(web, "content", "devlog"), devlog, { recursive: true });
	writeFileSync(join(devlog, "secret-plans.md"), draft);
	out = join(tmp, "site");
	execFileSync("npx", ["--no-install", "astro", "build", "--silent"], {
		cwd: web,
		stdio: "inherit",
		env: { ...process.env, LOREHALLOW_ASTRO_OUT: out, LOREHALLOW_DEVLOG_DIR: devlog },
	});
	for (const e of readdirSync(out, { recursive: true, withFileTypes: true })) {
		if (e.isFile()) {
			const p = join(e.parentPath, e.name).slice(out.length + 1);
			files.set(p, readFileSync(join(out, p)));
		}
	}
	pages = [...files.keys()].filter((p) => p.endsWith(".html")).sort();
});

after(() => {
	if (tmp) rmSync(tmp, { recursive: true, force: true });
});

const html = (p: string) => (files.get(p) ?? Buffer.alloc(0)).toString();
const css = () => {
	const p = [...files.keys()].find((f) => /^assets\/site\.[0-9a-f]{10}\.css$/.test(f));
	assert.ok(p, "no stylesheet in the output");
	return html(p);
};
const each = (fn: (page: string, tag: Tag) => void) => {
	for (const p of pages) for (const t of tags(html(p))) fn(p, t);
};

test("no scripts, embeds, inline handlers, style attributes or javascript: URLs", () => {
	const problems: string[] = [];
	each((p, t) => {
		if (["script", "iframe", "object", "embed"].includes(t.name)) problems.push(`${p}: <${t.name}>`);
		for (const [k, v] of t.attrs) {
			if (k.startsWith("on")) problems.push(`${p}: <${t.name} ${k}> is an inline event handler`);
			if (k === "style") problems.push(`${p}: <${t.name} style> is refused by the CSP`);
			if (v.trim().toLowerCase().startsWith("javascript:")) problems.push(`${p}: javascript: URL`);
		}
	});
	for (const [p, data] of files) {
		if (/\.(html|svg|xml)$/.test(p) && /<script/i.test(data.toString()))
			problems.push(`${p} contains a script`);
	}
	assert.ok(
		!files.has("_astro") && ![...files.keys()].some((f) => f.endsWith(".js")),
		"the build wrote JavaScript",
	);
	assert.deepEqual(problems, []);
});

test("every image has alt text and a whole-number multiple of its size", () => {
	const problems: string[] = [];
	each((p, t) => {
		if (t.name !== "img") return;
		const src = t.attrs.get("src") ?? "";
		if (!t.attrs.has("alt")) problems.push(`${p}: <img src=${src}> has no alt`);
		const w = Number(t.attrs.get("width"));
		const h = Number(t.attrs.get("height"));
		const data = files.get(src.replace(/^\//, ""));
		if (!data) return void problems.push(`${p}: <img src=${src}> doesn't exist`);
		let iw = 0;
		let ih = 0;
		if (src.endsWith(".png")) ({ width: iw, height: ih } = decodePNG(data));
		else if (src.endsWith(".svg")) {
			const m = /width="(\d+)" height="(\d+)"/.exec(data.toString());
			iw = Number(m?.[1]);
			ih = Number(m?.[2]);
		}
		if (!iw || w % iw !== 0 || h % ih !== 0 || w / iw !== h / ih) {
			problems.push(`${p}: <img src=${src}> is ${w}×${h}, not a whole-number multiple of ${iw}×${ih}`);
		}
	});
	assert.deepEqual(problems, []);
});

test("every responsive pixel-art rule scales by a whole number", () => {
	const base = new Map<string, [number, number]>();
	const problems: string[] = [];
	for (const m of css().matchAll(/\.(k-[0-9a-f]{8})\{width:(\d+)px;height:(\d+)px\}/g)) {
		const [w, h] = [Number(m[2]), Number(m[3])];
		const b = base.get(m[1] ?? "");
		if (!b) base.set(m[1] ?? "", [w, h]);
		else if (w % b[0] !== 0 || h % b[1] !== 0 || w / b[0] !== h / b[1]) problems.push(`${m[1]}: ${w}×${h}`);
	}
	assert.ok(base.size > 0, "no responsive image rules");
	assert.deepEqual(problems, []);
});

test("every internal link, fragment and CSS url() resolves", () => {
	const exists = (target: string) => {
		let p = target.split("#")[0]?.split("?")[0]?.replace(/^\//, "") ?? "";
		if (p === "" || p.endsWith("/")) p += "index.html";
		return files.has(p);
	};
	const ids = new Map<string, Set<string>>();
	for (const p of pages)
		ids.set(p, new Set(tags(html(p)).flatMap((t) => (t.attrs.has("id") ? [t.attrs.get("id") ?? ""] : []))));
	const problems: string[] = [];
	each((p, t) => {
		for (const key of ["href", "src"]) {
			let v = t.attrs.get(key);
			if (v === undefined) continue;
			if (v.startsWith(`${BaseURL}/`)) v = v.slice(BaseURL.length);
			if (v.startsWith("#")) {
				if (!ids.get(p)?.has(v.slice(1))) problems.push(`${p}: ${key}="${v}" points at a missing id`);
			} else if (v.startsWith("/")) {
				if (!exists(v)) problems.push(`${p}: ${key}="${v}" doesn't exist`);
				const [path, frag] = v.split("#");
				if (frag !== undefined) {
					const target = `${(path ?? "").replace(/^\//, "")}index.html`;
					if (ids.has(target) && !ids.get(target)?.has(frag)) problems.push(`${p}: ${v}: no id "${frag}"`);
				}
			} else if (!/^(https?:\/\/|mailto:)/.test(v)) {
				problems.push(`${p}: ${key}="${v}" is relative; use a root-relative link`);
			}
		}
	});
	for (const m of css().matchAll(/url\("?(\/[^")]+)"?\)/g)) {
		if (!exists(m[1] ?? "")) problems.push(`site.css: url(${m[1]}) doesn't exist`);
	}
	assert.deepEqual(problems, []);
});

test("nothing loads from another origin", () => {
	const problems: string[] = [];
	const foreign = (v: string | undefined) => v !== undefined && (v.includes("://") || v.startsWith("//"));
	each((p, t) => {
		if (["img", "source", "video", "audio"].includes(t.name)) {
			for (const k of ["src", "srcset"]) if (foreign(t.attrs.get(k))) problems.push(`${p}: <${t.name} ${k}>`);
		}
		if (
			t.name === "link" &&
			!["canonical", "alternate"].includes(t.attrs.get("rel") ?? "") &&
			foreign(t.attrs.get("href"))
		) {
			problems.push(`${p}: <link href=${t.attrs.get("href")}>`);
		}
	});
	assert.ok(!/url\("?(https?:)?\/\//.test(css()), "site.css loads from another origin");
	assert.deepEqual(problems, []);
});

test("the landing page stays under 1 MB with everything it can load", () => {
	const seen = new Set<string>();
	let total = html("index.html").length;
	const add = (u: string | undefined) => {
		if (!u?.startsWith("/")) return;
		const p = u.slice(1);
		if (seen.has(p)) return;
		seen.add(p);
		total += files.get(p)?.length ?? 0;
	};
	for (const t of tags(html("index.html"))) {
		if (t.name === "img") add(t.attrs.get("src"));
		if (t.name === "link" && ["stylesheet", "preload", "icon"].includes(t.attrs.get("rel") ?? "")) {
			add(t.attrs.get("href"));
		}
	}
	for (const m of css().matchAll(/url\("?(\/[^")]+)"?\)/g)) add(m[1]);
	assert.ok(total <= 1 << 20, `landing page is ${total >> 10} KB; the budget is 1024 KB`);
});

test("every page has its metadata and links to both repositories", () => {
	const problems: string[] = [];
	for (const p of pages) {
		const found = new Map<string, string>([["title", title(html(p))]]);
		for (const t of tags(html(p))) {
			if (t.name === "link" && t.attrs.get("rel") === "canonical")
				found.set("canonical", t.attrs.get("href") ?? "");
			if (t.name === "meta") {
				const k = t.attrs.get("name") ?? t.attrs.get("property") ?? "";
				found.set(k, t.attrs.get("content") ?? "");
			}
			if (t.name === "html") found.set("lang", t.attrs.get("lang") ?? "");
			if (t.name === "a" && t.attrs.get("href") === RepoURL) found.set("github", RepoURL);
			if (t.name === "a" && t.attrs.get("href") === CodebergURL) found.set("codeberg", CodebergURL);
		}
		for (const k of [
			"title",
			"canonical",
			"description",
			"og:title",
			"og:image",
			"og:url",
			"lang",
			"github",
			"codeberg",
		]) {
			if (!found.get(k)) problems.push(`${p}: missing ${k}`);
		}
		const canonical = found.get("canonical") ?? "";
		if (canonical && !canonical.startsWith(`${BaseURL}/`))
			problems.push(`${p}: canonical ${canonical} isn't absolute`);
		const og = found.get("og:image") ?? "";
		const card = files.get(og.replace(`${BaseURL}/`, ""));
		const size = card ? decodePNG(card) : null;
		if (size?.width !== 1200 || size?.height !== 630) problems.push(`${p}: og:image ${og} isn't 1200×630`);
	}
	assert.deepEqual(problems, []);
});

test("feeds and the sitemap are well-formed, and the progress data is complete", () => {
	for (const f of ["devlog/rss.xml", "devlog/atom.xml", "sitemap.xml"]) {
		const xml = html(f);
		assert.ok(xml.startsWith('<?xml version="1.0" encoding="UTF-8"?>'), `${f}: no XML declaration`);
		// Every tag closes in order.
		const stack: string[] = [];
		for (const m of xml.slice(38).matchAll(/<(\/?)([a-zA-Z:]+)[^>]*?(\/?)>/g)) {
			if (m[3] === "/") continue;
			if (m[1] === "/") assert.equal(stack.pop(), m[2], `${f}: </${m[2]}> closes the wrong element`);
			else stack.push(m[2] ?? "");
		}
		assert.deepEqual(stack, [], `${f}: unclosed elements`);
	}
	const sm = html("sitemap.xml");
	for (const p of pages) {
		if (p === "404.html") assert.ok(!sm.includes("404"), "the sitemap lists the 404 page");
		else
			assert.ok(
				sm.includes(`<loc>${BaseURL}/${p.replace(/index\.html$/, "")}</loc>`),
				`sitemap is missing ${p}`,
			);
	}
	const pj = JSON.parse(html("progress.json")) as { total: number; phases: unknown[] };
	assert.ok(pj.total > 0 && pj.phases.length === 21);
	assert.ok(html("progress.svg").startsWith("<svg"));
	assert.ok(html("robots.txt").includes(`Sitemap: ${BaseURL}/sitemap.xml`));
	for (const f of ["favicon.ico", "apple-touch-icon.png", "404.html"])
		assert.ok(files.get(f)?.length, `${f} missing`);
});

test("no draft reaches any file in the build", () => {
	const leaks = [...files].filter(
		([p, data]) =>
			/secret|unreleased/.test(p) || /Secret plans|zanzibar|unreleased-news/.test(data.toString("latin1")),
	);
	assert.deepEqual(
		leaks.map(([p]) => p),
		[],
	);
});
