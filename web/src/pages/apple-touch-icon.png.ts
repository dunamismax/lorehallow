import type { APIRoute } from "astro";
import { state } from "../lib/state.ts";

export const GET: APIRoute = () =>
	new Response(new Uint8Array(state().touchIcon), { headers: { "Content-Type": "image/png" } });
