// The devlog collection: web/content/devlog/*.md read in place, with the same
// front matter rules the Go-built site had. A bad field fails the build with the
// file and field named. LOREHALLOW_DEVLOG_DIR points it elsewhere (the output
// tests add a draft to a copy).

import { defineCollection } from "astro:content";
import { join } from "node:path";
import { glob } from "astro/loaders";
import { postSchema, slugRE } from "./lib/devlog-schema.ts";

const devlog = defineCollection({
	loader: glob({
		base: process.env.LOREHALLOW_DEVLOG_DIR ?? "./content/devlog",
		pattern: ["*.md", "!README.md"],
		generateId: ({ entry }) => {
			const slug = entry.replace(/\.md$/, "");
			if (!slugRE.test(slug)) {
				throw new Error(`devlog ${entry}: file name "${slug}" must be lowercase words joined by hyphens`);
			}
			return slug;
		},
	}),
	schema: postSchema(join(process.cwd(), "assets", "art")),
});

export const collections = { devlog };
