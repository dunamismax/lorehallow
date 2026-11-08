// BUILD.md as the site shows it: per-phase progress, the current phase, the
// unchecked owner items, and the "quests complete" feed from git history.
// A port of web/internal/site/progress.go.
import { execFileSync } from "node:child_process";
import { readdirSync } from "node:fs";
import { join } from "node:path";
import {
	ACCEPTANCE,
	currentPhase,
	type Item,
	type Phase,
	type Plan,
	phaseItems,
	phaseProgress,
	planProgress,
	type Section,
} from "./buildmd.ts";
import { inline, RepoURL } from "./markdown.ts";
import type { PixelFont } from "./pixelfont.ts";
import { CAP_ROWS } from "./pixelfont.ts";
import type { Theme } from "./theme.ts";

export interface ItemView extends Item {
	phase: PhaseView;
	html: string;
}

export interface SectionView extends Omit<Section, "items"> {
	items: ItemView[];
	done: number;
	acceptance: boolean;
}

export interface PhaseView extends Omit<Phase, "sections"> {
	done: number;
	total: number;
	percent: number;
	state: "done" | "current" | "future";
	url: string;
	/** Frame name in progress_landmarks. */
	landmark: string;
	/** GitHub URL of the phase's walkthrough, or "". */
	walkthrough: string;
	sections: SectionView[];
	/** From the Ships metadata, e.g. "v0.2.0", or "". */
	version: string;
	prev: PhaseView | null;
	next: PhaseView | null;
}

/** A box that was checked in a commit. */
export interface Achievement {
	text: string;
	html: string;
	/** null if the item has since been reworded. */
	phase: PhaseView | null;
	date: Date;
	commit: string;
	url: string;
}

export interface Progress {
	plan: Plan;
	phases: PhaseView[];
	done: number;
	total: number;
	percent: number;
	/** null when everything is done. */
	current: PhaseView | null;
	/** Checked boxes, newest first, from git history. */
	achievements: Achievement[];
	/** Unchecked HUMAN items in the current phase. */
	waiting: ItemView[];
}

/** Rounds down, so 100% means every box is checked. */
export function percent(done: number, total: number): number {
	return total === 0 ? 0 : Math.floor((done * 100) / total);
}

const walkthroughRE = /^phase-(\d+)-[a-z0-9-]+\.md$/;

export async function buildProgress(
	root: string,
	plan: Plan,
	landmarks: string[],
	git: boolean,
): Promise<Progress> {
	const walks = new Map<number, string>();
	try {
		for (const name of readdirSync(join(root, "docs", "walkthroughs")).sort()) {
			const m = walkthroughRE.exec(name);
			if (m) walks.set(Number(m[1]), `${RepoURL}/blob/main/docs/walkthroughs/${name}`);
		}
	} catch {
		// No walkthroughs yet.
	}
	const pr: Progress = {
		plan,
		phases: [],
		done: 0,
		total: 0,
		percent: 0,
		current: null,
		achievements: [],
		waiting: [],
	};
	const current = currentPhase(plan);
	let state: PhaseView["state"] = "done";
	for (const p of plan.phases) {
		const pp = phaseProgress(p);
		if (current && p.number === current.number) state = "current";
		else if (state === "current") state = "future";
		const v: PhaseView = {
			...p,
			done: pp.done,
			total: pp.total,
			percent: percent(pp.done, pp.total),
			state,
			url: `/progress/phase-${p.number}/`,
			landmark: landmarks[p.number] ?? "",
			walkthrough: walks.get(p.number) ?? "",
			sections: [],
			version: /v\d+\.\d+\.\d+/.exec(p.meta.Ships ?? "")?.[0] ?? "",
			prev: null,
			next: null,
		};
		for (const s of p.sections) {
			const sv: SectionView = { ...s, items: [], done: 0, acceptance: s.id === ACCEPTANCE };
			for (const it of s.items) {
				const iv: ItemView = { ...it, phase: v, html: await inline(it.text) };
				sv.items.push(iv);
				if (it.checked) sv.done++;
				if (v.state === "current" && !it.checked && it.tag === "HUMAN") pr.waiting.push(iv);
			}
			v.sections.push(sv);
		}
		const last = pr.phases.at(-1);
		if (last) {
			v.prev = last;
			last.next = v;
		}
		pr.phases.push(v);
		if (v.state === "current") pr.current = v;
	}
	const total = planProgress(plan);
	pr.done = total.done;
	pr.total = total.total;
	pr.percent = percent(total.done, total.total);
	if (git) {
		const byText = new Map<string, PhaseView>();
		for (const p of pr.phases) {
			for (const it of phaseItems(plan.phases.find((x) => x.number === p.number) as Phase))
				byText.set(it.text, p);
		}
		const log = execFileSync(
			"git",
			[
				"-C",
				root,
				"log",
				"--reverse",
				"--no-merges",
				"-p",
				"-U0",
				"--format=commit %H %ct",
				"--",
				"BUILD.md",
			],
			{ maxBuffer: 1 << 30, encoding: "utf8" },
		);
		pr.achievements = await Promise.all(
			parseAchievements(log).map(async (a) => ({
				...a,
				phase: byText.get(a.text) ?? null,
				html: await inline(a.text),
			})),
		);
	}
	return pr;
}

const diffCheckedRE = /^\+- \[x\] (?:`(?:HUMAN|ADR)` )?(.*)$/u;
const diffRemovedRE = /^-- \[( |x)\] (?:`(?:HUMAN|ADR)` )?(.*)$/u;
const commitLineRE = /^commit ([0-9a-f]{40}) (\d+)$/;

