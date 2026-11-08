// /progress.svg: the progress badge, drawn in the pixel font.
import type { APIRoute } from "astro";
import { badgeSVG } from "../lib/progress.ts";
import { site } from "../lib/site.ts";

export const GET: APIRoute = async () => {
	const { st, progress } = await site();
	return new Response(badgeSVG(progress, st.font, st.theme), {
		headers: { "Content-Type": "image/svg+xml" },
	});
};
