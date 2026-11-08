// The devlog front matter: all six fields required, nothing else allowed.
// Shared by the content collection (src/content.config.ts) and its tests.
import { existsSync } from "node:fs";
import { join } from "node:path";
import { z } from "astro/zod";

export const slugRE = /^[a-z0-9]+(?:-[a-z0-9]+)*$/;

/** The schema, checking cover art against artDir (web/assets/art). */
export function postSchema(artDir: string) {
	return z.strictObject({
		title: z.string().min(1, "title is empty"),
		date: z.date({ error: "date must be YYYY-MM-DD" }),
		summary: z.string().min(1, "summary is empty"),
		tags: z.array(z.string().regex(slugRE, "tags must be lowercase words joined by hyphens")),
		cover: z
			.string()
			.nullable()
			.refine((c) => c === null || existsSync(join(artDir, c)), "cover isn't in web/assets/art"),
		draft: z.boolean(),
	});
}
