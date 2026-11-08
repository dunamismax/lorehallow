// The site's data for pages and endpoints: BUILD.md progress, the devlog, and
// share cards. Only Astro pages import this (it reads the content collection).

import { getCollection } from "astro:content";
import { readFileSync } from "node:fs";
import { join } from "node:path";
import { parse } from "./buildmd.ts";
import { countWords, type Post, sortPosts, tagIndex } from "./devlog.ts";
import { ogImage } from "./og.ts";
import { encodePNG } from "./png.ts";
import { buildProgress, type Progress } from "./progress.ts";
import { type State, state } from "./state.ts";

/** Drafts are shown only in the local preview (`make web-dev`). */
export const DRAFTS = import.meta.env.DEV;

export interface SiteData {
	st: State;
	progress: Progress;
	posts: Post[];
	tags: string[];
	byTag: Map<string, Post[]>;
}

let cache: { key: string; data: Promise<SiteData> } | null = null;

/** The site's data, built once per build (and again in dev when BUILD.md changes). */
export function site(): Promise<SiteData> {
	const st = state();
	const buildmd = readFileSync(join(st.root, "BUILD.md"), "utf8");
	const key = DRAFTS ? buildmd : "build";
	if (cache?.key !== key) cache = { key, data: load(st, buildmd) };
	return cache.data;
}

async function load(st: State, buildmd: string): Promise<SiteData> {
	const plan = parse(buildmd);
	if (plan.problems.length > 0) {
		throw new Error(`BUILD.md: ${plan.problems.length} format problems; run make buildmd-lint`);
	}
	const landmarks = st.assets.frames("web/progress_landmarks.png");
	const progress = await buildProgress(st.root, plan, landmarks, true);
	const entries = await getCollection("devlog");
	const posts = sortPosts(
		entries
			.filter((e) => DRAFTS || !e.data.draft)
			.map((e) => ({
				slug: e.id,
				title: e.data.title,
				date: e.data.date,
				summary: e.data.summary,
				tags: e.data.tags,
				cover: e.data.cover ?? "",
				draft: e.data.draft,
				body: e.rendered?.html ?? "",
				url: `/devlog/${e.id}/`,
				words: countWords(e.body ?? ""),
			})),
	);
	const { tags, byTag } = tagIndex(posts);
	return { st, progress, posts, tags, byTag };
}

/** The share card for a page title, drawn once and published under /assets/. */
export function ogCard(title: string, kicker: string): string {
	const st = state();
	const key = `${title}|${kicker}`;
	const known = st.og.get(key);
	if (known) return known;
	const data = encodePNG(ogImage(st.ogBase, st.font, st.theme, title, kicker));
	const slug = title
		.toLowerCase()
		.replace(/[^a-z0-9]+/g, "-")
		.replace(/^-+|-+$/g, "");
	const url = st.assets.add(`og-${slug.slice(0, 40)}.png`, data);
	st.og.set(key, url);
	return url;
}
