// lorehallow.com: a static Astro build with no browser JavaScript (ADR 0014).
// Every URL keeps the shape the Go-built site gave it (docs/design/website-parity.md).
import { satteri } from "@astrojs/markdown-satteri";
import { defineConfig } from "astro/config";
import lorehallow from "./src/integration.ts";
import { dropRawHTML } from "./src/lib/markdown.ts";

export default defineConfig({
	site: "https://lorehallow.com",
	output: "static",
	// Its own folder until the cutover, so both builds can be compared (scripts/parity.ts).
	outDir: process.env.LOREHALLOW_ASTRO_OUT ?? "./dist-astro",
	publicDir: "./public",
	trailingSlash: "always",
	build: {
		format: "directory",
		// The CSP is style-src 'self': never inline a stylesheet.
		inlineStylesheets: "never",
		assets: "_astro",
	},
	// Anything that would add a script to a page stays off.
	prefetch: false,
	devToolbar: { enabled: false },
	compressHTML: false,
	markdown: {
		processor: satteri({ mdastPlugins: [dropRawHTML] }),
	},
	integrations: [lorehallow()],
	server: { host: "127.0.0.1", port: 4321 },
});
