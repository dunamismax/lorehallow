// A small PNG reader and writer for the site's pixel art, on node:zlib.
//
// It reads the 8-bit, non-interlaced PNGs the art pipeline writes (grey,
// grey and alpha, RGB, RGBA and palette, with tRNS) and writes 8-bit RGB or
// RGBA, choosing each row's filter the way Go's image/png does. Nothing here
// resamples: pixel art goes in and comes out with exactly the same pixels.
import { deflateSync, inflateSync } from "node:zlib";

/** An image as straight (non-premultiplied) RGBA bytes, row by row. */
export interface RGBA {
	width: number;
	height: number;
	data: Uint8Array;
}

const signature = Uint8Array.of(0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a);

/** A blank (transparent) image. */
export function blank(width: number, height: number): RGBA {
	return { width, height, data: new Uint8Array(width * height * 4) };
}

/** Decodes a PNG. */
export function decodePNG(file: Uint8Array, name = "png"): RGBA {
	if (file.length < 8 || !signature.every((b, i) => file[i] === b)) throw new Error(`${name}: not a PNG`);
	const view = new DataView(file.buffer, file.byteOffset, file.byteLength);
	let width = 0;
	let height = 0;
	let colorType = -1;
	let palette: Uint8Array | null = null;
	let trns: Uint8Array | null = null;
	const idat: Uint8Array[] = [];
	for (let at = 8; at + 8 <= file.length; ) {
		const len = view.getUint32(at);
		const type = String.fromCharCode(...file.subarray(at + 4, at + 8));
		const body = file.subarray(at + 8, at + 8 + len);
		at += 12 + len;
		if (type === "IHDR") {
			const ihdr = new DataView(body.buffer, body.byteOffset, body.byteLength);
			width = ihdr.getUint32(0);
			height = ihdr.getUint32(4);
			const depth = body[8];
			colorType = body[9] ?? -1;
			if (depth !== 8) throw new Error(`${name}: bit depth ${depth} isn't supported (want 8)`);
			if (body[12] !== 0) throw new Error(`${name}: interlaced PNGs aren't supported`);
		} else if (type === "PLTE") {
			palette = body;
		} else if (type === "tRNS") {
			trns = body;
		} else if (type === "IDAT") {
			idat.push(body);
		} else if (type === "IEND") {
			break;
		}
	}
	const channels = { 0: 1, 2: 3, 3: 1, 4: 2, 6: 4 }[colorType];
	if (channels === undefined) throw new Error(`${name}: color type ${colorType} isn't supported`);
	if (colorType === 3 && !palette) throw new Error(`${name}: palette image without PLTE`);
	const raw = inflateSync(Buffer.concat(idat));
	const stride = width * channels;
	if (raw.length < height * (stride + 1)) throw new Error(`${name}: image data is truncated`);
	const rows = unfilter(raw, width, height, channels);
	const out = blank(width, height);
	const d = out.data;
	for (let i = 0, p = 0; i < width * height; i++, p += channels) {
		const o = i * 4;
		switch (colorType) {
			case 0: {
				const g = rows[p] ?? 0;
				d[o] = d[o + 1] = d[o + 2] = g;
				d[o + 3] = trns && trns.length >= 2 && ((trns[0] ?? 0) << 8) + (trns[1] ?? 0) === g ? 0 : 255;
				break;
			}
			case 2: {
				const r = rows[p] ?? 0;
				const g = rows[p + 1] ?? 0;
				const b = rows[p + 2] ?? 0;
				d[o] = r;
				d[o + 1] = g;
				d[o + 2] = b;
				const transparent =
					trns !== null && trns.length >= 6 && trns[1] === r && trns[3] === g && trns[5] === b;
				d[o + 3] = transparent ? 0 : 255;
				break;
			}
			case 3: {
				const k = rows[p] ?? 0;
				d[o] = palette?.[k * 3] ?? 0;
				d[o + 1] = palette?.[k * 3 + 1] ?? 0;
				d[o + 2] = palette?.[k * 3 + 2] ?? 0;
				d[o + 3] = trns && k < trns.length ? (trns[k] ?? 255) : 255;
				break;
			}
			case 4:
				d[o] = d[o + 1] = d[o + 2] = rows[p] ?? 0;
				d[o + 3] = rows[p + 1] ?? 0;
				break;
			default:
				d[o] = rows[p] ?? 0;
				d[o + 1] = rows[p + 1] ?? 0;
				d[o + 2] = rows[p + 2] ?? 0;
				d[o + 3] = rows[p + 3] ?? 0;
		}
	}
	return out;
}

