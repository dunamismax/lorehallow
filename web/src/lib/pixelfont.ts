// Lorehallow's hand-drawn pixel font (web/static/fonts/lorehallow-pixel.txt):
// read the source, compile it into a TrueType font, and draw text with it
// onto images (the social cards) and into SVG (the progress badge).
//
// The source format is described at the top of that file: one grid of "#" and
// "." per glyph, capital height 7 rows, descenders in rows 8 and 9.
//
// Ported from the retired Go generator (ADR 0016); its TrueType output was
// checked byte for byte against the Go compiler's before the Go code was removed.
import { fillRect, type RGBA } from "./png.ts";

/** Rows above the baseline (capital height). */
export const CAP_ROWS = 7;
/** The most rows below it. */
export const DESCENT_ROWS = 2;
export const ROWS = CAP_ROWS + DESCENT_ROWS;

/** One character's pixels: bits[row][col], row 0 at the top of the capital height. */
export interface Glyph {
	codePoint: number;
	width: number;
	bits: boolean[][];
}

/** The glyph's width plus one pixel of spacing. */
export const advance = (g: Glyph): number => g.width + 1;

/** A rectangle in pixels: [x0, y0, x1, y1), like Go's image.Rectangle. */
export interface Rect {
	x0: number;
	y0: number;
	x1: number;
	y1: number;
}

/** A parsed pixel font. */
export class PixelFont {
	name = "";
	readonly glyphs = new Map<number, Glyph>();

	/** Reads the font source. Every error names its line. */
	static parse(src: string): PixelFont {
		const f = new PixelFont();
		let cur: Glyph | null = null;
		let rows: string[] = [];
		let start = 0;
		const finish = () => {
			if (cur === null) return;
			const ch = String.fromCodePoint(cur.codePoint);
			if (rows.length !== CAP_ROWS && rows.length !== ROWS) {
				throw new Error(`line ${start}: glyph "${ch}" has ${rows.length} rows; want ${CAP_ROWS} or ${ROWS}`);
			}
			cur.width = rows[0]?.length ?? 0;
			for (let i = 0; i < ROWS; i++) {
				const bits: boolean[] = new Array(cur.width).fill(false);
				cur.bits[i] = bits;
				const row = rows[i];
				if (row === undefined) continue;
				if (row.length !== cur.width) {
					throw new Error(
						`line ${start + i + 1}: glyph "${ch}" row ${i + 1} is ${row.length} wide; want ${cur.width}`,
					);
				}
				[...row].forEach((c, j) => {
					if (c === "#") bits[j] = true;
					else if (c !== ".")
						throw new Error(`line ${start + i + 1}: glyph "${ch}" has "${c}" (use # or a dot)`);
				});
			}
			if (f.glyphs.has(cur.codePoint)) throw new Error(`line ${start}: glyph "${ch}" defined twice`);
			f.glyphs.set(cur.codePoint, cur);
			cur = null;
			rows = [];
		};
		src.split("\n").forEach((raw, i) => {
			const n = i + 1;
			const line = raw.replace(/[ \t\r]+$/, "");
			if (line.startsWith("#") && cur === null) return; // comment
			if (line === "") finish();
			else if (line.startsWith("name: ")) f.name = line.slice("name: ".length).trim();
			else if (line.startsWith("glyph ")) {
				finish();
				cur = { codePoint: parseCodePoint(line.slice("glyph ".length), n), width: 0, bits: [] };
				start = n;
			} else if (cur !== null) rows.push(line);
			else throw new Error(`line ${n}: unexpected "${line}" outside a glyph`);
		});
		finish();
		if (f.name === "") throw new Error('missing "name:" line');
		if (!f.glyphs.has(0x20)) throw new Error("missing a space glyph (glyph U+0020)");
		return f;
	}

	/** The font's characters in code point order. */
	codePoints(): number[] {
		return [...this.glyphs.keys()].sort((a, b) => a - b);
	}

	/** Whether every character of s has a glyph. */
	has(s: string): boolean {
		return [...s].every((c) => this.glyphs.has(c.codePointAt(0) ?? 0));
	}

	/** The width of s in pixels (without the trailing spacing). */
	measure(s: string): number {
		let w = 0;
		for (const c of s) {
			const g = this.glyphs.get(c.codePointAt(0) ?? 0);
			if (g) w += advance(g);
		}
		return w > 0 ? w - 1 : 0;
	}

