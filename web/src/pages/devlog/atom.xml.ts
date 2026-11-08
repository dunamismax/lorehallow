import type { APIRoute } from "astro";
import { atomFeed } from "../../lib/feeds.ts";
import { site } from "../../lib/site.ts";
import { BaseURL } from "../../lib/state.ts";

export const GET: APIRoute = async () => {
	const { st, posts } = await site();
	return new Response(atomFeed(BaseURL, posts, st.built), {
		headers: { "Content-Type": "application/atom+xml" },
	});
};
