// web/data/releases.json: every version of the browser game that went live at
// play.lorehallow.com, newest first (schema in web/data/README.md). The build
// fails on any entry that doesn't match it.
import { readFileSync } from "node:fs";

export const PlayURL = "https://play.lorehallow.com/";

export interface Release {
	/** "v0.2.0". */
	version: string;
	/** "YYYY-MM-DD". */
	date: string;
	channel: "alpha" | "beta" | "stable";
	/** A site path or an https URL, or "". */
	notes: string;
	/** The PROTOCOL_VERSION this build speaks. */
	protocol: number;
}

const versionRE = /^v\d+\.\d+\.\d+$/;
const dateRE = /^\d{4}-\d{2}-\d{2}$/;
const keys = ["version", "date", "channel", "notes", "protocol"];

/** Checks release data and returns it. Every error names the release and the field. */
export function parseReleases(text: string): Release[] {
	let doc: unknown;
	try {
		doc = JSON.parse(text);
	} catch (err) {
		throw new Error(`releases.json: ${(err as Error).message}`);
	}
	const list = (doc as { releases?: unknown }).releases;
	if (
		typeof doc !== "object" ||
		doc === null ||
		Object.keys(doc).join() !== "releases" ||
		!Array.isArray(list)
	) {
		throw new Error('releases.json: must be {"releases": [...]}');
	}
	let prev = "";
	return list.map((raw, i) => {
		const where = `releases.json release ${i}`;
		const r = raw as Record<string, unknown>;
		for (const k of Object.keys(r)) if (!keys.includes(k)) throw new Error(`${where}: unknown field "${k}"`);
		const { version, date, channel, notes = "", protocol } = r;
		if (typeof version !== "string" || !versionRE.test(version)) {
			throw new Error(`${where}: version "${String(version)}" must look like v0.2.0`);
		}
		if (typeof date !== "string" || !dateRE.test(date) || Number.isNaN(Date.parse(date))) {
			throw new Error(`${where}: date must be YYYY-MM-DD`);
		}
		if (prev !== "" && date > prev) throw new Error(`${where}: releases must be listed newest first`);
		prev = date;
		if (channel !== "alpha" && channel !== "beta" && channel !== "stable") {
			throw new Error(`${where}: channel must be alpha, beta or stable`);
		}
		if (
			typeof notes !== "string" ||
			(notes !== "" && !notes.startsWith("/") && !notes.startsWith("https://"))
		) {
			throw new Error(`${where}: notes must be a site path or an https URL`);
		}
		if (typeof protocol !== "number" || !Number.isInteger(protocol) || protocol < 1) {
			throw new Error(`${where}: protocol must be the PROTOCOL_VERSION it speaks (1 or more)`);
		}
		return { version, date, channel, notes, protocol };
	});
}

export function loadReleases(file: string): Release[] {
	return parseReleases(readFileSync(file, "utf8"));
}
