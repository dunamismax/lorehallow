// Every fingerprinted file under /assets/, and the pixel image rules the pages
// ask for. A port of web/internal/site/assets.go.
//
// Pages register what they use while Astro renders them; the lorehallow
// integration (src/integration.ts) then writes the files and the stylesheet,
// which has to come last because it carries the responsive rules for every
// image the pages drew. The registry lives on globalThis so the page modules
// Astro bundles and the integration share one copy.
import { createHash } from "node:crypto";
import { existsSync, readFileSync } from "node:fs";
import { dirname, join } from "node:path";
import { crop, decodePNG, encodePNG, mirrorPair, type RGBA } from "./png.ts";

/** The repository root: the nearest directory above the working directory with BUILD.md in it. */
export function repoRoot(): string {
	let dir = process.cwd();
	for (;;) {
		if (existsSync(join(dir, "BUILD.md")) && existsSync(join(dir, "web"))) return dir;
		const up = dirname(dir);
		if (up === dir) throw new Error("can't find the repository root (BUILD.md) above the working directory");
		dir = up;
	}
}

/** One image: its pixels and its published URL. */
export interface Art {
	key: string;
	w: number;
	h: number;
	url: string;
	img: RGBA;
}

interface FitRule {
	cls: string;
	w: number;
	h: number;
	/** The widest it may be drawn, in CSS pixels. */
	max: number;
	minK: number;
	gutter: number;
}

/** How a pixel image is drawn. */
export interface ImgOpts {
	alt: string;
	/** Fixed whole-number scale; 0 means responsive (max). */
	k?: number;
	/** Responsive: the widest it may be drawn, in CSS pixels (default 1200). */
	max?: number;
	/** Responsive: the smallest scale (default 1). */
	min?: number;
	/** Above the fold: no lazy loading. */
	eager?: boolean;
	/** Extra classes. */
	class?: string;
}

const sha256hex = (data: Uint8Array | string) => createHash("sha256").update(data).digest("hex");

