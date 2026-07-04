# Baseline test run + file-provenance triage

- Date: 2026-07-03
- Author: Claude (Opus 4.8) — initial session
- Status: active
- Related: [[0001-kickoff]], [[0003-testing-strategy]], BUG-002, ../TODO.md

First run of the Phase-A gate (`scripts/refactor-e2e.sh --local-only`) plus a git
provenance sweep, to establish the pre-refactor baseline and separate real junk from
keepers.

## Baseline test result (Chrome 150)

- **A1** `internal/cdpclient` unit tests — **PASS** (hermetic).
- **A2** Chrome guard — **PASS**.
- **A3** full `internal/integration` suite (~all CDP domains + raw-CDP smoke) —
  **1 failure, everything else green**:
  - `TestDatabaseEnableDisable` (`domains6_test.go:218`) → `Database.enable: CDP
    error -32601: 'Database.enable' wasn't found`. The **WebSQL `Database` domain was
    removed from Chrome** → BUG-002. Dead upstream surface, not a code bug.
- **A4** never reached (A3 is a hard gate and aborted).

**Conclusion:** the test suite is healthy — a solid regression backbone. Only one
test fails and it's because an entire CDP domain vanished from Chrome.

## File provenance — when things were written

Histogram of file **creation** dates (git `--diff-filter=A`):

| Date | Files | What landed |
|------|------:|-------------|
| 2026-01-15 | 2 | `LICENSE`, `README.md` (only survivors of "the very beginning") |
| 2026-03-12 | 239 | the **entire real codebase** — server, all CDP domains, cdpclient, **all tests** |
| 2026-03-17 | 6 | `headlessbrowser` custom automation API + `cmd/automate` + `automations/` |
| 2026-05-11 | 5 | `chrome-testing/` (snap.sh + demo) |
| 2026-06-09 | 14 | deploy scripts, `cloudbuild.yaml`, `DEPLOY.md`, `recipes/`, `recipe2json` |
| 2026-06-10 | 13 | bidi `interactive` service, `chrome-proxy`, `loadtest/`, `isession`, the two `*-test/` narrative logs |

**Key facts:**
- "The very beginning" (2026-01-15) left only `LICENSE` + `README.md`. The first
  "chromerpc cli" commit just added **6 aspirational lines** to the README — there
  was no early code that lingers.
- **All tests were written 2026-03-12** in the initial code drop. They are **not**
  old junk — keep them (they're the regression backbone), minus the dead Database one.

## Cleanup candidates (the actual junk)

Not "from the very beginning," but stale / scratch / dead. Ordered by confidence:

1. **`clown-nose-test/CLAUDE_CLOWN_NOSE_TEST.md`** + **`redsocks-test/CLAUDE_RED_SOCKS_TEST.md`**
   (2026-06-10) — narrative agent shopping logs, **not executable tests**. Each
   documents one real bug+fix worth preserving as a note (bidi keepalive;
   pool-recycle must return a ready browser — already captured in the exploration).
   → **Extract the two lessons into a note, then delete both dirs.** ("redsocks" =
   leftover from transparent-proxy experimentation; only a stray .md remains.)
2. **`automations/screenshot_site.textproto`** (2026-03-17) — hardcoded personal path
   (`/Users/fred…`) + `localhost:3000`; personal scratch. → **delete.**
3. **`Database` domain** — `proto/cdp/database/*`, `internal/server/database/database.go`,
   its `main.go` registration, the Makefile `proto` entry, `TestDatabaseEnableDisable`
   (BUG-002). Upstream-dead. → **delete** (Phase 3 proto sweep; can pull forward to
   unblock A3).
4. `headlessbrowser` custom API + `cmd/automate` + `cmd/recipe2json` + `recipes/`
   (2026-03-17/06-09) — **not junk**, but scheduled for removal in Phase 4/5. Leave
   until then.

## Decision needed — A3 gate vs the dead Database domain

A3 runs the *full* suite as a hard gate, so BUG-002 blocks green. Options:
- **(A) Pull the Database removal forward now** (delete the dead domain + test). Small,
  safe, legitimate Phase-3 cleanup; makes A3 green immediately. **Recommended.**
- (B) Temporarily exclude `TestDatabaseEnableDisable` from the A3 gate and report it
  as a known-failure until Phase 3.

See TODO §1.2 / §Cleanup.