function paeth(a: number, b: number, c: number): number {
	const p = a + b - c;
	const pa = Math.abs(p - a);
	const pb = Math.abs(p - b);
	const pc = Math.abs(p - c);
	if (pa <= pb && pa <= pc) return a;
	return pb <= pc ? b : c;
}

/** Undoes the per-row filters and returns the packed pixel bytes. */
function unfilter(raw: Uint8Array, width: number, height: number, bpp: number): Uint8Array {
	const stride = width * bpp;
	const out = new Uint8Array(stride * height);
	for (let y = 0; y < height; y++) {
		const filter = raw[y * (stride + 1)];
		const src = y * (stride + 1) + 1;
		const row = y * stride;
		const up = row - stride;
		for (let x = 0; x < stride; x++) {
			const v = raw[src + x] ?? 0;
			const a = x >= bpp ? (out[row + x - bpp] ?? 0) : 0;
			const b = y > 0 ? (out[up + x] ?? 0) : 0;
			const c = y > 0 && x >= bpp ? (out[up + x - bpp] ?? 0) : 0;
			let r: number;
			switch (filter) {
				case 0:
					r = v;
					break;
				case 1:
					r = v + a;
					break;
				case 2:
					r = v + b;
					break;
				case 3:
					r = v + ((a + b) >> 1);
					break;
				case 4:
					r = v + paeth(a, b, c);
					break;
				default:
					throw new Error(`unknown PNG filter ${filter}`);
			}
			out[row + x] = r & 0xff;
		}
	}
	return out;
}

const crcTable = (() => {
	const t = new Uint32Array(256);
	for (let n = 0; n < 256; n++) {
		let c = n;
		for (let k = 0; k < 8; k++) c = c & 1 ? 0xedb88320 ^ (c >>> 1) : c >>> 1;
		t[n] = c >>> 0;
	}
	return t;
})();

function crc32(bytes: Uint8Array): number {
	let c = 0xffffffff;
	for (const b of bytes) c = (crcTable[(c ^ b) & 0xff] ?? 0) ^ (c >>> 8);
	return (c ^ 0xffffffff) >>> 0;
}

function chunk(type: string, body: Uint8Array): Uint8Array {
	const out = new Uint8Array(12 + body.length);
	const view = new DataView(out.buffer);
	view.setUint32(0, body.length);
	for (let i = 0; i < 4; i++) out[4 + i] = type.charCodeAt(i);
	out.set(body, 8);
	view.setUint32(8 + body.length, crc32(out.subarray(4, 8 + body.length)));
	return out;
}

/**
 * Encodes an image as an 8-bit PNG: RGB when every pixel is opaque, RGBA
 * otherwise, the same choice Go's image/png makes. Each row gets the filter
 * whose output has the smallest sum of absolute values, as in Go.
 */
