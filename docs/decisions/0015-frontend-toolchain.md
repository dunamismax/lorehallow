# 0015: The frontend toolchain

- **Status:** Accepted
- **Date:** 2026-11-07
- **BUILD.md:** 1.2

## Context

[ADR 0014](0014-astro-website-and-phaser-browser-client.md) moves lorehallow.com to Astro and the game to Phaser, both in TypeScript, and asks for one Node toolchain that covers both: one Node LTS, one package manager, and one set of TypeScript, lint, format and test tools, pinned in `toolchain.env` and checked by `make doctor`. Node runs on developer machines and in CI only. The host never runs it: Caddy serves the built files.

The project's rule for dependencies is "few and justified", and every one of them is a supply-chain risk on a public repository.

## Decision

| Tool | Version | Where it's pinned |
|---|---|---|
| Node.js | 26.13.0 (the Active LTS line, "Lithium") | `toolchain.env` (`NODE_VERSION`), the root `package.json` `engines`, checked by `make doctor` |
| npm | 11.20.0 (the npm that ships with Node 26.13.0) | `toolchain.env` (`NPM_VERSION`), `packageManager` and `engines` |
| TypeScript | 6.0.3 | `package-lock.json` |
| Biome | 2.6.6, lint and format | `package-lock.json` |
| Astro | 7.4.7, with its built-in Vite | `package-lock.json` (`web/`) |
| `@astrojs/check` | 0.9.12, type-checks `.astro` files | `package-lock.json` (`web/`) |
| Tests | `node --test`, built into Node | Node |

- **npm workspaces with one committed lockfile.** The root `package.json` is the workspace; `web/` is its first member and `client/` joins in BUILD.md 2.6. `.npmrc` sets `save-exact` (every version in `package.json` is exact) and `engine-strict` (npm refuses to run under the wrong Node or npm). CI installs with `npm ci`, which installs exactly the lockfile.
- **TypeScript is run directly by Node.** Node 26 strips types natively, so the BUILD.md parser, the linter command and the tests are `.ts` files that `node` runs with no build step. They use only erasable syntax (`erasableSyntaxOnly`: no enums, namespaces or parameter properties) and import each other with `.ts` extensions, which Astro's Vite also understands. `tsc --noEmit` checks them all with `strict` on.
- **TypeScript 6, not 7.** TypeScript 7.0 (the native compiler) is out, but `@astrojs/check` supports TypeScript 5 and 6 only. Move to 7 when Astro's checker does.
- **Biome for lint and format.** One package with no dependencies of its own replaces ESLint, typescript-eslint, Prettier and their Astro plugins. It checks `.ts`, `.json` and the frontmatter of `.astro` files. `astro check` covers the types in `.astro` templates. Generated code (protobuf bindings, from 2.6) is excluded.
- **`node --test` for unit and output tests.** Node's built-in runner and `node:assert` cover what the project needs (tables of cases, fixtures, snapshots by hand), with nothing to install.
- **`npm audit` runs in `make web-lint`** and therefore in CI, failing on any known vulnerability of moderate severity or worse.
- **Install scripts stay off.** npm 11.20 doesn't run dependencies' install scripts unless they are approved. None are needed: Astro's optional `fsevents` (macOS file watching) works without its script.

## Alternatives considered

- **pnpm or Bun.** pnpm's strict `node_modules` and Bun's speed are real, but both are another tool to install and pin, and npm ships with Node. npm workspaces do everything two packages need.
- **ESLint and Prettier.** The usual pair, with the Astro and TypeScript plugins, is five or more packages and their dependency trees. Biome does both jobs in one.
- **Vitest.** Good, and it shares Vite with Astro and Phaser, but `node --test` needs no install and runs the same TypeScript files Node runs. Revisit for the Phaser client if browser-like test environments are needed; the headless-browser checks in 2.6 use a real browser instead.
- **Node 24.** Still maintained, but 26 is the current Active LTS, supported until 2029, and what the machines here already run.

## Consequences

- `make doctor` checks Node and npm; `make setup` runs `npm ci`.
- `make check` gains the frontend checks: Biome, `astro check`, `tsc --noEmit`, the unit and output tests, and `npm audit`.
- A Node, npm or package bump changes `toolchain.env`, `package.json`, the lockfile and AGENTS.md section 13 in one commit.
- TypeScript 7 is a planned upgrade, blocked on `@astrojs/check`.
