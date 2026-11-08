// /favicon.ico (16 and 32 px, stored as PNG), for clients that ask for it without reading the page.
import type { APIRoute } from "astro";
import { state } from "../lib/state.ts";

export const GET: APIRoute = () =>
	new Response(new Uint8Array(state().favicon), { headers: { "Content-Type": "image/x-icon" } });