const escapeAttr = (s: string) =>
	s.replace(
		/[&<>"']/g,
		(c) => ({ "&": "&amp;", "<": "&lt;", ">": "&gt;", '"': "&#34;", "'": "&#39;" })[c] ?? c,
	);

export class Assets {
	readonly artDir: string;
	/** Published files by path (no leading slash), e.g. "assets/site.0123456789.css". */
	readonly files = new Map<string, Uint8Array>();
	readonly #images = new Map<string, Art>();
	readonly #fits = new Map<string, FitRule>();

	constructor(artDir: string) {
		this.artDir = artDir;
	}

	/** Publishes data under /assets/ with a content hash in its name and returns the URL. */
	add(name: string, data: Uint8Array): string {
		const dot = name.lastIndexOf(".");
		const ext = dot < 0 ? "" : name.slice(dot);
		const base = (dot < 0 ? name : name.slice(0, dot)).split("/").at(-1) ?? name;
		const p = `assets/${base}.${sha256hex(data).slice(0, 10)}${ext}`;
		this.files.set(p, data);
		return `/${p}`;
	}

	/**
	 * Loads an art file (relative to web/assets/art), publishes it and returns
	 * it. "sheet.png#frame" crops one named frame from a sheet using its sidecar
	 * .json; "file.png#mirror" is the image followed by its mirror image, which
	 * tiles seamlessly in a row.
	 */
	image(key: string): Art {
		const cached = this.#images.get(key);
		if (cached) return cached;
		const hash = key.indexOf("#");
		const file = hash < 0 ? key : key.slice(0, hash);
		const frag = hash < 0 ? "" : key.slice(hash + 1);
		const bytes = this.#read(file);
		let img = decodePNG(bytes, `art ${file}`);
		let name = (file.split("/").at(-1) ?? file).replace(/\.png$/, "");
		if (frag === "mirror") {
			img = mirrorPair(img);
			name += "-pair";
		} else if (frag !== "") {
			const f = this.#frame(file, frag);
			img = crop(img, f.x, f.y, f.w, f.h);
			name += `-${frag.replaceAll("_", "-")}`;
		}
		const data = frag === "" ? bytes : encodePNG(img);
		const art: Art = { key, w: img.width, h: img.height, url: this.add(`${name}.png`, data), img };
		this.#images.set(key, art);
		return art;
	}

	#read(file: string): Uint8Array {
		try {
			return readFileSync(join(this.artDir, file));
		} catch (err) {
			throw new Error(`art ${file}: ${(err as Error).message}`);
		}
	}

	#sidecar(file: string): { name: string; x: number; y: number; w: number; h: number }[] {
		const path = join(this.artDir, file.replace(/\.png$/, ".json"));
		try {
			const side = JSON.parse(readFileSync(path, "utf8")) as {
				frames?: { name: string; x: number; y: number; w: number; h: number }[];
			};
			return side.frames ?? [];
		} catch (err) {
			throw new Error(`sheet ${file}: ${(err as Error).message}`);
		}
	}

	#frame(file: string, name: string) {
		const f = this.#sidecar(file).find((x) => x.name === name);
		if (!f) throw new Error(`sheet ${file} has no frame "${name}"`);
		return f;
	}

	/** A sheet's frame names in order. */
	frames(file: string): string[] {
		return this.#sidecar(file).map((f) => f.name);
	}

	/**
	 * An <img> for pixel art: explicit width and height, always a whole-number
	 * multiple of the art, and pixelated scaling. Responsive images get a
	 * generated class that steps the scale up with the viewport.
	 */
	img(key: string, o: ImgOpts): string {
		const im = this.image(key);
		if (!o.alt) throw new Error(`image ${key} has no alt text (use "decorative" for decorative art)`);
		const alt = o.alt === "decorative" ? "" : o.alt;
		const classes = ["px"];
		let w: number;
		let h: number;
		if (o.k && o.k > 0) {
			w = im.w * o.k;
			h = im.h * o.k;
		} else {
			const k = Math.max(o.min ?? 0, 1);
			w = im.w * k;
			h = im.h * k;
			classes.push(this.#fit(im, o.max ?? 1200, k));
		}
		if (o.class) classes.push(o.class);
		const loading = o.eager ? "" : ' loading="lazy"';
		return `<img src="${im.url}" width="${w}" height="${h}" alt="${escapeAttr(alt)}" class="${classes.join(" ")}"${loading} decoding="async">`;
	}

	#fit(im: Art, max: number, minK: number): string {
		const cls = `k-${sha256hex(`${im.key}|${max}|${minK}`).slice(0, 8)}`;
		this.#fits.set(cls, { cls, w: im.w, h: im.h, max, minK, gutter: 48 });
		return cls;
	}

	/**
	 * The rules for every responsive pixel image the pages used: the largest
	 * whole-number scale that fits the viewport (less the page gutters) and the
	 * image's maximum width.
	 */
	fitCSS(): string {
		const classes = [...this.#fits.keys()].sort();
		const steps: { minVW: number; rule: string }[] = [];
		let b = "/* Responsive pixel art: whole-number scales only (generated). */\n";
		for (const c of classes) {
			const f = this.#fits.get(c) as FitRule;
			b += `.${c}{width:${f.w * f.minK}px;height:${f.h * f.minK}px}\n`;
			for (let k = f.minK + 1; f.w * k <= f.max; k++) {
				steps.push({ minVW: f.w * k + f.gutter, rule: `.${c}{width:${f.w * k}px;height:${f.h * k}px}` });
			}
		}
		steps.sort((x, y) => x.minVW - y.minVW); // stable, like slices.SortStableFunc
		for (let i = 0; i < steps.length; ) {
			const at = steps[i]?.minVW ?? 0;
			b += `@media (min-width:${at}px){`;
			while (i < steps.length && steps[i]?.minVW === at) b += steps[i++]?.rule ?? "";
			b += "}\n";
		}
		return b;
	}
}
