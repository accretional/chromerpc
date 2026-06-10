# Interactive pool — load test

Drives the multi-process interactive pool service (`chromerpc-bidi-pool`) under
concurrent load and checks Cloud Run health metrics afterward.

## What it does

`main.go` opens N concurrent bidi `Session` streams. Each runs the **same**
workflow, sending a step and **waiting for its response** before the next (a real
interactive client, not a batch):

1. set viewport
2. navigate `https://accretional.com` (wait for network idle)
3. screenshot
4. navigate `https://statue.dev`
5. screenshot
6. click
7. read (`document.title` / URL)

Sessions run in **pulses**: all N launched at once; the next pulse starts a
`-gap` after the *last finisher* of the previous one. Reports per-pulse
success/failure and session-latency percentiles (so a climb across pulses is
visible).

## Run

```bash
# Service must be deployed first (pool size 2):
POOL_SIZE=2 ./scripts/deploy-pool.sh

# Full run (20 concurrent x 3 pulses, 30s gap) + metrics:
./loadtest/run.sh

# Or directly:
go run ./loadtest -addr <host>:443 -tls \
  -token "$(gcloud auth print-identity-token)" \
  -concurrency 20 -pulses 3 -gap 30s
```

## Health checks (`check-metrics.sh`, MQL)

After the run it queries Cloud Run metrics for the service and prints:

| Signal | MQL metric | What good looks like |
|---|---|---|
| Instances over time | `container/instance_count` | Rises during pulses, falls back toward 0. A floor that keeps climbing ⇒ leaked/unhealthy instances (e.g. not cleaned up after clients disconnect). |
| Request latency p95 | `request_latencies` | Comparable across the 3 pulses — no drastic climb. |
| Container startups | `startup_latencies` (count) | Burst at pulse 1, near-zero after. Later spikes ⇒ instance deaths/restarts. |
| Requests by code class | `request_count` | Sanity on volume. Note: gRPC status rides inside HTTP 2xx, so the **loadtest client** is the source of truth for app-level failures. |

Because pool size = 2 and Cloud Run concurrency = 2, 20 concurrent sessions fan
out to ~10 instances (2 Chromes each). Each session gets a dedicated, isolated
Chrome process; on session close that Chrome is recycled (killed + relaunched)
and its per-process temp dir removed — the metrics confirm this doesn't leak
instances or memory across pulses.
