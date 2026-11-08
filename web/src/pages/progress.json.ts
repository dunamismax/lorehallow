// /progress.json: the progress numbers for scripts, the README badge and the uptime check.
import type { APIRoute } from "astro";
import { progressJSON } from "../lib/progress.ts";
import { site } from "../lib/site.ts";
import { BaseURL } from "../lib/state.ts";

export const GET: APIRoute = async () => {
	const { st, progress } = await site();
	return new Response(progressJSON(progress, BaseURL, st.built, st.commit));
};
