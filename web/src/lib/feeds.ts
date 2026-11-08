// RSS 2.0 and Atom feeds for the devlog, and the sitemap, with the same
// entries, ids and dates the Go-built site published before the Astro cutover,
// so feed readers see no duplicates after the cutover.
import type { Post } from "./devlog.ts";

/** Escapes text the way Go's encoding/xml does. */
export function xmlEscape(s: string): string {
	return s.replace(
		/[&<>"'\t\n\r]/g,
		(c) =>
			({
				"&": "&amp;",
				"<": "&lt;",
				">": "&gt;",
				'"': "&#34;",
				"'": "&#39;",
				"\t": "&#x9;",
				"\n": "&#xA;",
				"\r": "&#xD;",
			})[c] ?? c,
	);
}

const header = '<?xml version="1.0" encoding="UTF-8"?>\n';
const days = ["Sun", "Mon", "Tue", "Wed", "Thu", "Fri", "Sat"];
const months = ["Jan", "Feb", "Mar", "Apr", "May", "Jun", "Jul", "Aug", "Sep", "Oct", "Nov", "Dec"];
const p2 = (n: number) => String(n).padStart(2, "0");

/** Go's time.RFC1123Z for a UTC date: "Sat, 07 Nov 2026 00:00:00 +0000". */
export function rfc1123z(d: Date): string {
	return `${days[d.getUTCDay()]}, ${p2(d.getUTCDate())} ${months[d.getUTCMonth()]} ${d.getUTCFullYear()} ${p2(d.getUTCHours())}:${p2(d.getUTCMinutes())}:${p2(d.getUTCSeconds())} +0000`;
}

/** Go's time.RFC3339 for a UTC date: "2026-11-07T00:00:00Z". */
export function rfc3339(d: Date): string {
	return `${d.toISOString().slice(0, 19)}Z`;
}

const el = (indent: string, name: string, text: string) => `${indent}<${name}>${xmlEscape(text)}</${name}>\n`;

export function rssFeed(base: string, posts: Post[]): string {
	let s = `${header}<rss version="2.0" xmlns:atom="http://www.w3.org/2005/Atom">\n  <channel>\n`;
	s += el("    ", "title", "Lorehallow devlog");
	s += el("    ", "link", `${base}/devlog/`);
	s += `    <atom:link href="${xmlEscape(`${base}/devlog/rss.xml`)}" rel="self" type="application/rss+xml"></atom:link>\n`;
	s += el(
		"    ",
		"description",
		"Notes from building Lorehallow, a small pixel-art MMO with a Go server, in the open.",
	);
	s += el("    ", "language", "en");
	const first = posts[0];
	if (first) s += el("    ", "lastBuildDate", rfc1123z(first.date));
	for (const p of posts) {
		s += "    <item>\n";
		s += el("      ", "title", p.title);
		s += el("      ", "link", base + p.url);
		s += `      <guid isPermaLink="true">${xmlEscape(base + p.url)}</guid>\n`;
		s += el("      ", "pubDate", rfc1123z(p.date));
		s += el("      ", "description", p.body);
		for (const t of p.tags) s += el("      ", "category", t);
		s += "    </item>\n";
	}
	return `${s}  </channel>\n</rss>\n`;
}

export function atomFeed(base: string, posts: Post[], built: Date): string {
	const updated = posts[0]?.date ?? built;
	let s = `${header}<feed xmlns="http://www.w3.org/2005/Atom">\n`;
	s += el("  ", "title", "Lorehallow devlog");
	s += el("  ", "id", `${base}/devlog/`);
	s += el("  ", "updated", rfc3339(updated));
	s += `  <link href="${xmlEscape(`${base}/devlog/atom.xml`)}" rel="self" type="application/atom+xml"></link>\n`;
	s += `  <link href="${xmlEscape(`${base}/devlog/`)}" rel="alternate" type="text/html"></link>\n`;
	s += `  <author>\n${el("    ", "name", "Lorehallow")}${el("    ", "uri", `${base}/`)}  </author>\n`;
	for (const p of posts) {
		s += "  <entry>\n";
		s += el("    ", "title", p.title);
		s += el("    ", "id", base + p.url);
		s += `    <link href="${xmlEscape(base + p.url)}" rel="alternate" type="text/html"></link>\n`;
		s += el("    ", "published", rfc3339(p.date));
		s += el("    ", "updated", rfc3339(p.date));
		s += el("    ", "summary", p.summary);
		s += `    <content type="html">${xmlEscape(p.body)}</content>\n`;
		for (const t of p.tags) s += `    <category term="${xmlEscape(t)}"></category>\n`;
		s += "  </entry>\n";
	}
	return `${s}</feed>\n`;
}

export interface SitemapURL {
	loc: string;
	lastmod: string;
}

export function sitemap(base: string, pages: SitemapURL[]): string {
	let s = `${header}<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">\n`;
	for (const p of pages) {
		s += `  <url>\n${el("    ", "loc", base + p.loc)}`;
		if (p.lastmod) s += el("    ", "lastmod", p.lastmod);
		s += "  </url>\n";
	}
	return `${s}</urlset>\n`;
}
