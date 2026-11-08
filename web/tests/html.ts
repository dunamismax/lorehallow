// A small HTML tag scanner for the output tests. The site's pages are
// generated, so a tokenizer over start tags and their attributes is enough:
// no scripts, no comments in the way, attributes always quoted or bare.

export interface Tag {
	name: string;
	attrs: Map<string, string>;
}

const tagRE = /<([a-zA-Z][a-zA-Z0-9-]*)((?:\s+[^\s=/>]+(?:\s*=\s*(?:"[^"]*"|'[^']*'|[^\s"'>]+))?)*)\s*\/?>/g;
const attrRE = /([^\s=/>]+)(?:\s*=\s*(?:"([^"]*)"|'([^']*)'|([^\s"'>]+)))?/g;

function decode(s: string): string {
	return s
		.replace(/&#(\d+);/g, (_, n: string) => String.fromCodePoint(Number(n)))
		.replace(/&#x([0-9a-f]+);/gi, (_, n: string) => String.fromCodePoint(Number.parseInt(n, 16)))
		.replaceAll("&quot;", '"')
		.replaceAll("&lt;", "<")
		.replaceAll("&gt;", ">")
		.replaceAll("&amp;", "&");
}

/** Every start tag in the document, with its attributes (lowercased names, decoded values). */
export function tags(html: string): Tag[] {
	const body = html.replace(/<!--[\s\S]*?-->/g, "");
	const out: Tag[] = [];
	for (const m of body.matchAll(tagRE)) {
		const attrs = new Map<string, string>();
		for (const a of (m[2] ?? "").matchAll(attrRE)) {
			attrs.set((a[1] ?? "").toLowerCase(), decode(a[2] ?? a[3] ?? a[4] ?? ""));
		}
		out.push({ name: (m[1] ?? "").toLowerCase(), attrs });
	}
	return out;
}

/** The text of the first <title>. */
export function title(html: string): string {
	return decode(/<title>([^<]*)<\/title>/.exec(html)?.[1] ?? "");
}
