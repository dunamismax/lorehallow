// The devlog collection: web/content/devlog/*.md read in place, with the same
// front matter rules as the Go generator. A bad field fails the build with the
// file and field named.

import { defineCollection } from "astro:content";
import { existsSync } from "node:fs";
import { join } from "node:path";
import { glob } from "astro/loaders";
import { z } from "astro/zod";

const slugRE = /^[a-z0-9]+(?:-[a-z0-9]+)*$/;
const artDir = join(process.cwd(), "assets", "art");

const devlog = defineCollection({
	loader: glob({
		base: "./content/devlog",
		pattern: ["*.md", "!README.md"],
		generateId: ({ entry }) => {
			const slug = entry.replace(/\.md$/, "");
			if (!slugRE.test(slug)) {
				throw new Error(`devlog ${entry}: file name "${slug}" must be lowercase words joined by hyphens`);
			}
			return slug;
		},
	}),
	schema: z.strictObject({
		title: z.string().min(1, "title is empty"),
		date: z.date({ error: "date must be YYYY-MM-DD" }),
		summary: z.string().min(1, "summary is empty"),
		tags: z.array(z.string().regex(slugRE, "tags must be lowercase words joined by hyphens")),
		cover: z
			.string()
			.nullable()
			.refine((c) => c === null || existsSync(join(artDir, c)), "cover isn't in web/assets/art"),
		draft: z.boolean(),
	}),
});

export const collections = { devlog };
