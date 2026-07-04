# Refactor Notes — conventions

This directory is the **shared working memory** for the chromerpc refactor, written
to by every agent/Claude instance and every human working on the project. If you
did investigative work, made a decision, hit a wall, or learned something
non-obvious, write it here so the next instance doesn't repeat it.

## How to write a note

- One file per topic: `NNNN-short-slug.md` (zero-padded, monotonically increasing).
- Start each note with a header block:

  ```
  # <Title>
  - Date: YYYY-MM-DD
  - Author: <agent/human> (session/agent id if useful)
  - Status: draft | active | superseded-by NNNN | done
  - Related: [[NNNN-other-note]], TODO#<id>, BUGS#<id>
  ```

- Prefer durable facts (`file:line`, commands that worked, why a path was chosen)
  over narration. Record dead ends too — "tried X, failed because Y" saves hours.
- When a note is obsoleted, don't delete it: set `Status: superseded-by NNNN` and
  link forward.

## Index

| Note | Topic |
|------|-------|
| [0001-kickoff](0001-kickoff.md) | Refactor kickoff: scope, phases, decisions, open questions |
| [0002-environment-setup](0002-environment-setup.md) | Getting chromerpc running locally (Chrome install, build, verify) + headless bot-detection finding |
| [0003-testing-strategy](0003-testing-strategy.md) | Design of the single end-to-end test harness + chrome-proxy meta-test |
| [0004-accretional-repos](0004-accretional-repos.md) | Ecosystem repos (proto-linker=gluon?, proto-merge, cdp-agent) relevant to later phases |
| [0005-baseline-test-triage](0005-baseline-test-triage.md) | First Phase-A run (1 failure: dead Database domain), file provenance, cleanup candidates |
| [0006-bidi-agent-lessons](0006-bidi-agent-lessons.md) | Preserved lessons (keepalive, pool-recycle, CAPTCHA auto-resume) from the deleted narrative logs |

## Where things live

- `../README.md` — overall project status & roadmap (the dashboard).
- `../TODO.md` — task-level tracking (the backlog / in-flight work).
- `../BUGS.md` — known bugs & issues to address now or later.
- `./NNNN-*.md` — working notes (this dir).
