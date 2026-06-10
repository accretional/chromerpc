# Deploying chromerpc to Google Cloud Run

chromerpc ships as a self-contained container: a Go gRPC server plus a bundled
`google-chrome-stable`. The image does **not** depend on Chrome being present on
the build machine — Chrome is installed inside the image at build time. gRPC
server reflection is enabled, so `grpcurl` and other tools can discover the API
dynamically.

## Development workflow

Two loops — keep them separate:

- **Inner loop (fast, functional):** run the server natively against your local
  Chrome — no Docker, no emulation.
  ```bash
  make dev                                              # server on :50051
  make recipe RECIPE=recipes/search_and_screenshot.textproto
  ```
  Test changes here *before* deploying.
- **Deploy loop (amd64 image → Cloud Run):**
  ```bash
  make deploy          # Cloud Build (native amd64) — recommended on Apple Silicon
  make deploy-local    # local docker build (--platform linux/amd64; slow under
                       # qemu on arm64 hosts — see scripts/deploy-local-image.sh)
  ```
  On Apple Silicon, prefer `make deploy`: Cloud Run needs `linux/amd64`, and Cloud
  Build builds that natively, whereas a local arm64 host must emulate amd64 (slow
  for the Chrome layer).

## One command

```bash
gcloud auth login
gcloud config set project <YOUR_PROJECT>
./scripts/deploy-cloudrun.sh
```

The script enables the required APIs, creates an Artifact Registry repo, grants
the Cloud Build service account the roles it needs, then runs `cloudbuild.yaml`
which **builds → tags by Chrome version → pushes → deploys**. On success it
prints the service URL and a `grpcurl` smoke test.

Tunables (env vars): `PROJECT`, `REGION` (default `us-central1`), `REPO`,
`SERVICE`, `MAX_INSTANCES` (default `10`), `INVOKER_AUTH` (`allow` = public,
anything else = require an IAM invoker).

## Image tag = Chrome version

Each build reads the *installed* Chrome version out of the freshly built image
and tags with it, e.g.:

```
us-central1-docker.pkg.dev/<project>/chromerpc/chromerpc:126.0.6478.126
us-central1-docker.pkg.dev/<project>/chromerpc/chromerpc:latest
```

Rebuilding picks up whatever `stable` Chrome is current (auto-update), while the
exact version stays visible 1:1 in the tag. The deployed Cloud Run revision
references the **version-pinned** tag (not `:latest`), and carries a
`chrome-version` label.

## Cloud Run configuration (and why)

| Setting | Value | Reason |
|---|---|---|
| `--use-http2` | on | gRPC needs end-to-end HTTP/2 to the container. |
| `--port` | 8080 | Cloud Run injects `$PORT`; the server listens on it. |
| `--min-instances` | 0 | Scale to zero — no idle cost. Cold starts launch Chrome (~seconds). |
| `--concurrency` | 8 | Multiple requests per instance; each runs in its own isolated browser context (see below). |
| `--cpu / --memory` | 4 / 4Gi | Headless Chrome needs real CPU + RAM; sized for several concurrent contexts. |
| `--cpu-boost` | on | Faster cold-start Chrome launch. |
| `--no-sandbox`, `--disable-dev-shm-usage` | on (in CMD) | Cloud Run can't grant `SYS_ADMIN` or a large `/dev/shm`; Chrome would otherwise crash. |

## Session isolation

`HeadlessBrowserService` (`RunAutomation` / `ExecuteStep`) now gives **each call
its own isolated incognito `BrowserContext`** — a fresh target + session with
its own cookies, `localStorage`, cache, and tabs — which is created at the start
of the call and disposed at the end. This means:

- **Concurrent calls** on the same instance don't see each other's cookies/
  storage/tabs, so `--concurrency` can safely be > 1 (default 8).
- **Consecutive calls** start clean — no state leaks from a previous call on a
  warm instance.
- `open_tab` / `switch_tab` / `download_file` operate inside the call's own
  context and are cleaned up automatically on disposal.

This is best-effort isolation: all calls still share one Chrome **process**, so
process-global concerns (a browser crash, OS-level resources, `/tmp`) are shared.
For hard multi-tenant isolation, run separate instances/services. If the Chrome
build doesn't support browser contexts, the service degrades gracefully to the
shared default session and logs a warning.

**Client contract:** still treat every call as self-contained — don't rely on
state persisting **across** calls (scale-to-zero may route you to a cold or
different instance). The high-level `RunAutomation` RPC, which runs a whole
navigate/act/screenshot sequence in one call, is the recommended entry point.

> The **low-level per-domain services** (Page, Runtime, Network, …) still share
> the process-wide default session and are **not** per-call isolated. Use
> `RunAutomation` for concurrent/isolated workloads.

## Security note

`INVOKER_AUTH=allow` (the default) makes the service **publicly reachable**. A
public browser-automation endpoint is effectively an open fetch/SSRF proxy that
can navigate to internal or arbitrary URLs. For anything beyond a demo, deploy
with `INVOKER_AUTH=require` and call it with an identity token:

```bash
INVOKER_AUTH=require ./scripts/deploy-cloudrun.sh
# grant a caller:
gcloud run services add-iam-policy-binding chromerpc --region us-central1 \
  --member 'user:someone@example.com' --role roles/run.invoker
# call it:
grpcurl -H "authorization: Bearer $(gcloud auth print-identity-token)" \
  <service-host>:443 list
```

## Calling the service

The Cloud Run URL terminates TLS at the edge; clients speak gRPC over TLS/HTTP2
on port 443:

```bash
HOST=chromerpc-xxxxx-uc.a.run.app
grpcurl ${HOST}:443 list
grpcurl ${HOST}:443 describe cdp.headlessbrowser.HeadlessBrowserService
```

## Local / VM parity

Locally the server still defaults to `:50051` when `$PORT` is unset. For a plain
VM (e.g. a c4d instance) you can run the same image directly:

```bash
docker run --rm -p 50051:50051 \
  <image> --headless --no-sandbox --disable-dev-shm-usage --addr :50051
```