	/** The pixel squares for s with its capital top-left at (x, y), each font pixel scale×scale. */
	rects(s: string, x: number, y: number, scale: number): Rect[] {
		const out: Rect[] = [];
		for (const c of s) {
			const g = this.glyphs.get(c.codePointAt(0) ?? 0);
			if (!g) continue;
			for (let row = 0; row < ROWS; row++) {
				g.bits[row]?.forEach((on, col) => {
					if (!on) return;
					const px = x + col * scale;
					const py = y + row * scale;
					out.push({ x0: px, y0: py, x1: px + scale, y1: py + scale });
				});
			}
			x += advance(g) * scale;
		}
		return out;
	}

	/** Paints s onto an image. Characters without a glyph are skipped. */
	draw(img: RGBA, s: string, x: number, y: number, scale: number, color: number[]): void {
		for (const r of this.rects(s, x, y, scale)) fillRect(img, r.x0, r.y0, r.x1, r.y1, color);
	}

	/** Compiles the font into a TrueType file. Rebuilding it gives the same bytes. */
	ttf(): Uint8Array {
		return compileTTF(this);
	}
}

function parseCodePoint(s: string, line: number): number {
	if (s.startsWith("U+")) {
		const hex = s.slice(2);
		if (!/^[0-9A-Fa-f]+$/.test(hex)) throw new Error(`line ${line}: bad code point "${s}"`);
		return Number.parseInt(hex, 16);
	}
	const chars = [...s];
	if (chars.length !== 1) throw new Error(`line ${line}: glyph name "${s}" must be one character or U+XXXX`);
	return s.codePointAt(0) ?? 0;
}

// --- TrueType -----------------------------------------------------------------

/** Font units: one pixel is 100 units and the em is 10 pixels. */
export const UNIT = 100;
const UNITS_PER_EM = 1000;
const ASCENDER = 800; // capital height plus one pixel of headroom
const DESCENDER = -200;

interface Point {
	x: number;
	y: number;
}

/**
 * Traces the outline of a glyph's pixels: one closed loop per boundary,
 * clockwise around filled areas and counter-clockwise around holes
 * (TrueType's convention, y up), with straight runs merged.
 */
export function contours(g: Glyph): Point[][] {
	const filled = (r: number, c: number) =>
		r >= 0 && r < ROWS && c >= 0 && c < g.width && g.bits[r]?.[c] === true;
	// Corner (c, r) of the grid is at x = c*UNIT, y = (CAP_ROWS-r)*UNIT.
	const key = (p: Point) => `${p.x},${p.y}`;
	const next = new Map<string, Point[]>();
	const add = (a: Point, b: Point) => {
		const k = key(a);
		const list = next.get(k);
		if (list) list.push(b);
		else next.set(k, [b]);
	};
	for (let r = 0; r < ROWS; r++) {
		for (let c = 0; c < g.width; c++) {
			if (!filled(r, c)) continue;
			const tl = { x: c, y: r };
			const tr = { x: c + 1, y: r };
			const bl = { x: c, y: r + 1 };
			const br = { x: c + 1, y: r + 1 };
			if (!filled(r - 1, c)) add(tl, tr); // top edge, left to right
			if (!filled(r, c + 1)) add(tr, br); // right edge, downward
			if (!filled(r + 1, c)) add(br, bl); // bottom edge, right to left
			if (!filled(r, c - 1)) add(bl, tl); // left edge, upward
		}
	}
	// Walk the edges into loops. Where two loops touch at a corner, turn right
	// so each stays a simple polygon.
	const starts = [...next.keys()]
		.map((k) => {
			const [x, y] = k.split(",").map(Number);
			return { x: x ?? 0, y: y ?? 0 };
		})
		.sort((a, b) => (a.y !== b.y ? a.y - b.y : a.x - b.x));
	const loops: Point[][] = [];
	for (const s of starts) {
		for (;;) {
			const outs0 = next.get(key(s)) ?? [];
			if (outs0.length === 0) break;
			const loop: Point[] = [s];
			let prev = s;
			let cur = outs0.shift() as Point;
			while (cur.x !== s.x || cur.y !== s.y) {
				loop.push(cur);
				const outs = next.get(key(cur)) ?? [];
				let pick = 0;
				if (outs.length > 1) {
					const dx = cur.x - prev.x;
					const dy = cur.y - prev.y;
					// Right turn in grid coordinates (y down): (dx, dy) -> (-dy, dx).
					outs.forEach((o, i) => {
						if (o.x - cur.x === -dy && o.y - cur.y === dx) pick = i;
					});
				}
				prev = cur;
				cur = outs.splice(pick, 1)[0] as Point;
			}
			loops.push(simplify(loop));
		}
	}
	// Grid to font units.
	return loops.map((l) => l.map((p) => ({ x: p.x * UNIT, y: (CAP_ROWS - p.y) * UNIT })));
}

