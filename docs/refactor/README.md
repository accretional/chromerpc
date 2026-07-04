# chromerpc Refactor — Project Status

Coordination hub for the major refactor of chromerpc. This is the **dashboard**;
detail lives in the linked docs.

- **[TODO.md](TODO.md)** — task-level tracking (backlog + in-flight).
- **[BUGS.md](BUGS.md)** — known bugs & issues (now or later).
- **[notes/](notes/)** — shared working notes across all agent/human instances
  ([conventions](notes/README.md)).

> Update this file whenever a phase changes state. Keep it short — it's a map,
> not a log. Logs go in `notes/`.

## Current phase

**Phase 1 — Testing strategy: ✅ COMPLETE (harness green end-to-end, 2026-07-03).**
The single gate harness works across all phases against a live deployment. Next up:
**Phase 2 — docs sweep.**

What's green (see [notes/0007](notes/0007-first-cloud-run.md)):
- **Phase A (local):** cdpclient unit tests + all 54 CDP domains integration suite +
  raw-CDP smoke + fixture screenshot. `scripts/refactor-e2e.sh --local-only`.
- **Phase B (build+deploy):** Cloud Build → 2 services on Cloud Run
  (`chromerpc` + unified `chromerpc-interactive`, min-instances 0).
- **Phase C (agentic+automation):** C1 smoke ✅, C2 recipes 2/2 ✅, C3 loadtest 8/8 ✅,
  C4 chrome-proxy meta-test ✅ (functional + visual), C5 agentic 2/2 ✅.

Harness pieces: `scripts/refactor-setup.sh`, `scripts/refactor-e2e.sh`,
`scripts/metatest-chrome-proxy.sh`, `scripts/agentic-suite.sh`, `scripts/fixtures/`.
Also this phase: fixed BUG-002 (dead WebSQL `Database` domain → 54 domains), swept
scratch junk, and collapsed to the 2-service model.

## Why we're doing this

chromerpc grew organically into: ~55 CDP domain gRPC services, a custom
high-level `HeadlessBrowserService` automation API, a bidi/interactive session
service, a `chrome-proxy` steering UI, and a Cloud Run deployment. The refactor
re-bases the project on a cleaner architecture and removes accumulated cruft. The
guiding constraint: **we must be able to prove the whole system still works after
each step**, locally and deployed — hence testing comes first.

## Roadmap

Ordered. Each phase should be green on the test harness before the next begins.

| # | Phase | Status |
|---|-------|--------|
| 0 | **Coordination & tracking** — stand up `docs/refactor/` structure & note conventions | ✅ done |
| 1 | **Testing strategy** — single end-to-end harness (local smoke → build+deploy → agentic+automation suite) + chrome-proxy meta-test | ✅ done (green end-to-end) |
| 2 | **Docs sweep** — audit all existing docs; preserve useful resources/techniques, remove the rest | ⬜ not started |
| 3 | **CDP proto sweep + generation** — audit domains for add/remove; set up tracking + auto-pull + auto-generation from CDP PDL changes | ⬜ not started |
| 4 | **Single CDP server via proto-linker** — remove custom `HeadlessBrowserService`; use external proto-linker/build project to combine all CDP gRPC services into one low-level CDP server | ⬜ not started |
| 5 | **chromescript** — new automation implementation under `proto/chromescript/` (replaces the old step-based automation) | ⬜ not started |
| 6 | **chromerun** — move bidi/interactive navigation to `proto/chromerun/` | ⬜ not started |
| 7 | **Deps + tooling cleanup** — update all deps; simplify/remove unneeded tools | ⬜ not started |
| 8 | **Make it all work** — end-to-end green on the harness | ⬜ not started |
| 9 | **Build/test/docs strategy v2** — incl. fuzzing; execute; clean up whole repo | ⬜ not started |
| 10 | **Agents-in-Chrome** — follow-on integration projects (scoped later) | ⬜ future |

## Open questions blocking Phase 1

See [notes/0001-kickoff.md](notes/0001-kickoff.md#open-questions). Summary:

1. GCP target for the deploy step (project, auth, region, service names) — needed
   to make the harness's build+deploy leg actually run.
2. Concrete meaning of "multi-agent workflow / agentic suite" — how the harness
   invokes agents to drive chrome-proxy dynamic navigation.
3. Identity/URL of the external "proto-linker" build project (Phase 4; capture now).
