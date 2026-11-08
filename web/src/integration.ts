// The lorehallow integration: once Astro has rendered every page, write the
// fingerprinted assets the pages registered and the stylesheet (which needs
// the responsive rules for every image they drew), and point each page at it.
// In `astro dev` it serves the same files from memory.
import { mkdirSync, readdirSync, readFileSync, writeFileSync } from "node:fs";
import { dirname, join } from "node:path";
import { fileURLToPath } from "node:url";
import type { AstroIntegration } from "astro";
import { parse } from "./lib/buildmd.ts";
import { siteCSS } from "./lib/sitecss.ts";
import { CSS_TOKEN, existingState, resetState, type State } from "./lib/state.ts";

function phaseCount(st: State): number {
	return parse(readFileSync(join(st.root, "BUILD.md"), "utf8")).phases.length;
}

function htmlFiles(dir: string): string[] {
	return readdirSync(dir, { recursive: true, encoding: "utf8" })
		.filter((f) => f.endsWith(".html"))
		.map((f) => join(dir, f));
}

const types: Record<string, string> = {
	".css": "text/css; charset=utf-8",
	".png": "image/png",
	".woff": "font/woff",
};

export default function lorehallow(): AstroIntegration {
	return {
		name: "lorehallow",
		hooks: {
			"astro:build:start": () => {
				resetState();
			},
			"astro:server:setup": ({ server }) => {
				server.middlewares.use((req, res, next) => {
					const st = existingState();
					const path = (req.url ?? "").split("?")[0] ?? "";
					if (!st || !path.startsWith("/assets/")) return next();
					const body =
						path === CSS_TOKEN
							? new TextEncoder().encode(siteCSS(st, phaseCount(st)))
							: st.assets.files.get(path.slice(1));
					if (!body) return next();
					res.setHeader(
						"Content-Type",
						types[path.slice(path.lastIndexOf("."))] ?? "application/octet-stream",
					);
					res.setHeader("Cache-Control", "no-store");
					res.end(body);
				});
			},
			"astro:build:done": ({ dir, logger }) => {
				const st = existingState();
				if (!st) throw new Error("lorehallow: no page was rendered, so there are no assets to write");
				const out = fileURLToPath(dir);
				const cssURL = st.assets.add("site.css", new TextEncoder().encode(siteCSS(st, phaseCount(st))));
				for (const file of htmlFiles(out)) {
					const html = readFileSync(file, "utf8");
					writeFileSync(file, html.replaceAll(CSS_TOKEN, cssURL));
				}
				for (const [path, data] of st.assets.files) {
					const file = join(out, path);
					mkdirSync(dirname(file), { recursive: true });
					writeFileSync(file, data);
				}
				logger.info(`wrote ${st.assets.files.size} assets; stylesheet ${cssURL}`);
			},
		},
	};
}
