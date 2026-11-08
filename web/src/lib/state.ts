// What the whole build shares: the palette and theme, the compiled fonts, the
// icons and the asset registry. Pages create it the first time they need it;
// the lorehallow integration reads it to write the assets and the stylesheet.
//
// It lives on globalThis so every copy of this module (Astro bundles the
// pages separately from astro.config.ts) sees the same build.
import { execFileSync } from "node:child_process";
import { readFileSync } from "node:fs";
import { join } from "node:path";
import { Assets, repoRoot } from "./assets.ts";
import { PixelFont } from "./pixelfont.ts";
import { decodePNG, type RGBA } from "./png.ts";
import { parsePalette, Theme } from "./theme.ts";
import { fromSFNT } from "./woff.ts";

export const BaseURL = "https://lorehallow.com";
export const RepoURL = "https://github.com/dunamismax/lorehallow";
export const CodebergURL = "https://codeberg.org/dunamismax/lorehallow";
export const BuildMDURL = `${RepoURL}/blob/main/BUILD.md`;

/** The URL pages link to for the stylesheet; replaced with the real one once every page is rendered. */
export const CSS_TOKEN = "/assets/__site__.css";

export interface State {
	root: string;
	web: string;
	built: Date;
	commit: string;
	assets: Assets;
	theme: Theme;
	font: PixelFont;
	fonts: Record<"pixel" | "regular" | "italic" | "bold" | "mono", string>;
	icons: Record<"16" | "32" | "180" | "192" | "512", string>;
	/** /favicon.ico and /apple-touch-icon.png, for clients that ask without reading the page. */
	favicon: Uint8Array;
	touchIcon: Uint8Array;
	ogBase: RGBA;
	/** Share cards already drawn, by title and kicker. */
	og: Map<string, string>;
}

const KEY = Symbol.for("lorehallow.web.state");
type Global = typeof globalThis & { [KEY]?: State };

/** The build's shared state, created on first use. */
export function state(): State {
	const g = globalThis as Global;
	g[KEY] ??= create();
	return g[KEY];
}

/** The state if a page has created it, for the integration. */
export function existingState(): State | undefined {
	return (globalThis as Global)[KEY];
}

/** Forgets the state, so the next page starts a fresh build. */
export function resetState(): void {
	delete (globalThis as Global)[KEY];
}

function create(): State {
	const root = repoRoot();
	const web = join(root, "web");
	const assets = new Assets(join(web, "assets", "art"));
	const theme = new Theme(parsePalette(readFileSync(join(root, "art", "style", "palette.hex"), "utf8")));
	const font = PixelFont.parse(readFileSync(join(web, "static", "fonts", "lorehallow-pixel.txt"), "utf8"));
	const ttf = (name: string) => readFileSync(join(web, "static", "fonts", `${name}.ttf`));
	const fonts = {
		pixel: assets.add("lorehallow-pixel.woff", fromSFNT(font.ttf())),
		regular: assets.add("go-regular.woff", fromSFNT(ttf("Go-Regular"))),
		italic: assets.add("go-italic.woff", fromSFNT(ttf("Go-Italic"))),
		bold: assets.add("go-bold.woff", fromSFNT(ttf("Go-Bold"))),
		mono: assets.add("go-mono.woff", fromSFNT(ttf("Go-Mono"))),
	};
	const icon = (n: number) => assets.image(`icons/favicon_${n}.png`).url;
	const icons = { "16": icon(16), "32": icon(32), "180": icon(180), "192": icon(192), "512": icon(512) };
	const iconFile = (n: number) => readFileSync(join(assets.artDir, "icons", `favicon_${n}.png`));
	let commit = process.env.LOREHALLOW_COMMIT ?? "";
	if (commit === "") {
		try {
			commit = execFileSync("git", ["-C", root, "rev-parse", "HEAD"], { encoding: "utf8" }).trim();
		} catch {
			commit = "unknown";
		}
	}
	return {
		root,
		web,
		built: new Date(),
		commit,
		assets,
		theme,
		font,
		fonts,
		icons,
		favicon: ico([16, 32], [iconFile(16), iconFile(32)]),
		touchIcon: iconFile(180),
		ogBase: decodePNG(readFileSync(join(assets.artDir, "web", "og_card.png")), "og_card"),
		og: new Map(),
	};
}

/** Packs PNG images into an .ico file. */
export function ico(sizes: number[], pngs: Uint8Array[]): Uint8Array {
	const head = 6 + 16 * pngs.length;
	const total = head + pngs.reduce((n, p) => n + p.length, 0);
	const out = new Uint8Array(total);
	const v = new DataView(out.buffer);
	v.setUint16(0, 0, true);
	v.setUint16(2, 1, true);
	v.setUint16(4, pngs.length, true);
	let offset = head;
	pngs.forEach((data, i) => {
		const e = 6 + 16 * i;
		const s = sizes[i] ?? 0;
		out[e] = s;
		out[e + 1] = s;
		v.setUint16(e + 4, 1, true);
		v.setUint16(e + 6, 32, true);
		v.setUint32(e + 8, data.length, true);
		v.setUint32(e + 12, offset, true);
		out.set(data, offset);
		offset += data.length;
	});
	return out;
}
