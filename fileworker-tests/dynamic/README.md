# Dynamic Fileworker navigation tests

This harness asks a non-interactive Claude Code agent (`claude -p`) to inspect
and drive a **real, already-running** browser through `chrome-proxy`. Unlike a
fixed assertion script, the agent repeatedly observes DOM state and screenshots,
chooses its next interaction, and reports usability defects it encounters.

The harness deliberately does not start Fileworker, chromerpc, or chrome-proxy.
Keeping process ownership outside the agent makes the test reproducible and
prevents it from silently substituting a mock application or browser.

## Prerequisites

1. Serve Fileworker on HTTP (for example `http://127.0.0.1:8765/`).
2. Start an interactive chromerpc and a chrome-proxy as documented in
   [`../../chrome-proxy/README.md`](../../chrome-proxy/README.md).
3. Confirm `curl http://127.0.0.1:8099/health` succeeds.
4. Be authenticated in the `claude` CLI.

Then run:

```bash
fileworker-tests/dynamic/run.sh \
  --proxy http://127.0.0.1:8099 \
  --app http://127.0.0.1:8765/
```

Artifacts are placed in `fileworker-tests/dynamic/artifacts/<UTC timestamp>/`:

- `prompt.txt`: exact instructions given to the agent
- `claude.stdout.json`: Claude Code's machine-readable response envelope
- `claude.stderr.log`: CLI diagnostics
- `result.json`: schema-validated agent verdict, when available
- `run.json`: harness timings, exit state, and artifact locations

Proxy screenshots remain in the proxy's configured `-shots` directory and their
paths are cited in `result.json`.

## Safety and bounds

- The agent receives only `Bash(curl *)` and `Read`; it cannot edit the project.
- It is told to use `curl` only against the loopback proxy.
- The CLI gets a configurable dollar ceiling (default `$1.00`).
- A parent-process deadline (default 8 minutes) kills the complete Claude
  process group on timeout.
- The target application and proxy must both use loopback HTTP URLs by default.
  `--allow-non-loopback` is an explicit escape hatch.
- Each run uses no persistent Claude session and has a strict JSON result schema.

Useful overrides:

```bash
run.sh --timeout 600 --max-budget-usd 1.50 --model sonnet
run.sh --scenario prompt.md
```

Run the harness's deterministic tests (no Claude credentials or Chrome needed):

```bash
python3 -m unittest fileworker-tests/dynamic/test_runner.py
```