export function encodePNG(img: RGBA): Uint8Array {
	const { width, height, data } = img;
	let opaque = true;
	for (let i = 3; i < data.length; i += 4) {
		if (data[i] !== 255) {
			opaque = false;
			break;
		}
	}
	const bpp = opaque ? 3 : 4;
	const stride = width * bpp;
	const pixels = new Uint8Array(stride * height);
	for (let i = 0, p = 0; i < width * height; i++) {
		pixels[p++] = data[i * 4] ?? 0;
		pixels[p++] = data[i * 4 + 1] ?? 0;
		pixels[p++] = data[i * 4 + 2] ?? 0;
		if (!opaque) pixels[p++] = data[i * 4 + 3] ?? 0;
	}
	const raw = new Uint8Array((stride + 1) * height);
	const candidate = new Uint8Array(stride);
	const best = new Uint8Array(stride);
	for (let y = 0; y < height; y++) {
		const row = y * stride;
		let bestFilter = 0;
		let bestScore = Number.POSITIVE_INFINITY;
		for (let f = 0; f < 5; f++) {
			let score = 0;
			for (let x = 0; x < stride; x++) {
				const v = pixels[row + x] ?? 0;
				const a = x >= bpp ? (pixels[row + x - bpp] ?? 0) : 0;
				const b = y > 0 ? (pixels[row - stride + x] ?? 0) : 0;
				const c = y > 0 && x >= bpp ? (pixels[row - stride + x - bpp] ?? 0) : 0;
				const pred = [0, a, b, (a + b) >> 1, paeth(a, b, c)][f] ?? 0;
				const out = (v - pred) & 0xff;
				candidate[x] = out;
				score += out < 128 ? out : 256 - out;
			}
			if (score < bestScore) {
				bestScore = score;
				bestFilter = f;
				best.set(candidate);
			}
		}
		raw[y * (stride + 1)] = bestFilter;
		raw.set(best, y * (stride + 1) + 1);
	}
	const ihdr = new Uint8Array(13);
	const v = new DataView(ihdr.buffer);
	v.setUint32(0, width);
	v.setUint32(4, height);
	ihdr[8] = 8;
	ihdr[9] = opaque ? 2 : 6;
	return Buffer.concat([
		signature,
		chunk("IHDR", ihdr),
		chunk("IDAT", deflateSync(raw)),
		chunk("IEND", new Uint8Array(0)),
	]);
}

/** Copies a rectangle out of an image. */
export function crop(img: RGBA, x: number, y: number, w: number, h: number): RGBA {
	const out = blank(w, h);
	for (let row = 0; row < h; row++) {
		const from = ((y + row) * img.width + x) * 4;
		out.data.set(img.data.subarray(from, from + w * 4), row * w * 4);
	}
	return out;
}

/** The image followed by its mirror image, which tiles seamlessly in a row. */
export function mirrorPair(img: RGBA): RGBA {
	const { width: w, height: h } = img;
	const out = blank(2 * w, h);
	for (let y = 0; y < h; y++) {
		for (let x = 0; x < w; x++) {
			const px = img.data.subarray((y * w + x) * 4, (y * w + x) * 4 + 4);
			out.data.set(px, (y * 2 * w + x) * 4);
			out.data.set(px, (y * 2 * w + (2 * w - 1 - x)) * 4);
		}
	}
	return out;
}

/** Enlarges an image by a whole number, nearest neighbor. */
export function upscale(img: RGBA, k: number): RGBA {
	const out = blank(img.width * k, img.height * k);
	for (let y = 0; y < out.height; y++) {
		for (let x = 0; x < out.width; x++) {
			const from = (Math.floor(y / k) * img.width + Math.floor(x / k)) * 4;
			out.data.set(img.data.subarray(from, from + 4), (y * out.width + x) * 4);
		}
	}
	return out;
}

/** Fills a rectangle with an opaque color, clipped to the image. */
export function fillRect(img: RGBA, x0: number, y0: number, x1: number, y1: number, rgb: number[]): void {
	for (let y = Math.max(0, y0); y < Math.min(img.height, y1); y++) {
		for (let x = Math.max(0, x0); x < Math.min(img.width, x1); x++) {
			const o = (y * img.width + x) * 4;
			img.data[o] = rgb[0] ?? 0;
			img.data[o + 1] = rgb[1] ?? 0;
			img.data[o + 2] = rgb[2] ?? 0;
			img.data[o + 3] = 255;
		}
	}
}
