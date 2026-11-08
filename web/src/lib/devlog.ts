// Devlog posts as the pages and feeds use them. The posts themselves come from
// the devlog content collection (src/content.config.ts), which reads
// web/content/devlog/ in place.

/** One devlog post from web/content/devlog/<slug>.md. */
export interface Post {
	slug: string;
	title: string;
	date: Date;
	summary: string;
	tags: string[];
	/** Art path under web/assets/art, or "". */
	cover: string;
	draft: boolean;
	/** The rendered body. */
	body: string;
	url: string;
	words: number;
}

/** A rough reading time. */
export const minutes = (p: Post): number => Math.max(1, Math.floor((p.words + 199) / 200));

/** Words in a Markdown body, counted like Go's strings.Fields. */
export const countWords = (body: string): number => body.split(/\s+/).filter(Boolean).length;

/** Newest first; posts on the same day by slug. */
export function sortPosts(posts: Post[]): Post[] {
	return posts.sort(
		(a, b) => b.date.getTime() - a.date.getTime() || (a.slug < b.slug ? -1 : a.slug > b.slug ? 1 : 0),
	);
}

/** Every tag with its posts, alphabetically. */
export function tagIndex(posts: Post[]): { tags: string[]; byTag: Map<string, Post[]> } {
	const byTag = new Map<string, Post[]>();
	for (const p of posts) {
		for (const t of p.tags) byTag.set(t, [...(byTag.get(t) ?? []), p]);
	}
	return { tags: [...byTag.keys()].sort(), byTag };
}

/** "7 November 2026". */
export function longDate(d: Date): string {
	return `${d.getUTCDate()} ${d.toLocaleString("en-GB", { month: "long", timeZone: "UTC" })} ${d.getUTCFullYear()}`;
}

/** "7 Nov 2026". */
export function shortDate(d: Date): string {
	return `${d.getUTCDate()} ${d.toLocaleString("en-GB", { month: "short", timeZone: "UTC" })} ${d.getUTCFullYear()}`;
}

/** "2026-11-07". */
export const isoDate = (d: Date): string => d.toISOString().slice(0, 10);

/** "7 November 2026" in the build machine's time zone, as the footer has always shown it. */
export function localLongDate(d: Date): string {
	return `${d.getDate()} ${d.toLocaleString("en-GB", { month: "long" })} ${d.getFullYear()}`;
}
