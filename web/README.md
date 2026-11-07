# web/

The lorehallow.com website. It is built in Phase 1 as an Astro static site that parses [BUILD.md](../BUILD.md) to render the live progress tracker.

Rules: TypeScript strict, colors from the shared palette tokens, no third-party requests, pixel art at integer scales only. See [AGENTS.md section 7](../AGENTS.md#web-conventions-lorehallowcom).

| Path | What |
|---|---|
| `src/lib/buildmd/parse.ts` | The BUILD.md parser. It implements the format contract at the top of BUILD.md: phases, metadata, sections, items, tags and checked state, plus every format problem with its line number. |
| `src/lib/buildmd/lint.ts` | `make buildmd-lint`: fails on any format problem. Part of `make check`. |
| `src/lib/buildmd/parse.test.ts` | Parser tests, including malformed input and the real BUILD.md. |

The parser is plain TypeScript that Node runs directly (type stripping): no enums, no parameter properties, and imports end in `.ts`. Change the parser, its tests and the contract in BUILD.md together.

```bash
make web-lint      # strict typecheck (tsc)
make web-test      # node --test
make buildmd-lint  # lint BUILD.md
```
