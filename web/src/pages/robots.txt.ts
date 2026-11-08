import type { APIRoute } from "astro";
import { BaseURL } from "../lib/state.ts";

export const GET: APIRoute = () =>
	new Response(`User-agent: *\nAllow: /\n\nSitemap: ${BaseURL}/sitemap.xml\n`);