export type RawAchievement = Omit<Achievement, "html" | "phase">;

/**
 * Every box that was checked in BUILD.md's history, with the commit that
 * checked it, newest first.
 *
 * In each commit, an added "- [x]" line is a checked box when the same commit
 * removed that line unchecked, or added it new. An added checked line that
 * pairs with a removed checked line is a reword or a move, not news.
 */
export function parseAchievements(log: string): RawAchievement[] {
	const all: RawAchievement[] = [];
	let hash = "";
	let when = new Date(0);
	let added: string[] = [];
	let unchecked = new Set<string>();
	let removedChecked = new Map<string, number>();
	const entry = (text: string): RawAchievement => ({
		text,
		date: when,
		commit: hash.slice(0, 7),
		url: `${RepoURL}/commit/${hash}`,
	});
	const flush = () => {
		const maybe: string[] = [];
		for (const t of added) {
			const n = removedChecked.get(t) ?? 0;
			if (unchecked.has(t)) all.push(entry(t));
			else if (n > 0) removedChecked.set(t, n - 1);
			else maybe.push(t);
		}
		// Whatever checked lines were removed and not re-added verbatim pair up
		// with these as rewords; the rest are new, already-checked items.
		let left = 0;
		for (const n of removedChecked.values()) left += n;
		for (const t of maybe) {
			if (left > 0) {
				left--;
				continue;
			}
			all.push(entry(t));
		}
		added = [];
		unchecked = new Set();
		removedChecked = new Map();
	};
	for (const raw of log.split("\n")) {
		const line = raw.endsWith("\r") ? raw.slice(0, -1) : raw;
		const c = commitLineRE.exec(line);
		if (c) {
			if (hash !== "") flush();
			hash = c[1] ?? "";
			when = new Date(Number(c[2]) * 1000);
			continue;
		}
		const a = diffCheckedRE.exec(line);
		if (a) {
			added.push((a[1] ?? "").trim());
			continue;
		}
		const r = diffRemovedRE.exec(line);
		if (r) {
			const t = (r[2] ?? "").trim();
			if (r[1] === " ") unchecked.add(t);
			else removedChecked.set(t, (removedChecked.get(t) ?? 0) + 1);
		}
	}
	if (hash !== "") flush();
	return all.reverse();
}

const BADGE_PAD = 6;

function badgeText(pr: Progress, f: PixelFont) {
	const label = "lorehallow";
	const value = pr.current ? `phase ${pr.current.number} · ${pr.percent}%` : `${pr.percent}% built`;
	return { label, value, lw: f.measure(label) + 2 * BADGE_PAD, vw: f.measure(value) + 2 * BADGE_PAD };
}

/** The badge's width in pixels. */
export function badgeWidth(pr: Progress, f: PixelFont): number {
	const { lw, vw } = badgeText(pr, f);
	return lw + vw;
}

/** /progress.svg, drawn with the hand-drawn pixel font so it looks the same everywhere and needs no font. */
export function badgeSVG(pr: Progress, f: PixelFont, t: Theme): string {
	const { label, value, lw, vw } = badgeText(pr, f);
	const h = 20;
	const scale = 1;
	const top = Math.trunc((h - CAP_ROWS * scale) / 2);
	let b = `<svg xmlns="http://www.w3.org/2000/svg" width="${lw + vw}" height="${h}" viewBox="0 0 ${lw + vw} ${h}" shape-rendering="crispEdges" role="img" aria-label="${label}: ${value}">`;
	b += `<title>${label}: ${value}</title>`;
	b += `<rect width="${lw}" height="${h}" fill="${t.hex("night-plum")}"/>`;
	b += `<rect x="${lw}" width="${vw}" height="${h}" fill="${t.hex("lantern-gold")}"/>`;
	const path = (s: string, x: number, color: string) => {
		b += `<path fill="${color}" d="`;
		for (const r of f.rects(s, x, top, scale)) {
			const dx = r.x1 - r.x0;
			const dy = r.y1 - r.y0;
			b += `M${r.x0} ${r.y0}h${dx}v${dy}h-${dx}z`;
		}
		b += `"/>`;
	};
	path(label, BADGE_PAD, t.hex("parchment-white"));
	path(value, lw + BADGE_PAD, t.hex("ink"));
	return `${b}</svg>\n`;
}

const isoDate = (d: Date) => d.toISOString().slice(0, 10);

/** /progress.json. */
export function progressJSON(pr: Progress, base: string, built: Date, commit: string): string {
	const doc = {
		generated: `${built.toISOString().slice(0, 19)}Z`,
		commit,
		done: pr.done,
		total: pr.total,
		percent: pr.percent,
		current: pr.current
			? { number: pr.current.number, title: pr.current.title, url: base + pr.current.url }
			: null,
		phases: pr.phases.map((p) => ({
			number: p.number,
			title: p.title,
			done: p.done,
			total: p.total,
			percent: p.percent,
			state: p.state,
			url: base + p.url,
		})),
		waiting_on_owner: pr.waiting.map((w) => ({ text: w.text, phase: w.phase.number })),
		recent_achievements: pr.achievements.slice(0, 20).map((a) => ({
			text: a.text,
			phase: a.phase ? a.phase.number : null,
			date: isoDate(a.date),
			commit: a.commit,
		})),
	};
	return `${goJSON(doc)}\n`;
}

/** JSON.stringify with two-space indents and Go's encoding/json escaping of <, > and &. */
function goJSON(v: unknown): string {
	return JSON.stringify(v, null, 2).replace(
		/[<>&\u2028\u2029]/g,
		(c) => `\\u${c.charCodeAt(0).toString(16).padStart(4, "0")}`,
	);
}
