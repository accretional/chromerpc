# Known Bugs & Issues

Tracked defects and rough edges — things to fix now or keep in view for later.
Add entries as they're found; link to the note or TODO that resolves them.

Format: `### BUG-NNN <title>` with severity, status, and a repro/why.

---

### BUG-001 Headless Chrome trips Google bot-detection (reCAPTCHA "unusual traffic")
- **Severity:** medium (affects test/automation realism, not the RPC path)
- **Status:** open — known, has a workaround
- **Found:** 2026-07-03, during initial environment verification (see
  [notes/0002-environment-setup.md](notes/0002-environment-setup.md)).
- **Symptom:** Navigating a fresh **headless** chromerpc-launched Chrome to
  `https://www.google.com/search?q=css` returns Google's "Our systems have
  detected unusual traffic… I'm not a robot" reCAPTCHA interstitial instead of
  results. The gRPC navigate+screenshot path works correctly; Google is blocking
  the client.
- **Workaround that worked:** launch a real **non-headless** Chrome with
  `--remote-debugging-port=9222` and connect chromerpc via `--ws-url`; the real
  browser passes and returns genuine results.
- **Why it matters for the refactor:** the automation-suite test cases must not
  depend on bot-gated sites in headless mode, or must use the connect-to-real-Chrome
  path / a controlled fixture site. Decide fixture strategy in Phase 1.
- **Note:** the server already passes `--disable-blink-features=AutomationControlled`
  by default (README), which is insufficient against Google in headless mode.

---

### BUG-002 `Database` CDP domain removed from Chrome — integration test fails
- **Severity:** medium (breaks the local test gate; stale dead surface)
- **Status:** open — **root cause identified**
- **Found:** 2026-07-03, first run of `scripts/refactor-e2e.sh --local-only` (Phase A3).
- **Symptom:** `TestDatabaseEnableDisable` (`internal/integration/domains6_test.go:218`)
  fails: `Database.enable: CDP error -32601: 'Database.enable' wasn't found`. This is
  the **only** failing test in the whole integration suite on Chrome 150 (everything
  else passes).
- **Root cause:** the CDP `Database` domain was the **WebSQL** interface; WebSQL was
  removed from Chrome, so the domain no longer exists in Chrome 150. `Database.enable`
  returns method-not-found.
- **Scope of dead surface:** `proto/cdp/database/database.proto`,
  `internal/server/database/database.go`, its registration in
  `cmd/chromerpc/main.go`, the Makefile `proto` list entry, and
  `TestDatabaseEnableDisable`. All created 2026-03-12 (commit `f34df1b`), never a
  "beginning" file — but now upstream-dead.
- **Fix (Phase 3, proto sweep):** remove the `Database` domain entirely. This is the
  canonical example motivating the **CDP PDL diff / auto-generation** work — a domain
  that vanished upstream since setup.
- **Harness implication:** A3 currently runs the *full* integration suite as a hard
  gate, so this one dead domain blocks the gate. Interim options: (a) remove Database
  now, or (b) scope A3's hard gate to a stable core and report the full suite as
  collected. See [notes/0005-baseline-test-triage.md](notes/0005-baseline-test-triage.md).

### BUG-003 (reserved)

_Bugs discovered during exploration will be appended here._
