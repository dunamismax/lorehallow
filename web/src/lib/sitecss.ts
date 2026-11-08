// The stylesheet: web/static/css/site.css with the palette tokens and color
// roles first, font and art URLs filled in, the generated quest-map, firefly
// and progress-bar rules, and the responsive pixel-art rules last. Built
// after every page, because the last part depends on the images they drew.
//
// Placeholders in site.css: {{font NAME}}, {{art PATH}}, {{fireflies}},
// {{percents}} and {{questmap}}. Any other {{...}} fails the build.
import { readFileSync } from "node:fs";
import { join } from "node:path";
import type { State } from "./state.ts";

/** Fireflies, placed by hand in art pixels around the hero's center: x, height above the bottom, drift and blink seconds, and their delays. */
export const FIREFLIES: number[][] = [
	[-210, 34, 9, 3, 1, 0],
	[-172, 58, 11, 4, 4, 1],
	[-140, 22, 8, 3, 2, 2],
	[-104, 70, 12, 5, 7, 3],
	[-76, 40, 10, 3, 3, 1],
	[-38, 64, 9, 4, 6, 2],
	[-12, 30, 13, 3, 9, 0],
	[22, 52, 10, 5, 2, 4],
	[58, 26, 11, 3, 5, 1],
	[92, 74, 9, 4, 8, 2],
	[126, 44, 12, 3, 1, 0],
	[164, 30, 10, 4, 6, 3],
	[196, 62, 8, 3, 3, 1],
	[232, 36, 11, 5, 4, 2],
];

/** Lays the landmarks out along a winding road: one column on phones, then rows that turn back on themselves (4 and then 7 across). */
export function questMapCSS(n: number): string {
	let s = "";
	for (const { minW, cols } of [
		{ minW: 640, cols: 4 },
		{ minW: 1024, cols: 7 },
	]) {
		s += `@media (min-width:${minW}px){\n.quest-map ol{grid-template-columns:repeat(${cols},1fr)}\n`;
		for (let i = 0; i < n; i++) {
			const row = Math.floor(i / cols);
			const pos = i % cols;
			const col = row % 2 === 1 ? cols - 1 - pos : pos;
			s += `.quest-map .qm-${i}{grid-row:${row + 1};grid-column:${col + 1}}`;
			if (i === n - 1) {
				s += `.quest-map .qm-${i}::before{display:none}`;
			} else if (Math.floor((i + 1) / cols) !== row) {
				s += `.quest-map .qm-${i}::before{left:calc(50% - var(--road)/2);top:var(--road-y);width:var(--road);height:calc(100% + var(--gap))}`;
			} else if (row % 2 === 0) {
				s += `.quest-map .qm-${i}::before{left:50%;top:calc(var(--road-y) - var(--road)/2);width:calc(100% + var(--gap));height:var(--road)}`;
			} else {
				s += `.quest-map .qm-${i}::before{right:50%;left:auto;top:calc(var(--road-y) - var(--road)/2);width:calc(100% + var(--gap));height:var(--road)}`;
			}
			s += "\n";
		}
		s += "}\n";
	}
	return s;
}

/** Renders the stylesheet for a finished build. */
export function siteCSS(st: State, phases: number): string {
	let src = readFileSync(join(st.web, "static", "css", "site.css"), "utf8");
	// The header comment documents the placeholders; drop it so they aren't filled in there.
	src = src.replace(/^\/\*[\s\S]*?\*\/\n/, "");
	src = src.replaceAll("{{fireflies}}", () =>
		FIREFLIES.map(
			(p, i) =>
				`.ff-${i} { left: calc(50% + var(--k) * ${p[0]}px); bottom: calc(var(--k) * ${p[1]}px); animation-duration: ${p[2]}s, ${p[3]}s; animation-delay: -${p[4]}s, -${p[5]}s; }\n`,
		).join(""),
	);
	src = src.replaceAll("{{percents}}", () =>
		Array.from({ length: 101 }, (_, i) => `.xp-${i}{width:${i}%}`).join(""),
	);
	src = src.replaceAll("{{questmap}}", () => questMapCSS(phases));
	src = src.replace(/\{\{font ([a-z]+)\}\}/g, (_, k: string) => {
		const url = st.fonts[k as keyof State["fonts"]];
		if (!url) throw new Error(`site.css: unknown font "${k}"`);
		return url;
	});
	src = src.replace(/\{\{art ([^}\s]+)\}\}/g, (_, key: string) => st.assets.image(key).url);
	const left = /\{\{[\s\S]*?\}\}/.exec(src);
	if (left) throw new Error(`site.css: unknown placeholder ${left[0]}`);
	return `/* lorehallow.com. Built from web/static/css/site.css by src/lib/sitecss.ts. */\n${st.theme.css()}${src}${st.assets.fitCSS()}`;
}
