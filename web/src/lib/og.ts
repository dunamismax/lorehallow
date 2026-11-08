// Share cards: the 240×126 og_card art at exactly 5x (1200×630), with the page
// title drawn on a plaque in the hand-drawn pixel font. Everything is placed on
// the art's own pixel grid, so the card stays true pixel art.
// A port of web/internal/site/og.go.
import { CAP_ROWS, type PixelFont, ROWS } from "./pixelfont.ts";
import { fillRect, type RGBA, upscale } from "./png.ts";
import { rgb, type Theme } from "./theme.ts";

export const OG_SCALE = 5;

export function ogImage(base: RGBA, f: PixelFont, t: Theme, title: string, kicker: string): RGBA {
	const art: RGBA = { width: base.width, height: base.height, data: new Uint8Array(base.data) };
	const ink = rgb(t.hex("ink"));
	const plum = rgb(t.hex("night-plum"));
	const gold = rgb(t.hex("lantern-gold"));
	const paper = rgb(t.hex("parchment-white"));
	const edge = rgb(t.hex("dusk-violet-grey"));

	// Wrap the title to fit the plaque (at most two lines).
	const lines = wrapPixel(f, title, 132).slice(0, 2);
	let w = f.measure(kicker);
	for (const l of lines) w = Math.max(w, f.measure(l));
	const x0 = 8;
	const y0 = 8;
	const pad = 5;
	const h = pad + CAP_ROWS + 4 + lines.length * (ROWS + 2) + pad - 2;
	const px0 = x0;
	const py0 = y0;
	const px1 = x0 + w + 2 * pad;
	const py1 = y0 + h;
	fillRect(art, px0 - 1, py0 - 1, px1 + 1, py1 + 1, ink);
	fillRect(art, px0, py0, px1, py1, plum);
	fillRect(art, px0, py1 - 1, px1, py1, edge);
	fillRect(art, px0 + 1, py1, px1 + 1, py1 + 1, ink); // drop shadow
	fillRect(art, px1 + 1, py0 + 1, px1 + 2, py1 + 1, ink);

	let y = y0 + pad;
	f.draw(art, kicker, x0 + pad, y, 1, gold);
	y += CAP_ROWS + 4;
	for (const l of lines) {
		f.draw(art, l, x0 + pad, y, 1, paper);
		y += ROWS + 2;
	}
	return upscale(art, OG_SCALE);
}

/** Breaks text into lines no wider than maxW pixels, dropping characters the font doesn't have. */
export function wrapPixel(f: PixelFont, s: string, maxW: number): string[] {
	const clean = [...s].filter((c) => f.glyphs.has(c.codePointAt(0) ?? 0));
	const lines: string[] = [];
	let line = "";
	let word = "";
	const flushWord = () => {
		if (word === "") return;
		const attempt = line === "" ? word : `${line} ${word}`;
		if (f.measure(attempt) > maxW && line !== "") {
			lines.push(line);
			line = word;
		} else {
			line = attempt;
		}
		word = "";
	};
	for (const c of clean) {
		if (c === " ") flushWord();
		else word += c;
	}
	flushWord();
	if (line !== "") lines.push(line);
	return lines;
}
