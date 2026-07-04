# CDP upstream — vendored protocol definitions (Phase-3 tracking)

Pinned Chrome DevTools Protocol definitions, used to track drift and to generate
`.proto` (see `tools/cdpgen` and `docs/refactor/notes/0008-phase3-cdp-diff.md`).

| File | Source | Committed? |
|------|--------|-----------|
| `chrome-protocol.json` | the **exact** protocol of the Chrome build we run, dumped from its live `/json/protocol` endpoint | **yes** — the authoritative baseline |
| `VERSION` | that Chrome's version string (e.g. `Chrome/150.0.7871.47`) | yes |
| `browser_protocol.json` | `ChromeDevTools/devtools-protocol@master` (tip-of-tree) | no (gitignored; regenerable, churns) |
| `js_protocol.json` | `ChromeDevTools/devtools-protocol@master` (tip-of-tree) | no (gitignored) |

## Refresh + re-diff

```bash
scripts/cdp-pull.sh          # launches Chrome, re-dumps /json/protocol, refetches
                             # master, and prints a domains/commands diff vs the repo
```

`chrome-protocol.json` is ground truth for **correctness** (what the Chrome we ship
actually supports). The master files are for **anticipating** upstream changes
(tip-of-tree runs ahead of stable Chrome). Commit a refreshed `chrome-protocol.json`
+ `VERSION` whenever the bundled Chrome version bumps.

## Generate .proto from it

```bash
go run ./tools/cdpgen -proto proto/cdp/_upstream/chrome-protocol.json -domain Page -out /tmp
go run ./tools/cdpgen -proto proto/cdp/_upstream/chrome-protocol.json -domain all -out /tmp/gen
```
