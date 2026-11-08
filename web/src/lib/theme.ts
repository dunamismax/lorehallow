// The palette from art/style/palette.hex and the color roles the stylesheet
// uses.

/** One palette entry. */
export interface Color {
	/** CSS token name, e.g. "lantern-gold". */
	name: string;
	/** "#f2c45a". */
	hex: string;
}

/**
 * Reads palette.hex: one hex color per line, optionally followed by
 * "; name: description" or "; name". Every color gets a CSS custom property
 * named after it (--c-<name>).
 */
export function parsePalette(src: string): Color[] {
	const out: Color[] = [];
	const seen = new Set<string>();
	src.split(/\r?\n/).forEach((raw, i) => {
		const n = i + 1;
		const line = raw.trim();
		if (line === "" || line.startsWith(";")) return;
		const cut = line.indexOf(";");
		const hex = (cut < 0 ? line : line.slice(0, cut)).trim().toLowerCase();
		const comment = cut < 0 ? "" : line.slice(cut + 1);
		if (hex.length !== 6) throw new Error(`palette line ${n}: "${hex}" is not a 6-digit hex color`);
		if (!/^[0-9a-f]{6}$/.test(hex)) throw new Error(`palette line ${n}: "${hex}" is not hex`);
		const name = (comment.split(":")[0] ?? "").toLowerCase().trim().split(/\s+/).filter(Boolean).join("-");
		if (name === "") throw new Error(`palette line ${n}: color ${hex} has no name`);
		if (seen.has(name)) throw new Error(`palette line ${n}: duplicate name "${name}"`);
		seen.add(name);
		out.push({ name, hex: `#${hex}` });
	});
	if (out.length === 0) throw new Error("palette is empty");
	return out;
}

/** A semantic color the stylesheet uses for text or backgrounds: --<name> is var(--c-<token>). */
export interface Role {
	name: string;
	token: string;
}

/**
 * Roles of the one (dark, lantern-lit) theme. The stylesheet may only set
 * `color` and `background-color` through roles, so the contrast table below
 * covers every pair it can draw.
 */
export const ROLES: Role[] = [
	["bg", "ink"],
	["bg-raised", "night-plum"],
	["bg-dusk", "deep-dusk"],
	["bg-sky", "dusk-blue"],
	["bg-page", "parchment-white"],
	["bg-button", "lantern-gold"],
	["bg-button-hover", "glow"],
	["bg-done", "sunlit-leaf"],
	["bg-human", "banner-red"],
	["bg-adr", "arcane-violet"],
	["bg-code", "sand"],
	["text", "parchment-white"],
	["text-muted", "pale-stone"],
	["text-heading", "lantern-gold"],
	["text-link", "glow"],
	["text-accent", "amber"],
	["text-done", "sunlit-leaf"],
	["text-on-light", "ink"],
	["text-heading-on-light", "bark"],
	["text-link-on-light", "banner-red"],
	["text-muted-on-light", "dusk-violet-grey"],
].map(([name, token]) => ({ name: name as string, token: token as string }));

/** A text role drawn on a background role. Large text (24px+, or 19px+ bold) needs 3:1, not 4.5:1. */
export interface Pair {
	fg: string;
	bg: string;
	large: boolean;
}

/** Every text-on-background combination the stylesheet uses. The tests check each meets WCAG AA. */
export const PAIRS: Pair[] = [
	["text", "bg"],
	["text", "bg-raised"],
	["text", "bg-dusk"],
	["text", "bg-sky"],
	["text-muted", "bg"],
	["text-muted", "bg-raised"],
	["text-muted", "bg-dusk"],
	["text-heading", "bg"],
	["text-heading", "bg-raised"],
	["text-heading", "bg-dusk"],
	["text-heading", "bg-sky"],
	["text-link", "bg"],
	["text-link", "bg-raised"],
	["text-link", "bg-dusk"],
	["text-link", "bg-sky"],
	["text-accent", "bg"],
	["text-accent", "bg-raised"],
	["text-accent", "bg-dusk"],
	["text-done", "bg"],
	["text-done", "bg-raised"],
	["text-on-light", "bg-page"],
	["text-heading-on-light", "bg-page"],
	["text-link-on-light", "bg-page"],
	["text-muted-on-light", "bg-page"],
	["text-on-light", "bg-button"],
	["text-on-light", "bg-button-hover"],
	["text-on-light", "bg-done"],
	["text-on-light", "bg-code"],
	["text", "bg-human"],
	["text", "bg-adr"],
].map(([fg, bg]) => ({ fg: fg as string, bg: bg as string, large: false }));

function luminance(hex: string): number {
	const v = Number.parseInt(hex.replace("#", ""), 16);
	const ch = (c: number) => {
		const s = c / 255;
		return s <= 0.04045 ? s / 12.92 : ((s + 0.055) / 1.055) ** 2.4;
	};
	return 0.2126 * ch((v >> 16) & 0xff) + 0.7152 * ch((v >> 8) & 0xff) + 0.0722 * ch(v & 0xff);
}

/** The WCAG 2 contrast ratio of two "#rrggbb" colors. */
export function contrast(a: string, b: string): number {
	let la = luminance(a);
	let lb = luminance(b);
	if (la < lb) [la, lb] = [lb, la];
	return (la + 0.05) / (lb + 0.05);
}

/** Roles resolved to palette colors. */
export class Theme {
	readonly palette: Color[];
	readonly #byName = new Map<string, string>();

	/** Checks that every role names a palette color. */
	constructor(palette: Color[]) {
		this.palette = palette;
		for (const c of palette) this.#byName.set(c.name, c.hex);
		for (const r of ROLES) {
			if (!this.#byName.has(r.token)) {
				throw new Error(`role --${r.name} uses "${r.token}", which isn't in the palette`);
			}
		}
	}

	/** A palette color by name ("" if unknown). */
	hex(name: string): string {
		return this.#byName.get(name) ?? "";
	}

	/** A role's color. */
	roleHex(role: string): string {
		const r = ROLES.find((x) => x.name === role);
		return r ? this.hex(r.token) : "";
	}

	/** The palette tokens and roles as custom properties. */
	css(): string {
		let s = ":root {\n\t/* Palette: generated from art/style/palette.hex. */\n";
		for (const c of this.palette) s += `\t--c-${c.name}: ${c.hex};\n`;
		s += "\t/* Roles: the only colors text and backgrounds use (src/lib/theme.ts). */\n";
		for (const r of ROLES) s += `\t--${r.name}: var(--c-${r.token});\n`;
		return `${s}}\n`;
	}
}

/** "#rrggbb" as [r, g, b]. */
export function rgb(hex: string): number[] {
	const v = Number.parseInt(hex.replace("#", ""), 16);
	return [(v >> 16) & 0xff, (v >> 8) & 0xff, v & 0xff];
}
