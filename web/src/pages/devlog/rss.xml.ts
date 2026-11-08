import type { APIRoute } from "astro";
import { rssFeed } from "../../lib/feeds.ts";
import { site } from "../../lib/site.ts";
import { BaseURL } from "../../lib/state.ts";

export const GET: APIRoute = async () => {
	const { posts } = await site();
	return new Response(rssFeed(BaseURL, posts), { headers: { "Content-Type": "application/rss+xml" } });
};
