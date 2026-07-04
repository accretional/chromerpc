# Bidi / agent-steering lessons (preserved from deleted narrative logs)

- Date: 2026-07-03
- Author: Claude (Opus 4.8) — initial session
- Status: reference
- Related: [[0005-baseline-test-triage]], [[0003-testing-strategy]] (Phase C4/C5)

The `clown-nose-test/` and `redsocks-test/` directories were narrative agent
shopping logs, not executable tests — deleted in the junk sweep. Before deleting
them, these durable lessons were extracted. They matter for the Phase-1 chrome-proxy
meta-test and the multi-agent dynamic-navigation suite, and as regression behaviors
the new tests should encode.

## Lesson 1 — a long-lived bidi session needs gRPC keepalive

- **Symptom:** after a few moves, the next call returned `send: EOF`. The bidi
  stream had **idle-closed during agent/operator think-time** (gaps between moves).
  The server saw the stream close and recycled its Chrome. Not a navigation bug.
- **Fix (already in the tree):** `chrome-proxy` sets
  `keepalive.ClientParameters{Time: 30s, Timeout: 10s, PermitWithoutStream: true}`
  so the idle stream survives think-time gaps.
- **Encode as a test (Phase C4/C5):** open a bidi session, idle > 30s, then issue a
  step — it must still succeed (no EOF, no lost session state).

## Lesson 2 — pool recycle must only return a *ready* browser

- **Symptom:** every step returned `session: browser not ready`.
- **Root cause:** the old **synchronous-recycle** pool code, when a session
  disconnected, ran the recycle *throttled* (client gone → little CPU between
  requests), timed out, and returned a Chrome with a **nil browser** to the pool.
  The next session acquired that broken instance.
- **Fix:** **background-immediate recycle that only returns a Chrome to the pool
  after a *successful* relaunch** (retry-until-success) — a failed recycle never
  yields a "not ready" browser. (See `internal/server/interactive/` ChromePool.)
- **Encode as a test (Phase C3/C5):** after a session closes and the pool recycles,
  the next acquire must always get a ready browser (loop several acquire/release
  cycles under the pool service; zero "not ready").

## Lesson 3 — CAPTCHA / human-handoff: don't fully hand off; auto-resume

- **Pattern that worked:** when a page hit a CAPTCHA or needed a human (OTP, puzzle),
  the agent launched a **lightweight background monitor** — a detached loop polling
  `/steps` every ~8s with an `evaluate_script` that classifies the page
  (`OTP` / `MOBILE` / `VERIFY` / `CAPTCHA` / `PAGE:<title>`). While it returns
  `CAPTCHA` it keeps waiting; the instant it returns anything else it emits
  `CAPTCHA CLEARED -> state=<x>` and exits, re-invoking the agent.
- **Why it matters:** the agent **owns "detect finished + drive the next step"**
  rather than blocking on a manual ping. Directly relevant to the Phase-C5 agentic
  suite design — the same classify-and-resume loop is how a headless `claude` agent
  can drive dynamic navigation robustly past interstitials.

## Also worth knowing

- These sessions were driven through `chrome-proxy` against the deployed
  `chromerpc-bidi-pool` — the exact local-client → remote-instance topology the
  Phase-1 meta-test formalizes.
- Payment/PII steps were deliberately omitted in the logs; keep test fixtures free
  of real credentials.
