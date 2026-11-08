// Markdown the way every page renders it: GitHub Flavored Markdown, smart
// punctuation and heading ids (Sätteri, Astro's own Markdown processor), with
// raw HTML dropped so no post or BUILD.md line can carry a script.
//
// The devlog collection uses the same plugin through astro.config.ts; this
// module renders the one-line Markdown in BUILD.md items and phase metadata.
import { createSatteriMarkdownProcessor } from "@astrojs/markdown-satteri";
import { defineMdastPlugin } from "satteri";

/** Removes raw HTML (blocks and inline tags) before the Markdown becomes HTML. */
export const dropRawHTML = defineMdastPlugin({
	name: "lorehallow-drop-raw-html",
	html(node, ctx) {
		ctx.removeNode(node);
	},
});

export const RepoURL = "https://github.com/dunamismax/lorehallow";

type Renderer = Awaited<ReturnType<typeof createSatteriMarkdownProcessor>>;
let renderer: Promise<Renderer> | null = null;

function processor(): Promise<Renderer> {
	renderer ??= createSatteriMarkdownProcessor({ mdastPlugins: [dropRawHTML] });
	return renderer;
}

/** Renders Markdown to HTML. */
export async function render(src: string): Promise<string> {
	const out = await (await processor()).render(src);
	return out.code;
}

/** Points relative links (BUILD.md's links to files in the repository) at GitHub. */
export function repoLinks(html: string): string {
	return html.replace(/href="([^"#/:][^":]*)"/g, `href="${RepoURL}/blob/main/$1"`);
}

/** Renders one line of Markdown without the surrounding <p>, with repository links. */
export async function inline(src: string): Promise<string> {
	let h = (await render(src)).trim();
	if (h.startsWith("<p>")) h = h.slice(3);
	if (h.endsWith("</p>")) h = h.slice(0, -4);
	return repoLinks(h);
}