/** Drops points in the middle of straight runs. */
function simplify(loop: Point[]): Point[] {
	const n = loop.length;
	return loop.filter((p, i) => {
		const a = loop[(i + n - 1) % n] as Point;
		const b = loop[(i + 1) % n] as Point;
		return !((a.x === p.x && p.x === b.x) || (a.y === p.y && p.y === b.y));
	});
}

/** A growable big-endian byte buffer. */
class Bytes {
	#buf = new Uint8Array(256);
	#view = new DataView(this.#buf.buffer);
	length = 0;

	#grow(n: number): void {
		if (this.length + n <= this.#buf.length) return;
		const next = new Uint8Array(Math.max(this.#buf.length * 2, this.length + n));
		next.set(this.#buf);
		this.#buf = next;
		this.#view = new DataView(next.buffer);
	}
	u8(v: number): this {
		this.#grow(1);
		this.#view.setUint8(this.length, v & 0xff);
		this.length += 1;
		return this;
	}
	u16(...vs: number[]): this {
		for (const v of vs) {
			this.#grow(2);
			this.#view.setUint16(this.length, v & 0xffff);
			this.length += 2;
		}
		return this;
	}
	i16(...vs: number[]): this {
		for (const v of vs) {
			this.#grow(2);
			this.#view.setInt16(this.length, v);
			this.length += 2;
		}
		return this;
	}
	u32(...vs: number[]): this {
		for (const v of vs) {
			this.#grow(4);
			this.#view.setUint32(this.length, v >>> 0);
			this.length += 4;
		}
		return this;
	}
	i64(v: bigint): this {
		this.#grow(8);
		this.#view.setBigInt64(this.length, v);
		this.length += 8;
		return this;
	}
	raw(b: Uint8Array): this {
		this.#grow(b.length);
		this.#buf.set(b, this.length);
		this.length += b.length;
		return this;
	}
	bytes(): Uint8Array {
		return this.#buf.slice(0, this.length);
	}
}

interface GlyphData {
	advance: number;
	data: Uint8Array;
	xMin: number;
	bbox: [number, number, number, number];
	points: number;
	loops: number;
	empty: boolean;
}

function encodeGlyph(loops: Point[][], adv: number): GlyphData {
	const gd: GlyphData = {
		advance: adv,
		data: new Uint8Array(0),
		xMin: 0,
		bbox: [0, 0, 0, 0],
		points: 0,
		loops: loops.length,
		empty: false,
	};
	if (loops.length === 0) {
		gd.empty = true;
		return gd;
	}
	const bb: [number, number, number, number] = [2 ** 30, 2 ** 30, -(2 ** 30), -(2 ** 30)];
	for (const l of loops) {
		for (const p of l) {
			bb[0] = Math.min(bb[0], p.x);
			bb[1] = Math.min(bb[1], p.y);
			bb[2] = Math.max(bb[2], p.x);
			bb[3] = Math.max(bb[3], p.y);
			gd.points++;
		}
	}
	gd.bbox = bb;
	gd.xMin = bb[0];
	const b = new Bytes();
	b.i16(loops.length, ...bb);
	let end = -1;
	for (const l of loops) {
		end += l.length;
		b.u16(end);
	}
	b.u16(0); // no instructions
	for (let i = 0; i < gd.points; i++) b.u8(0x01); // on-curve, x and y as int16 deltas
	let px = 0;
	let py = 0;
	const ys: number[] = [];
	for (const l of loops) {
		for (const p of l) {
			b.i16(p.x - px);
			ys.push(p.y - py);
			px = p.x;
			py = p.y;
		}
	}
	b.i16(...ys);
	while (b.length % 4 !== 0) b.u8(0);
	gd.data = b.bytes();
	return gd;
}

function compileTTF(f: PixelFont): Uint8Array {
	const cps = f.codePoints();
	if (cps.length > 1000) throw new Error("too many glyphs");
	// Glyph 0 is .notdef: a hollow box.
	const notdef: Glyph = { codePoint: 0, width: 4, bits: [] };
	for (let r = 0; r < ROWS; r++) {
		notdef.bits[r] = [0, 1, 2, 3].map(
			(c) => r < CAP_ROWS && (r === 0 || r === CAP_ROWS - 1 || c === 0 || c === 3),
		);
	}
	const glyphs = [notdef, ...cps.map((c) => f.glyphs.get(c) as Glyph)].map((g) =>
		encodeGlyph(contours(g), advance(g) * UNIT),
	);

	// glyf and loca (long offsets).
	const glyf = new Bytes();
	const loca = new Bytes();
	const bb = [2 ** 30, 2 ** 30, -(2 ** 30), -(2 ** 30)];
	let maxPoints = 0;
	let maxContours = 0;
	let advMax = 0;
	let minLSB = 2 ** 30;
	let minRSB = 2 ** 30;
	let maxExtent = -(2 ** 30);
	let advSum = 0;
	for (const g of glyphs) {
		loca.u32(glyf.length);
		glyf.raw(g.data);
		advMax = Math.max(advMax, g.advance);
		advSum += g.advance;
		if (g.empty) continue;
		bb[0] = Math.min(bb[0] ?? 0, g.bbox[0]);
		bb[1] = Math.min(bb[1] ?? 0, g.bbox[1]);
		bb[2] = Math.max(bb[2] ?? 0, g.bbox[2]);
		bb[3] = Math.max(bb[3] ?? 0, g.bbox[3]);
		maxPoints = Math.max(maxPoints, g.points);
		maxContours = Math.max(maxContours, g.loops);
		minLSB = Math.min(minLSB, g.xMin);
		minRSB = Math.min(minRSB, g.advance - g.bbox[2]);
		maxExtent = Math.max(maxExtent, g.bbox[2]);
	}
	loca.u32(glyf.length);
	const n = glyphs.length;

	const head = new Bytes()
		.u32(0x00010000, 0x00010000, 0, 0x5f0f3cf5)
		.u16(0x0001, UNITS_PER_EM)
		.i64(3_870_000_000n) // fixed dates keep the build reproducible
		.i64(3_870_000_000n)
		.i16(bb[0] ?? 0, bb[1] ?? 0, bb[2] ?? 0, bb[3] ?? 0)
		.u16(0, 8)
		.i16(2, 1, 0);

	const hhea = new Bytes()
		.u32(0x00010000)
		.i16(ASCENDER, DESCENDER, 0)
		.u16(advMax)
		.i16(minLSB, minRSB, maxExtent, 1, 0, 0, 0, 0, 0, 0, 0)
		.u16(n);

	const maxp = new Bytes().u32(0x00010000).u16(n, maxPoints, maxContours, 0, 0, 2, 0, 0, 0, 0, 0, 0, 0, 0);

	const hmtx = new Bytes();
	for (const g of glyphs) hmtx.u16(g.advance).i16(g.xMin);

	const first = cps[0] ?? 0;
	const last = cps.at(-1) ?? 0;
	const os2 = new Bytes()
		.u16(4)
		.i16(Math.trunc(advSum / n))
		.u16(400, 5, 0)
		.i16(650, 600, 0, 100, 650, 600, 0, 400, UNIT, 300, 0)
		.raw(new Uint8Array(10)) // panose
		.u32((1 | (1 << 1) | (1 << 31)) >>> 0, 0, 0, 0) // Basic Latin, Latin-1, General Punctuation
		.raw(new TextEncoder().encode("LORE"))
		.u16(0x0040 | 0x0080, Math.min(first, 0xffff), Math.min(last, 0xffff))
		.i16(ASCENDER, DESCENDER, 0)
		.u16(ASCENDER, -DESCENDER)
		.u32(1, 0)
		.i16(500, CAP_ROWS * UNIT)
		.u16(0, 0x20, 1);

	// cmap: format 4, one segment per character, shared by the Unicode (0, 3)
	// and Windows (3, 1) encodings.
	const segs = cps.length + 1;
	let searchRange = 1;
	let entrySelector = 0;
	while (searchRange * 2 <= segs) {
		searchRange *= 2;
		entrySelector++;
	}
	searchRange *= 2;
	const sub = new Bytes().u16(
		4,
		16 + segs * 8,
		0,
		segs * 2,
		searchRange,
		entrySelector,
		segs * 2 - searchRange,
	);
	sub.u16(...cps, 0xffff, 0);
	sub.u16(...cps, 0xffff);
	sub.u16(...cps.map((c, i) => (i + 1 - c) & 0xffff), 1);
	sub.u16(...new Array(segs).fill(0));
	const cmap = new Bytes().u16(0, 2, 0, 3).u32(20).u16(3, 1).u32(20).raw(sub.bytes());

	const family = f.name;
	const ps = "LorehallowPixel-Regular";
	const names: [number, string][] = [
		[0, "Hand-drawn for lorehallow.com"],
		[1, family],
		[2, "Regular"],
		[3, `${ps};1.000`],
		[4, `${family} Regular`],
		[5, "Version 1.000"],
		[6, ps],
		[13, "Creative Commons Attribution-NonCommercial 4.0 International"],
		[14, "https://creativecommons.org/licenses/by-nc/4.0/"],
	];
	const nameRecs = new Bytes();
	const nameStr = new Bytes();
	for (const [id, s] of names) {
		const units: number[] = [];
		for (let i = 0; i < s.length; i++) units.push(s.charCodeAt(i)); // UTF-16, like utf16.Encode
		nameRecs.u16(3, 1, 0x0409, id, units.length * 2, nameStr.length);
		nameStr.u16(...units);
	}
	const name = new Bytes()
		.u16(0, names.length, 6 + 12 * names.length)
		.raw(nameRecs.bytes())
		.raw(nameStr.bytes());

	const post = new Bytes().u32(0x00030000, 0).i16(-UNIT, UNIT).u32(0, 0, 0, 0, 0);

	return assemble({
		"OS/2": os2.bytes(),
		cmap: cmap.bytes(),
		glyf: glyf.bytes(),
		head: head.bytes(),
		hhea: hhea.bytes(),
		hmtx: hmtx.bytes(),
		loca: loca.bytes(),
		maxp: maxp.bytes(),
		name: name.bytes(),
		post: post.bytes(),
	});
}

/** The sfnt checksum: the sum of big-endian 32-bit words, zero-padded. */
export function checksum(b: Uint8Array): number {
	let sum = 0;
	for (let i = 0; i < b.length; i += 4) {
		const w = ((b[i] ?? 0) << 24) | ((b[i + 1] ?? 0) << 16) | ((b[i + 2] ?? 0) << 8) | (b[i + 3] ?? 0);
		sum = (sum + (w >>> 0)) >>> 0;
	}
	return sum;
}

/** Writes the sfnt container and fixes head.checkSumAdjustment. */
function assemble(tables: Record<string, Uint8Array>): Uint8Array {
	// Go sorts table tags as byte strings: uppercase before lowercase.
	const tags = Object.keys(tables).sort((a, b) => (a < b ? -1 : a > b ? 1 : 0));
	const n = tags.length;
	let sr = 1;
	let es = 0;
	while (sr * 2 <= n) {
		sr *= 2;
		es++;
	}
	const hdr = new Uint8Array(12 + 16 * n);
	const hv = new DataView(hdr.buffer);
	hv.setUint32(0, 0x00010000);
	hv.setUint16(4, n);
	hv.setUint16(6, sr * 16);
	hv.setUint16(8, es);
	hv.setUint16(10, n * 16 - sr * 16);
	const body = new Bytes();
	let headAt = 0;
	tags.forEach((t, i) => {
		const data = tables[t] as Uint8Array;
		const rec = 12 + 16 * i;
		for (let k = 0; k < 4; k++) hdr[rec + k] = t.charCodeAt(k);
		hv.setUint32(rec + 4, checksum(data));
		hv.setUint32(rec + 8, hdr.length + body.length);
		hv.setUint32(rec + 12, data.length);
		if (t === "head") headAt = hdr.length + body.length;
		body.raw(data);
		while (body.length % 4 !== 0) body.u8(0);
	});
	const out = new Uint8Array(hdr.length + body.length);
	out.set(hdr);
	out.set(body.bytes(), hdr.length);
	new DataView(out.buffer).setUint32(headAt + 8, (0xb1b0afba - checksum(out)) >>> 0);
	return out;
}
