// /sitemap.xml: every page except the 404, in the order the Go generator lists them.
import type { APIRoute } from "astro";
import { isoDate } from "../lib/devlog.ts";
import { type SitemapURL, sitemap } from "../lib/feeds.ts";
import { site } from "../lib/site.ts";
import { BaseURL } from "../lib/state.ts";

export const GET: APIRoute = async () => {
	const { progress, posts, tags } = await site();
	const page = (loc: string, lastmod = ""): SitemapURL => ({ loc, lastmod });
	const pages = [
		page("/"),
		page("/world/"),
		page("/progress/"),
		...progress.phases.map((p) => page(p.url)),
		page("/devlog/"),
		...tags.map((t) => page(`/devlog/tags/${t}/`)),
		...posts.map((p) => page(p.url, isoDate(p.date))),
		page("/play/"),
		page("/about/"),
		page("/faq/"),
	];
	return new Response(sitemap(BaseURL, pages), { headers: { "Content-Type": "application/xml" } });
};
